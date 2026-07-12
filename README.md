# TaskFrame

Gerenciador de tarefas no terminal, inspirado no
[taskwarrior](https://taskwarrior.org/) porém mais simples. A interface padrão
é um REPL inline estilo Claude Code: logo no topo, um prompt embaixo, e a saída
rola no histórico do próprio terminal.

```
      .  *  .      T A S K F R A M E
   .   \ | /   .   tarefas no terminal
 --  --  (*)  --
   .   / | \   .

› add comprar leite pro:casa due:sex
  ✓ tarefa 5 criada: comprar leite

› list
╭─ tarefas ────────────────────────────────╮
│ › 5 [ ] H sex    comprar leite            │
│   3 [ ]          revisar relatório        │
╰──────────────────────────────────────────╯
  ↑↓ move · enter abre · d conclui · esc fecha

╭──────────────────────────────────────────╮
│ › _                                       │
╰──────────────────────────────────────────╯
```

## Instalação

Para rodar `taskframe` de qualquer pasta (como o `claude`):

```powershell
.\install.ps1
```

Compila e instala em `%LOCALAPPDATA%\Programs\taskframe`, adicionando ao PATH do
usuário (sem admin). Abra um novo terminal e rode `taskframe`.

Ou só compile localmente:

```
go build -o taskframe.exe ./cmd/taskframe
```

## Uso

`taskframe` sem argumentos abre o REPL. `taskframe classic` abre a antiga TUI
Norton Commander de dois painéis. Com um subcomando, funciona como CLI de
captura rápida (sem abrir interface):

```
taskframe add "Comprar leite" pro:casa.mercado +urgente due:tomorrow prio:H
taskframe list                  # tabela ordenada por urgência
taskframe done 12               # concluir (recorrentes criam a próxima)
taskframe del 12                # soft-delete (undo desfaz, purge remove)
taskframe note 12 "esperando o Marcos"
taskframe undo                  # desfaz a última operação
taskframe purge                 # remove definitivamente as deletadas
taskframe export > backup.json  # backup completo (tarefas, notas, histórico)
taskframe import backup.json    # restaura (apenas em banco vazio)
```

Tokens aceitos em `add` e `list`: `pro:projeto.sub`, `+tag`, `due:<data>`,
`prio:H|M|L`, `wait:<data>`, `recur:daily|weekly|3d…`, `sub:<id>` (subtarefa).
Datas: `today`, `tomorrow`, `3d`, `2w`, `fri`/`sex`, `15/08`, `2026-08-15`, `eow`, `eom`.

## REPL (interface padrão)

Comandos naturais (sem barra) e de app (com barra). Histórico com `↑`/`↓` e
autocompletar com `Tab` (comandos, projetos e tags).

| Comando | Ação |
|---|---|
| `add <título> [tokens]` | cria tarefa |
| `list [tokens]` | abre a lista navegável (setas, `enter` abre, `esc` fecha) |
| `done <id…>` · `del <id…>` | conclui · deleta |
| `note <id> [texto]` | nota (sem texto abre um campo) |
| `edit <id> <tokens>` | altera campos |
| `move <id> pro:x sub:N` | muda projeto/pai (`sub:0` vira raiz) |
| `undo` | desfaz a última operação |
| `/theme [nome]` · `/sort [modo]` | tema · ordenação |
| `/help` · `/clear` · `/quit` | ajuda · limpa · sai (`Ctrl+D`) |

Na lista navegável: `↑↓`/`jk` move, `enter` abre o detalhe (notas + histórico),
`d`/espaço conclui, `x` deleta, `esc` fecha.

## TUI clássica (`taskframe classic`)

A antiga interface Norton Commander de dois painéis continua disponível:

| Tecla | Ação |
|---|---|
| `Tab` | alterna painéis (projetos ↔ tarefas) |
| `↑↓`/`jk`, `←→`/`hl` | navega / recolhe / expande subtarefas |
| `Enter`, `F3` | detalhes: notas + histórico completo |
| `F2`/`a` · `s` | nova tarefa · nova subtarefa |
| `F4`/`e` · `F5`/`n` | editar · adicionar nota |
| `F6`/`m` | mover (projeto / pai) |
| `F9`/`d`/`Espaço` | concluir / reabrir |
| `F8`/`x` | deletar (com confirmação) |
| `F7`/`/` | busca por texto |
| `o` · `t` · `u` | ordenação · tema · desfazer |
| `F10`/`q` | sair |

A sidebar traz, além dos projetos, filtros virtuais (**Hoje**, **Atrasadas**,
**Semana**, **Aguardando**) e as tags em uso.

## Temas

Quatro temas, trocáveis com `/theme` (REPL) ou `t` (clássica); a escolha fica
salva:

- **dark** (padrão) — usa o fundo do seu terminal, acentos discretos
- **borland** — navy retrô estilo Turbo Vision, dessaturado
- **green** / **amber** — fósforo monocromático estilo CRT

Precedência: flag `--theme` > env `TASKFRAME_THEME` > escolha salva > dark.

## Dados

Banco SQLite em `%APPDATA%\taskframe\taskframe.db` (Windows). Sobrescreva com
a variável `TASKFRAME_DB` ou a flag `--db`. Toda mutação é registrada na
tabela `activity` — o histórico completo de cada tarefa fica visível no
detalhe e alimenta o `undo`.

Ordenação padrão por **urgência** (fórmula ponderada estilo taskwarrior:
vencimento, prioridade, tag `+next`, idade, subtarefas pendentes).

Terminal recomendado: Windows Terminal. Em consoles sem suporte a bordas
duplas Unicode, use `taskframe --ascii`.
