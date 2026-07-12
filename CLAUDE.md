# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## O projeto

TaskFrame é um gerenciador de tarefas de terminal inspirado no taskwarrior, porém mais simples, com TUI estilo Norton Commander (dois painéis, bordas duplas `╔═╗`, barra de F-keys, cores ANSI 16 indexadas). Stack decidida com o usuário: **Go + Bubble Tea + SQLite puro-Go** (`modernc.org/sqlite`, sem CGo — importante no Windows). Interface híbrida: TUI principal + CLI de captura rápida. O plano original completo está em `C:\Users\victo\.claude\plans\quero-criar-um-app-piped-castle.md`.

## Comandos

```powershell
go build -o taskframe.exe ./cmd/taskframe   # build
go test ./...                                # todos os testes
go test ./internal/store/ -run TestCompleteAndUndo -v   # um teste específico
go test ./internal/tui/ -v                   # smoke tests da TUI (dirigem o modelo Bubble Tea)
gofmt -w . ; go vet ./...                    # formato + lint antes de finalizar
```

Rodar: `.\taskframe.exe` abre a TUI (precisa de terminal real; use Windows Terminal). Com args vira CLI: `add`, `list`, `done`, `del`, `note`, `undo`, `purge`. Para testar manualmente sem sujar o banco real: `$env:TASKFRAME_DB = "$env:TEMP\tf.db"`.

## Arquitetura

Camadas com dependência estrita em um só sentido — violar isso é regressão:

- `internal/task` — domínio puro (Task, Note, Activity, Filter, urgency, parser de datas). **Não importa nada do projeto.**
- `internal/store` — SQLite. Importa só `task`. Todo SQL escrito à mão; sem ORM.
- `internal/cli` e `internal/tui` — importam `task` + `store`. **A TUI nunca toca `database/sql`**: chama métodos do store dentro de closures `tea.Cmd` e recebe o resultado como mensagem (`tasksLoadedMsg` etc., ver `tui/msgs.go`).
- `cmd/taskframe/main.go` — sem args → TUI; com args → CLI. ~40 linhas.

### Decisões de modelagem (não mudar sem motivo forte)

- **Projetos não são tabela**: string pontuada (`casa.mercado`) na coluna `tasks.project`; a árvore da sidebar é derivada de `ProjectCounts()` em tempo de leitura. Filtro por projeto é prefix-match (`= ?` OR `LIKE ?.%`).
- **Subtarefas**: adjacency list (`parent_id`), nesting arbitrário. `store.BuildTree()` monta a árvore em memória a partir da lista flat — sem CTEs recursivas.
- **Urgency nunca é armazenada** — calculada em `task.Urgency()` na leitura, para a fórmula poder mudar sem migration. Coeficientes em `DefaultCoefficients` (urgency.go).
- **Audit log + undo**: toda mutação grava linhas na tabela `activity` dentro da MESMA transação, agrupadas por `op_id` (hex aleatório). `store.Undo()` acha o último op não-desfeito (ops desfeitos são marcados por uma linha `kind='undo'` cujo `field` guarda o op_id revertido) e reverte linha a linha. Undo de um `done` com recorrência reverte o done E apaga a instância clonada — porque ambos compartilham o op_id. **Ao adicionar qualquer mutação nova, registre em `activity` no mesmo tx e implemente o caso reverso em `undoModify`/`Undo`.**
- **Exceção deliberada**: a tabela `settings` (migration 2; tema, ordenação) NÃO loga em `activity` — são preferências de máquina, e o undo nunca deve reverter uma troca de tema. Settings também ficam fora do export.
- **Export/import** (`store/export.go`): backup JSON completo com ids preservados verbatim (o log restaurado mantém o undo funcionando). Import só em banco vazio, em tx única com `PRAGMA defer_foreign_keys = ON` (após um Move, um filho pode referenciar pai com id maior).
- **Delete é soft** (`status='deleted'`) para o undo funcionar; `purge` faz o hard-delete.
- **Timestamps**: TEXT RFC3339 UTC (ordenável, legível no sqlite3). Helpers `fmtTime`/`parseTime` em store.go — sempre use-os.
- **Migrations**: `PRAGMA user_version` + slice ordenado em `migrate.go` (índice+1 == versão). schema.sql embutido via `go:embed` é a migration 1; alterações de schema entram como novas strings no slice, nunca editando a 1.
- **DB**: `%APPDATA%\taskframe\taskframe.db`, override por `TASKFRAME_DB`/`--db`. WAL + `busy_timeout` + `SetMaxOpenConns(1)` — a conexão única elimina contenção de write-lock; não aumentar. DSN usa `filepath.ToSlash` (backslash em DSN é bug clássico no Windows).

### TUI (Bubble Tea)

`tui/app.go` é o root model e único dono do estado: `filter task.Filter` atual, foco (sidebar/lista), e `modal Modal` (interface local; um modal por vez — Form, Confirm, NotePrompt, Detail, Help). Fluxo: tecla → modal ou painel → mensagem de intenção (`formSubmittedMsg`, `noteSubmittedMsg`...) → App persiste via `tea.Cmd` → `statusMsg` → **todo `statusMsg` dispara `reload()`** (recarrega tarefas + projetos). Sem mutação otimista.

- Visual inteiro em `tui/theme.go`: **4 temas nomeados** (`dark` padrão sem pintar fundo, `borland` navy truecolor, `green`/`amber` fósforo) definidos como structs `palette` → builder `NewTheme(name, ascii)`. Tecla `t` cicla e persiste em settings; precedência resolvida em `main.go` (`--theme` > `TASKFRAME_THEME` > setting > dark). Ao adicionar cor nova, adicione em TODAS as paletas. `drawBox()` desenha painéis com título embutido na borda (lipgloss v1 não tem border title). Flag `--ascii` troca para bordas simples (conhost legado) e compõe com qualquer tema.
- Ordenação: `task.SortMode` (string: urgency/due/created), tecla `o` cicla e persiste; `store.BuildTree(tasks, now, mode)`.
- Mover (F6/`m`): modal `move.go`; o App valida ciclos via `checkMoveCycle` antes de `UpdateTask` — sem isso, `BuildTree` some silenciosamente com o subgrafo cíclico.
- A lista de tarefas (`tasklist.go`) tem renderer de árvore próprio — `bubbles/table` não suporta indentação; não tentar migrar para ele.
- **Toda F-key tem alias em letra** (`F2`/`a`, `F8`/`x`...) porque terminais/IDEs engolem F-keys. Manter ao adicionar atalhos; documentar em `help.go` e no README.
- Textinputs usam `cursor.CursorStatic` — o blink gera ticks que travam os testes síncronos (e redraws desnecessários). Manter em inputs novos.

### Testes

`tui/smoke_test.go` dirige o modelo real sem terminal: `drive()`/`exec()` executam cada `tea.Cmd` sincronamente e realimentam as mensagens. Testes de TUI novos devem seguir esse padrão (abrir modal, digitar com `typeText`, conferir o efeito no store em memória via `store.OpenMemory()`). `stripANSI` + `strings.Contains` para asserções de layout.

### CLI

Stdlib `flag` + switch manual — **sem cobra** (decisão deliberada; os tokens estilo taskwarrior `pro:x +tag due:sex prio:H` não são flags e têm parser próprio em `cli.parseTokens` + `task.ParseDate`). Saída do `list` é texto puro sem ANSI (pipe-friendly). Mensagens da CLI/TUI em português; código e comentários em inglês.

## Estado atual (2026-07-12) e retomada

**v1**: núcleo completo (tarefas, subtarefas, projetos hierárquicos, notas, activity log, filtros/busca) + urgency sort, undo, recorrência, soft-delete/purge, datas em linguagem natural (pt e en: `sex`, `fri`, `3d`, `eom`...).

**v2** (mesma data): sistema de 4 temas com persistência (motivado por "as cores doem os olhos" — o antigo fundo azul ANSI-4 foi substituído pelo `dark` como padrão); filtros virtuais Hoje/Atrasadas/Semana/Aguardando e seção de tags na sidebar; campos wait/scheduled no form e no detail; toggle de ordenação (`o`); diálogo Mover (F6/`m`) com validação de ciclo; export/import JSON; git + GitHub Actions CI (vet+test em ubuntu/windows, gofmt no ubuntu; `.gitattributes` força LF em `.go` para o check de gofmt não quebrar com autocrlf).

Testes: ~25 casos; smoke tests da TUI cobrem tema, sort, filtros virtuais, move (com undo) e form com wait. CLI e TUI verificadas fim-a-fim.

Ideias futuras (nada pendente do escopo aprovado):
- contexts (filtro default salvo — a tabela settings já existe)
- redo (undo do undo — o activity log suporta)
- FTS5 para busca (só se a busca LIKE ficar lenta — improvável em escala pessoal)
- lembretes/notificações (fora do escopo atual)
