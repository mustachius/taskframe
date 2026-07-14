# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## O projeto

TaskFrame é um gerenciador de tarefas de terminal inspirado no taskwarrior, porém mais simples. Stack: **Go + Bubble Tea + SQLite puro-Go** (`modernc.org/sqlite`, sem CGo — importante no Windows). Três formas de uso sobre o mesmo core: **REPL inline estilo Claude Code** (padrão), **TUI clássica Norton Commander** (`taskframe classic`), e **CLI de captura rápida** (`taskframe <subcomando>`). O plano completo está em `C:\Users\victo\.claude\plans\quero-criar-um-app-piped-castle.md`.

## Comandos

```powershell
go build -o taskframe.exe ./cmd/taskframe   # build
go test ./...                                # todos os testes
go test ./internal/repl/ -run TestAddAndList -v         # um teste específico
go test ./internal/repl/ -v                  # smoke tests do REPL (dirigem o modelo Bubble Tea)
gofmt -w . ; go vet ./...                    # formato + lint antes de finalizar
.\install.ps1                                # instala como comando global do usuário
```

Rodar: `.\taskframe.exe` abre o REPL; `taskframe classic` abre a TUI de dois painéis (ambos precisam de terminal real — Windows Terminal). Com args vira CLI: `add`, `list`, `done`, `del`, `note`, `undo`, `purge`, `export`, `import`. Para testar sem sujar o banco real: `$env:TASKFRAME_DB = "$env:TEMP\tf.db"`. **Não dá para dirigir o REPL/TUI por pipe no stdin** (o Bubble Tea lê eventos de console no Windows) — use os smoke tests para validar comportamento interativo.

## Arquitetura

Camadas com dependência estrita em um só sentido — violar isso é regressão:

- `internal/task` — domínio puro (Task, Note, Activity, Filter, urgency, datas, **`ParseTokens`** — parser dos tokens `pro:`/`+tag`/`due:` etc., compartilhado por CLI e REPL). **Não importa nada do projeto.** Strings de domínio/erro sempre em **inglês** (não passam pelo i18n).
- `internal/store` — SQLite. Importa só `task`. Todo SQL escrito à mão; sem ORM. `CheckMoveCycle` valida hierarquia (usado por REPL e TUI).
- `internal/i18n` — catálogo de mensagens da camada de apresentação (pacote leaf, **não importa nada**). Inglês é a fonte canônica (default), pt-br é a tradução. `Lang.T(key)`/`Lang.Tf(key, args...)` sobre `catalog` (`map[string][2]string`, `[0]=en`, `[1]=pt-br`). Usado por `ui`/`repl`/`tui`/`cli`. **Só apresentação** — erros de `task`/`store` ficam em inglês para não quebrar o layering. Toggle: `/lang` no REPL, `taskframe lang` na CLI; precedência resolvida em `main.go` (`--lang` > `TASKFRAME_LANG` > setting `lang` > config > en).
- `internal/ui` — camada visual compartilhada: `Theme`, `NewTheme`, `NormalizeTheme`, `NextTheme`, `DrawBox`/`DrawBoxChars`, `TruncRunes`, `PadRow`, `PadRowPlain`, `VisibleWidth`. Importado por `tui` e `repl`.
- `internal/tui` (clássica) e `internal/repl` (padrão) — irmãos; importam `task`+`store`+`ui`. **Nenhum toca `database/sql`**: chamam o store dentro de closures `tea.Cmd` e recebem o resultado como mensagem. `tui/theme.go` é um shim fino que re-exporta `internal/ui` para o código antigo (que usa grafias minúsculas `truncRunes` etc.).
- `internal/cli` — CLI one-shot; importa `task`+`store`.
- `cmd/taskframe/main.go` — sem args → `repl.Run`; `classic` → `tui.Run`; senão → `cli.Run`. `resolveOptions` (precedência `--theme` > `TASKFRAME_THEME` > setting > default; idem para `lang`) é reusado pelos três.

### REPL (`internal/repl`, o padrão)

Bubble Tea **sem alt-screen**. `run.go` imprime o banner uma vez (`fmt.Print`) e roda o programa; a saída de cada comando vai ao scrollback real via `tea.Println` (o renderer padrão só o processa fora de alt-screen). Regras: `View()` sempre curto (só o prompt ou um overlay **limitado** a ~14 linhas); o eco do comando + resultado saem juntos em **um único** `tea.Println` (ordem garantida, ver `model.emit`); nunca ecoar pelo `View()`. Máquina de estados em `model.go`: `modePrompt` ↔ `modeList`/`modeDetail` (overlays transientes que somem ao fechar, sem poluir scrollback) ↔ `modeNote`. `commands.go` faz o dispatch (natural vs `/slash`) e reusa `task.ParseTokens` + verbos do store. `complete.go`/`history.go` = Tab e ↑↓. Seam de teste: `model.transcript` acumula cada bloco emitido (asserir nele, nunca em códigos ANSI nem no `printLineMessage` unexported do bubbletea).

### Decisões de modelagem (não mudar sem motivo forte)

- **Projetos não são tabela**: string pontuada (`casa.mercado`) na coluna `tasks.project`; a árvore da sidebar é derivada de `ProjectCounts()` em tempo de leitura. Filtro por projeto é prefix-match (`= ?` OR `LIKE ?.%`).
- **Subtarefas**: adjacency list (`parent_id`), nesting arbitrário. `store.BuildTree()` monta a árvore em memória a partir da lista flat — sem CTEs recursivas.
- **Urgency nunca é armazenada** — calculada em `task.Urgency()` na leitura, para a fórmula poder mudar sem migration. Coeficientes em `DefaultCoefficients` (urgency.go).
- **Audit log + undo/redo**: toda mutação grava linhas na tabela `activity` dentro da MESMA transação, agrupadas por `op_id` (hex aleatório). Undo/redo usam **marcadores**: uma linha `kind='undo'` (ou `'redo'`) cujo `field` guarda o op_id afetado. Um op está "aplicado" quando seu marcador mais recente (`lastMarkerExpr`) não é `undo`; "desfeito" quando é `undo`. `store.Undo()` reverte o op aplicado mais recente (linha a linha, `old_val`); `store.Redo()` reaplica o op desfeito mais recente (`new_val`), a menos que um op novo tenha surgido depois do undo (descarta o redo). O `create` grava um **snapshot JSON da task em `old_val`** (senão vazio → redo de create recusa); undo de create ainda é DELETE físico. Undo de um `done` com recorrência reverte o done E apaga a instância clonada (mesmo op_id). **Ao adicionar qualquer mutação nova, registre em `activity` no mesmo tx e garanta o caminho reverso (undo) e o direto (redo) — campos escalares/tags passam por `setField`.**
- **Exceção deliberada**: a tabela `settings` (migration 2) NÃO loga em `activity` — preferências de máquina (tema, ordenação, **contexts**: `context` + `context.<nome>`); o undo nunca deve revertê-las. Settings ficam fora do export. Config de arquivo (`internal/config`, `config.json`) é uma camada separada de defaults: settings de runtime vencem o config.
- **Export/import** (`store/export.go`): backup JSON completo com ids preservados verbatim (o log restaurado mantém o undo funcionando). Import só em banco vazio, em tx única com `PRAGMA defer_foreign_keys = ON` (após um Move, um filho pode referenciar pai com id maior).
- **Delete é soft** (`status='deleted'`) para o undo funcionar; `purge` faz o hard-delete.
- **Timestamps**: TEXT RFC3339 UTC (ordenável, legível no sqlite3). Helpers `fmtTime`/`parseTime` em store.go — sempre use-os.
- **Migrations**: `PRAGMA user_version` + slice ordenado em `migrate.go` (índice+1 == versão). schema.sql embutido via `go:embed` é a migration 1; alterações de schema entram como novas strings no slice, nunca editando a 1.
- **DB**: `%APPDATA%\taskframe\taskframe.db`, override por `TASKFRAME_DB`/`--db`. WAL + `busy_timeout` + `SetMaxOpenConns(1)` — a conexão única elimina contenção de write-lock; não aumentar. DSN usa `filepath.ToSlash` (backslash em DSN é bug clássico no Windows).

### TUI (Bubble Tea)

`tui/app.go` é o root model e único dono do estado: `filter task.Filter` atual, foco (sidebar/lista), e `modal Modal` (interface local; um modal por vez — Form, Confirm, NotePrompt, Detail, Help). Fluxo: tecla → modal ou painel → mensagem de intenção (`formSubmittedMsg`, `noteSubmittedMsg`...) → App persiste via `tea.Cmd` → `statusMsg` → **todo `statusMsg` dispara `reload()`** (recarrega tarefas + projetos). Sem mutação otimista.

- Visual em `internal/ui`: **4 temas nomeados** (`dark` padrão sem pintar fundo, `borland` navy truecolor, `green`/`amber` fósforo) como structs `palette` → `NewTheme(name, ascii)`. Ao adicionar cor nova, adicione em TODAS as paletas. `DrawBox`/`DrawBoxChars` (borda dupla ou `RoundBox` do REPL) desenham com título embutido na borda (lipgloss v1 não tem border title). Flag `--ascii` compõe com qualquer tema.
- Ordenação: `task.SortMode` (string: urgency/due/created); `store.BuildTree(tasks, now, mode)`.
- Mover: o chamador valida ciclos via `store.CheckMoveCycle` antes de `UpdateTask` — sem isso, `BuildTree` some silenciosamente com o subgrafo cíclico.
- A lista de tarefas tem renderer de árvore próprio — `bubbles/table` não suporta indentação; não migrar.
- **Toda F-key tem alias em letra** (clássica) e o REPL tem verbos + `/slash` — terminais/IDEs engolem F-keys.
- Textinputs usam `cursor.CursorStatic` — o blink gera ticks que travam os testes síncronos. Manter em inputs novos.

### Testes

`repl/repl_test.go` e `tui/smoke_test.go` dirigem o modelo real sem terminal: `drive()`/`exec()` executam cada `tea.Cmd` sincronamente e realimentam as mensagens (o `printLineMessage` do `tea.Println` é ignorado por nome de tipo, pois é unexported). Novos testes: `store.OpenMemory()`, `typeText`, e asserir em `model.transcript` (scrollback do REPL) / `stripANSI(View())` (overlays), nunca em códigos ANSI.

### CLI

Stdlib `flag` + switch manual — **sem cobra** (os tokens taskwarrior não são flags; parser em `task.ParseTokens`). Saída do `list` é texto puro sem ANSI (pipe-friendly). Mensagens em português; código e comentários em inglês.

## Estado atual (2026-07-12) e retomada

**v1**: núcleo completo (tarefas, subtarefas, projetos hierárquicos, notas, activity log, filtros/busca) + urgency sort, undo, recorrência, soft-delete/purge, datas em linguagem natural (pt e en: `sex`, `fri`, `3d`, `eom`...).

**v2** (mesma data): sistema de 4 temas com persistência (motivado por "as cores doem os olhos" — o antigo fundo azul ANSI-4 foi substituído pelo `dark` como padrão); filtros virtuais Hoje/Atrasadas/Semana/Aguardando e seção de tags na sidebar; campos wait/scheduled no form e no detail; toggle de ordenação (`o`); diálogo Mover (F6/`m`) com validação de ciclo; export/import JSON; git + GitHub Actions CI (vet+test em ubuntu/windows, gofmt no ubuntu; `.gitattributes` força LF em `.go` para o check de gofmt não quebrar com autocrlf).

**v3** (mesma data): pivô de interface. O usuário preferiu algo estilo Claude Code, então o **REPL inline** (`internal/repl`) virou o padrão; a TUI Norton Commander virou `taskframe classic`. Fase 0 extraiu `internal/ui` (tema), `task.ParseTokens` (parser) e `store.CheckMoveCycle`, compartilhados. Novo `install.ps1` instala como comando global do usuário (PATH por-usuário, idempotente, `SetEnvironmentVariable` não `setx`).

**v4** (2026-07-13): aproximação do Taskwarrior, centro de gravidade no terminal por comandos (CLI+REPL sobre o mesmo núcleo; a TUI clássica é referência, não muda). Cinco frentes:
- **Seleção/filtros/reports** (`task/idspec.go`, `task/report.go`): `ParseIDSpec` (ranges/listas `1-3,5`, substitui os `parseIDs` duplicados); token `-tag` (exclusão) e `status:`; `Filter.ExcludeTags`/`ActiveOnly`/`NoContext` + `Filter.Merge`; registry de reports nomeados `next/overdue/today/week/waiting/active` (fonte única — a sidebar da TUI deriva dele). Ergonomia: tecla `a` no overlay do REPL cria filho sob o cursor (sem digitar id).
- **Contexts** (`store/context.go`): filtro default salvo em `settings` (`context`, `context.<nome>`); comandos `context [nome|none|list|define|delete]`; aplicado em list/reports (base ⊕ contexto ⊕ tokens do usuário, usuário vence); token `nocontext` ignora.
- **start/stop** (coluna `start` via migration 3): `Task.Start`/`IsActive`, `StartTask`/`StopTask` (activity+undo), coeficiente de urgência `Active` (+4), report/filtro `active`, marca `[>]`.
- **Config `.taskrc`** (`internal/config`): `config.json` (`TASKFRAME_CONFIG` ou `%APPDATA%`), campos `theme/sort/urgency`; `task.ActiveCoefficients` + `ConfigureUrgency` (urgência configurável sem recompilar). Precedência: settings (runtime) vence config (default de arquivo).
- **redo** (`store.Redo`): marcadores `kind='redo'` que cancelam `undo`; detecção do último op desfeito via `lastMarkerExpr` (último marcador do op); um op novo depois do undo descarta o redo (semântica padrão). `create` grava snapshot JSON em `old_val` (retrocompatível: log antigo sem snapshot recusa redo de create). `setField` aplica valor a um campo (undo usa old, redo usa new).

Testes: ~55 casos; verificados com o binário real (reports, ranges, `-tag`, contexts, start/urgência/active, config muda ordenação, ciclos undo↔redo, redo de create/invalidação). Tudo verde.

Ideias futuras (nada pendente do escopo aprovado):
- histórico do REPL persistido (hoje em memória, `repl/history.go`)
- FTS5 para busca (só se a busca LIKE ficar lenta)
- filtros com OR / `not:` de atributo (hoje só AND + `-tag`)
- `edit` que limpa campos (hoje só define, não limpa — `ParseTokens` não distingue "ausente" de "vazio")
- lembretes/notificações (fora do escopo atual)
