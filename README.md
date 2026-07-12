# TaskFrame

Gerenciador de tarefas no terminal com visual anos 90 estilo Norton Commander.
Inspirado no [taskwarrior](https://taskwarrior.org/), porém mais simples.

```
╔═ Projetos ═════════════╗╔═ Tarefas ═══════════════════════════════╗
║ (todas)              4 ║║    1 [ ] H 14/07  Comprar leite +urgente║
║ casa                 2 ║║ ▾  2 [ ]          Revisar relatório     ║
║   mercado            1 ║║    4 [ ]            Escrever testes     ║
║ trabalho             2 ║║    3 [ ]          Regar plantas         ║
╚════════════════════════╝╚═════════════════════════════════════════╝
 4 tarefa(s)
1Ajuda 2Add 3Ver 4Edit 5Nota 6Sub 7Busca 8Del 9Done 10Sair
```

## Build

```
go build -o taskframe.exe ./cmd/taskframe
```

## Uso

`taskframe` sem argumentos abre a TUI. Com argumentos, funciona como CLI de
captura rápida:

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

## TUI

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
| `o` | alterna ordenação (urgência / vencimento / criação) |
| `t` | alterna tema |
| `u` | desfazer |
| `F10`/`q` | sair |

Toda tecla de função tem um alias em letra (alguns terminais capturam F-keys).

A sidebar traz, além dos projetos, filtros virtuais (**Hoje**, **Atrasadas**,
**Semana**, **Aguardando**) e as tags em uso.

## Temas

Quatro temas, trocáveis em tempo real com `t` (a escolha fica salva):

- **dark** (padrão) — usa o fundo do seu terminal, acentos discretos
- **borland** — navy retrô estilo Turbo Vision, dessaturado
- **green** / **amber** — fósforo monocromático estilo CRT

Precedência: flag `--theme` > env `TASKFRAME_THEME` > escolha salva > dark.

## Dados

Banco SQLite em `%APPDATA%\taskframe\taskframe.db` (Windows). Sobrescreva com
a variável `TASKFRAME_DB` ou a flag `--db`. Toda mutação é registrada na
tabela `activity` — o histórico completo de cada tarefa fica visível no
detalhe (F3) e alimenta o `undo`.

Ordenação padrão por **urgência** (fórmula ponderada estilo taskwarrior:
vencimento, prioridade, tag `+next`, idade, subtarefas pendentes).

Terminal recomendado: Windows Terminal. Em consoles sem suporte a bordas
duplas Unicode, use `taskframe --ascii`.
