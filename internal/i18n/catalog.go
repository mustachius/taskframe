package i18n

// catalog maps a stable key to its {english, pt-br} pair. English (index 0) is
// the canonical source; an empty pt-br entry falls back to English at lookup.
var catalog = map[string][2]string{
	// --- banner / hint (repl) ---
	"banner.subtitle": {"tasks in your terminal", "tarefas no terminal"},
	"hint.tip":        {"tip: ", "dica: "},
	"hint.example":    {"'add buy milk due:fri'", "'add comprar leite due:sex'"},
	"hint.creates":    {" creates · ", " cria · "},
	"hint.navigates":  {" navigates · ", " navega · "},
	"hint.help":       {" help · ", " ajuda · "},
	"hint.quit":       {" quit", " sai"},

	// --- prompt / note / child (repl model) ---
	"prompt.placeholder": {"add task… · list · /help", "add tarefa… · list · /help"},
	"note.promptGlyph":   {"note› ", "nota› "},
	"child.promptGlyph":  {"child› ", "filho› "},
	"edit.promptGlyph":   {"edit› ", "editar› "},
	"note.boxTitle":      {"note · ", "nota · "},
	"note.boxHint":       {" ctrl+d saves · enter new line · esc cancels", " ctrl+d salva · enter nova linha · esc cancela"},
	"child.boxTitle":     {"child of · ", "filho de · "},
	"child.boxHint":      {" enter creates · esc cancels", " enter cria · esc cancela"},
	"edit.boxTitle":      {"edit · ", "editar · "},
	"edit.boxHint":       {" pro:x +tag due:… prio:H · enter applies · esc cancels", " pro:x +tag due:… prio:H · enter aplica · esc cancela"},
	"note.cancelled":     {"  (note cancelled)", "  (nota cancelada)"},
	"note.empty":         {"  (empty note, ignored)", "  (nota vazia, ignorada)"},

	// --- status messages (repl) ---
	"status.noteAdded":        {"  note added to task %d", "  nota adicionada à tarefa %d"},
	"status.taskCreated":      {"  task %d created: %s", "  tarefa %d criada: %s"},
	"status.taskCreatedUnder": {"  task %d created under %d: %s", "  tarefa %d criada sob %d: %s"},
	"status.taskDone":         {"  task %d done", "  tarefa %d concluída"},
	"status.recur":            {"    recurrence: task %d due %s", "    recorrência: tarefa %d vence %s"},
	"status.taskDeletedUndo":  {"  task %d deleted (undo reverts)", "  tarefa %d deletada (undo desfaz)"},
	"status.taskDeleted":      {"  task %d deleted", "  tarefa %d deletada"},
	"status.taskUpdated":      {"  task %d updated", "  tarefa %d atualizada"},
	"status.taskMoved":        {"  task %d moved", "  tarefa %d movida"},
	"status.taskStarted":      {"  task %d started", "  tarefa %d iniciada"},
	"status.taskStopped":      {"  task %d stopped", "  tarefa %d parada"},
	"status.undone":           {"  undone: %s", "  desfeito: %s"},
	"status.redone":           {"  redone: %s", "  refeito: %s"},
	"status.purged":           {"  %d task(s) permanently removed", "  %d tarefa(s) removida(s) definitivamente"},

	// --- usage / UI errors (repl) ---
	"usage.add":           {"usage: add <title> [pro:x +tag due:x prio:H sub:N]", "uso: add <título> [pro:x +tag due:x prio:H sub:N]"},
	"usage.sub":           {"usage: sub <parent> <title> [pro:x +tag due:x prio:H]", "uso: sub <pai> <título> [pro:x +tag due:x prio:H]"},
	"usage.note":          {"usage: note <id> [text]", "uso: note <id> [texto]"},
	"usage.read":          {"usage: read <id>", "uso: read <id>"},
	"usage.edit":          {"usage: edit <id> <fields>  (e.g. edit 5 prio:H due:fri new title)", "uso: edit <id> <campos>  (ex: edit 5 prio:H due:sex novo título)"},
	"usage.move":          {"usage: move <id> pro:project [sub:parentId]  (sub:0 = root)", "uso: move <id> pro:projeto [sub:idPai]  (sub:0 vira raiz)"},
	"err.titleEmpty":      {"empty title", "título vazio"},
	"err.parentMissing":   {"parent %d does not exist", "pai %d não existe"},
	"err.parentIdInvalid": {"invalid parent id: %s", "id do pai inválido: %s"},
	"err.idInvalid":       {"invalid id: %s", "id inválido: %s"},
	"err.subNumeric":      {"sub: expects a numeric id (or 0)", "sub: espera um id numérico (ou 0)"},
	"err.nothingToMove":   {"nothing to move: provide pro: and/or sub:", "nada a mover: informe pro: e/ou sub:"},
	"err.unknownCmd":      {"x unknown command: %s", "x comando desconhecido: %s"},
	"hint.helpList":       {"  (/help for the list)", "  (/help para a lista)"},
	"hint.helpShort":      {"  (/help)", "  (/help)"},
	"err.themeInvalid":    {"x invalid theme: %s", "x tema inválido: %s"},
	"hint.themes":         {"  (dark, borland, green, amber)", "  (dark, borland, green, amber)"},
	"status.theme":        {"  theme: %s", "  tema: %s"},
	"status.sort":         {"  sort: %s", "  ordenação: %s"},
	"err.langInvalid":     {"x invalid language: %s", "x idioma inválido: %s"},
	"hint.langs":          {"  (en, pt-br)", "  (en, pt-br)"},
	"status.lang":         {"  language: %s", "  idioma: %s"},
	"classic.run":         {"  run: ", "  rode: "},
	"classic.hint":        {"  (two-pane interface)", "  (interface de dois painéis)"},
	"repl.sync.running":   {"syncing…", "sincronizando…"},
	"repl.sync.busy":      {"  (sync already running)", "  (sync já em andamento)"},

	// --- context (repl) ---
	"ctx.noneActive":   {"  no active context  (context <name> activates · context list shows)", "  nenhum contexto ativo  (context <nome> ativa · context list mostra)"},
	"ctx.activeLabel":  {"  active context: ", "  contexto ativo: "},
	"ctx.noneDefined":  {"  no context defined", "  nenhum contexto definido"},
	"usage.ctxDefine":  {"usage: context define <name> <tokens>", "uso: context define <nome> <tokens>"},
	"status.ctxDefine": {"  context %s: %s", "  contexto %s: %s"},
	"ctx.deactivated":  {"  context deactivated", "  contexto desativado"},
	"usage.ctxDelete":  {"usage: context delete <name>", "uso: context delete <nome>"},
	"ctx.removed":      {"  context %s removed", "  contexto %s removido"},
	"err.ctxUndefined": {"context %q not defined (context define %s <tokens>)", "contexto %q não definido (context define %s <tokens>)"},
	"status.ctxActive": {"  active context: %s", "  contexto ativo: %s"},

	// --- overlay / detail (repl) ---
	"overlay.empty":  {" no tasks", " nenhuma tarefa"},
	"overlay.hint":   {"  ↑↓ move · ←→ fold · a child · n note · e edit · enter open · d done · x delete · esc close", "  ↑↓ move · ←→ recolhe · a filho · n nota · e edita · enter abre · d conclui · x deleta · esc fecha"},
	"detail.footer":  {"  ↑↓ scroll · n note · e edit · esc back", "  ↑↓ rola · n nota · e edita · esc volta"},
	"detail.title":   {"task", "tarefa"},
	"detail.titleN":  {"task %d", "tarefa %d"},
	"list.title":     {"tasks", "tarefas"},
	"list.searchSep": {" · search: ", " · busca: "},

	// --- shared detail field labels (repl + tui) ---
	"lbl.status":      {"Status", "Status"},
	"lbl.started":     {"Started", "Iniciada"},
	"lbl.parent":      {"Parent", "Pai"},
	"lbl.project":     {"Project", "Projeto"},
	"lbl.tags":        {"Tags", "Tags"},
	"lbl.priority":    {"Priority", "Prioridade"},
	"lbl.due":         {"Due", "Vencimento"},
	"lbl.waitUntil":   {"Wait until", "Aguardar até"},
	"lbl.scheduled":   {"Scheduled", "Agendada"},
	"lbl.recurrence":  {"Recurrence", "Recorrência"},
	"lbl.created":     {"Created", "Criada em"},
	"lbl.completed":   {"Completed", "Concluída em"},
	"detail.subtasks": {"subtasks %d/%d", "subtarefas %d/%d"},
	"detail.notes":    {"Notes", "Notas"},
	"detail.history":  {"History", "Histórico"},

	// --- activity descriptions (repl + tui) ---
	"act.created": {"created: ", "criada: "},
	"act.done":    {"completed", "concluída"},
	"act.deleted": {"deleted", "deletada"},
	"act.note":    {"note added", "nota adicionada"},
	"act.setTo":   {"%s set: %s", "%s definido: %s"},
	"act.changed": {"%s: %s → %s", "%s: %s → %s"},

	// --- sort labels (repl + tui) ---
	"sort.urgency": {"urgency", "urgência"},
	"sort.due":     {"due", "vencimento"},
	"sort.created": {"created", "criação"},

	// --- report descriptions (repl + tui + cli) ---
	"report.next":    {"most urgent pending", "pendências mais urgentes"},
	"report.overdue": {"overdue", "vencidas"},
	"report.today":   {"due today", "vencem até hoje"},
	"report.week":    {"next 7 days", "próximos 7 dias"},
	"report.waiting": {"waiting (future wait)", "aguardando (wait futuro)"},
	"report.active":  {"in progress (started)", "em andamento (iniciadas)"},

	// --- repl /help table ---
	"help.title":       {"  TaskFrame — commands", "  TaskFrame — comandos"},
	"help.add.k":       {"add <title> [tokens]", "add <título> [tokens]"},
	"help.add.v":       {"create task (pro:x +tag due:fri prio:H wait:3d recur:weekly sub:N)", "cria tarefa (pro:x +tag due:sex prio:H wait:3d recur:weekly sub:N)"},
	"help.sub.k":       {"sub <parent> <title>", "sub <pai> <título>"},
	"help.sub.v":       {"create subtask under <parent>", "cria subtarefa sob <pai>"},
	"help.list.k":      {"list [tokens]", "list [tokens]"},
	"help.list.v":      {"open the navigable list (arrows, ←→ fold, a add child, enter open)", "abre a lista navegável (setas, ←→ recolhe, a add filho, enter abre)"},
	"help.reports.k":   {"next · overdue · today · week · waiting · active", "next · overdue · today · week · waiting · active"},
	"help.reports.v":   {"reports (accept tokens: next pro:work)", "reports (aceitam tokens: next pro:work)"},
	"help.done.k":      {"done <ids>", "done <ids>"},
	"help.done.v":      {"complete — ids: 1  1,5  1-3", "conclui — ids: 1  1,5  1-3"},
	"help.startstop.k": {"start/stop <ids>", "start/stop <ids>"},
	"help.startstop.v": {"mark in progress (urgency rises)", "marca em andamento (urgência sobe)"},
	"help.del.k":       {"del <ids>", "del <ids>"},
	"help.del.v":       {"delete (undo reverts)", "deleta (undo desfaz)"},
	"help.note.k":      {"note <id> [text]", "note <id> [texto]"},
	"help.note.v":      {"add note (no text opens the field)", "adiciona nota (sem texto abre o campo)"},
	"help.edit.k":      {"edit <id> <tokens>", "edit <id> <tokens>"},
	"help.edit.v":      {"change fields (+tag adds, -tag removes)", "altera campos (+tag adiciona, -tag remove)"},
	"help.move.k":      {"move <id> pro:x sub:N", "move <id> pro:x sub:N"},
	"help.move.v":      {"change project/parent", "muda projeto/pai"},
	"help.context.k":   {"context [name|none|list|define …]", "context [nome|none|list|define …]"},
	"help.context.v":   {"saved default filter (nocontext ignores)", "filtro default salvo (nocontext ignora)"},
	"help.filters.k":   {"filters", "filtros"},
	"help.filters.v":   {"+tag -tag pro:x due:x prio:H status:done all", "+tag -tag pro:x due:x prio:H status:done all"},
	"help.undoredo.k":  {"undo · redo", "undo · redo"},
	"help.undoredo.v":  {"undo · redo the last operation", "desfaz · refaz a última operação"},
	"help.theme.k":     {"/theme [name]", "/theme [nome]"},
	"help.theme.v":     {"switch color theme", "troca o tema de cores"},
	"help.sort.k":      {"/sort [mode]", "/sort [modo]"},
	"help.sort.v":      {"sort: urgency, due, created", "ordenação: urgency, due, created"},
	"help.lang.k":      {"/lang [en|pt-br]", "/lang [en|pt-br]"},
	"help.lang.v":      {"switch language", "troca o idioma"},
	"help.sync.k":      {"/sync [init <url>|status|pull|push]", "/sync [init <url>|status|pull|push]"},
	"help.sync.v":      {"git sync (same as taskframe sync)", "sincroniza via git (igual a taskframe sync)"},
	"help.clear.k":     {"/clear", "/clear"},
	"help.clear.v":     {"clear the screen", "limpa a tela"},
	"help.quit.k":      {"/help · /quit", "/help · /quit"},
	"help.quit.v":      {"help · quit (Ctrl+D)", "ajuda · sair (Ctrl+D)"},

	// --- CLI help block ---
	"cli.help": {`taskframe — terminal task manager

usage:
  taskframe                     open the TUI
  taskframe add <title> [tokens]
  taskframe list [tokens]
  taskframe done <ids>          ids: 1  1,5  1-3  (ranges and lists)
  taskframe del <ids>
  taskframe note <id> <text>
  taskframe move <id> [pro:x] [sub:N]   change project/parent (sub:0 = root)
  taskframe context [<name>|none|list|define <name> <tokens>|delete <name>]
  taskframe start <ids>         mark in progress (urgency rises)
  taskframe stop <ids>
  taskframe lang [en|pt-br]     show/switch language
  taskframe undo
  taskframe redo                redo the last undo
  taskframe purge               permanently remove deleted tasks
  taskframe export              full JSON backup to stdout
  taskframe import [--replace] <file>   restore backup (--replace overwrites)
  taskframe sync [init <url>|status|pull|push]   git sync across machines

reports (accept extra tokens, e.g. taskframe next pro:work):
  next            most urgent pending (top 15)
  overdue         overdue
  today           due today
  week            next 7 days
  waiting         waiting (future wait)
  active          in progress (started)

tokens (add and list):
  pro:work.api    project (dotted hierarchy)
  +tag / -tag     require / exclude the tag (-tag: list only)
  due:fri         due date (today, tomorrow, 3d, fri/sex, 2026-08-01...)
  prio:H          priority H, M or L
  wait:1w         hide until the date
  recur:weekly    recurrence (daily, weekly, monthly, 3d...)
  sub:12          create as subtask of task 12 (add only)
  status:done     filter by status (pending, done, deleted, all)
  all             include completed/deleted (list only)
  free text       in add becomes the title; in list becomes the search
`, `taskframe — gerenciador de tarefas no terminal

uso:
  taskframe                     abre a TUI
  taskframe add <título> [tokens]
  taskframe list [tokens]
  taskframe done <ids>          ids: 1  1,5  1-3  (ranges e listas)
  taskframe del <ids>
  taskframe note <id> <texto>
  taskframe move <id> [pro:x] [sub:N]   muda projeto/pai (sub:0 vira raiz)
  taskframe context [<nome>|none|list|define <nome> <tokens>|delete <nome>]
  taskframe start <ids>         marca em andamento (urgência sobe)
  taskframe stop <ids>
  taskframe lang [en|pt-br]     mostra/troca o idioma
  taskframe undo
  taskframe redo                refaz o último undo
  taskframe purge               remove definitivamente tarefas deletadas
  taskframe export              backup JSON completo no stdout
  taskframe import [--replace] <arquivo>   restaura backup (--replace sobrescreve)
  taskframe sync [init <url>|status|pull|push]   sincroniza via git entre máquinas

reports (aceitam tokens extras, ex: taskframe next pro:work):
  next            pendências mais urgentes (top 15)
  overdue         vencidas
  today           vencem até hoje
  week            próximos 7 dias
  waiting         aguardando (wait futuro)
  active          em andamento (iniciadas)

tokens (add e list):
  pro:work.api    projeto (hierarquia com pontos)
  +tag / -tag     exige / exclui a tag (só list para -tag)
  due:sex         vencimento (today, tomorrow, 3d, fri/sex, 2026-08-01...)
  prio:H          prioridade H, M ou L
  wait:1w         esconder até a data
  recur:weekly    recorrência (daily, weekly, monthly, 3d...)
  sub:12          criar como subtarefa da tarefa 12 (só add)
  status:done     filtra por status (pending, done, deleted, all)
  all             incluir concluídas/deletadas (só list)
  texto livre     no add vira título; no list vira busca
`},

	// --- CLI messages ---
	"cli.usage.add":         {"usage: taskframe add <title> [pro:x +tag due:x prio:H]", "uso: taskframe add <título> [pro:x +tag due:x prio:H]"},
	"cli.taskCreated":       {"task %d created: %s", "tarefa %d criada: %s"},
	"cli.ctxTag":            {"[context: %s]", "[contexto: %s]"},
	"cli.taskDone":          {"task %d completed", "tarefa %d concluída"},
	"cli.recurCreated":      {"recurrence: task %d created, due %s", "recorrência: tarefa %d criada, vence %s"},
	"cli.taskDeleted":       {"task %d deleted (undo to revert, purge to remove for good)", "tarefa %d deletada (undo para desfazer, purge para remover de vez)"},
	"cli.usage.note":        {"usage: taskframe note <id> <text>", "uso: taskframe note <id> <texto>"},
	"cli.noteAdded":         {"note added to task %d", "nota adicionada à tarefa %d"},
	"cli.usage.move":        {"usage: taskframe move <id> [pro:project] [sub:parentId]  (sub:0 = root)", "uso: taskframe move <id> [pro:projeto] [sub:idPai]  (sub:0 vira raiz)"},
	"cli.taskMoved":         {"task %d moved", "tarefa %d movida"},
	"cli.ctx.noneActive":    {"no active context (context <name> activates · context list shows)", "nenhum contexto ativo (context <nome> ativa · context list mostra)"},
	"cli.ctx.active":        {"active context: %s  (%s)", "contexto ativo: %s  (%s)"},
	"cli.ctx.noneDefined":   {"no context defined", "nenhum contexto definido"},
	"cli.usage.ctxDefine":   {"usage: taskframe context define <name> <tokens>", "uso: taskframe context define <nome> <tokens>"},
	"cli.ctx.defined":       {"context %s defined: %s", "contexto %s definido: %s"},
	"cli.ctx.deactivated":   {"context deactivated", "contexto desativado"},
	"cli.usage.ctxDelete":   {"usage: taskframe context delete <name>", "uso: taskframe context delete <nome>"},
	"cli.ctx.removed":       {"context %s removed", "contexto %s removido"},
	"cli.ctx.active2":       {"active context: %s", "contexto ativo: %s"},
	"cli.taskStarted":       {"task %d started", "tarefa %d iniciada"},
	"cli.taskStopped":       {"task %d stopped", "tarefa %d parada"},
	"cli.undone":            {"undone:", "desfeito:"},
	"cli.redone":            {"redone:", "refeito:"},
	"cli.usage.import":      {"usage: taskframe import [--replace] <file.json>", "uso: taskframe import [--replace] <arquivo.json>"},
	"cli.err.jsonInvalid":   {"invalid json", "json inválido"},
	"cli.import.summary":    {"%s: %d task(s), %d note(s), %d history record(s)", "%s: %d tarefa(s), %d nota(s), %d registro(s) de histórico"},
	"cli.import.imported":   {"imported", "importado"},
	"cli.import.replaced":   {"replaced", "substituído"},
	"cli.purged":            {"%d task(s) permanently removed", "%d tarefa(s) removida(s) definitivamente"},
	"cli.lang.current":      {"language: %s", "idioma: %s"},
	"cli.lang.invalid":      {"invalid language: %s (use en, pt-br)", "idioma inválido: %s (use en, pt-br)"},
	"cli.render.noTasks":    {"no tasks", "nenhuma tarefa"},
	"cli.render.countLimit": {"\n%d of %d task(s) (limit %d)\n", "\n%d de %d tarefa(s) (limite %d)\n"},
	"cli.render.count":      {"\n%d task(s)\n", "\n%d tarefa(s)\n"},

	// --- sync (git) ---
	"cli.sync.usage":           {"usage: taskframe sync [init <git-url> [--pull-wins|--push-wins]|status|pull|push]", "uso: taskframe sync [init <git-url> [--pull-wins|--push-wins]|status|pull|push]"},
	"cli.sync.gitMissing":      {"git not found in PATH — install git to use sync", "git não encontrado no PATH — instale o git para usar o sync"},
	"cli.sync.notConfigured":   {"sync not configured — run: taskframe sync init <git-remote-url>", "sync não configurado — rode: taskframe sync init <git-remote-url>"},
	"cli.sync.alreadyInit":     {"sync already initialized (%s) — remove that folder to re-init", "sync já inicializado (%s) — remova essa pasta para reinicializar"},
	"cli.sync.initDone":        {"sync initialized: %s -> %s", "sync inicializado: %s -> %s"},
	"cli.sync.adopted":         {"adopted remote data: %d task(s), %d note(s)", "dados do remoto adotados: %d tarefa(s), %d nota(s)"},
	"cli.sync.bothData":        {"local and remote both have data — re-run: sync init <url> --pull-wins (adopt remote, backs up local) or --push-wins (publish local)", "local e remoto têm dados — rode: sync init <url> --pull-wins (adota o remoto, faz backup do local) ou --push-wins (publica o local)"},
	"cli.sync.upToDate":        {"already up to date", "já está atualizado"},
	"cli.sync.pulled":          {"pulled remote changes", "baixou as mudanças do remoto"},
	"cli.sync.pushed":          {"pushed local changes", "publicou as mudanças locais"},
	"cli.sync.firstPush":       {"published local data to remote", "publicou os dados locais no remoto"},
	"cli.sync.diverged":        {"local and remote both changed since last sync — run 'taskframe sync pull' (adopt remote) or 'taskframe sync push' (publish local)", "local e remoto mudaram desde o último sync — rode 'taskframe sync pull' (adota o remoto) ou 'taskframe sync push' (publica o local)"},
	"cli.sync.remoteEmpty":     {"remote has no data yet — nothing to pull", "o remoto ainda não tem dados — nada a baixar"},
	"cli.sync.backup":          {"local database backed up to %s", "banco local salvo em backup: %s"},
	"cli.sync.authFailed":      {"git authentication failed — make sure you can push to %s (GitHub credential manager or SSH key)", "falha de autenticação do git — confirme que você consegue dar push em %s (gerenciador de credenciais do GitHub ou chave SSH)"},
	"cli.sync.pushRejected":    {"remote moved during sync — run taskframe sync again", "o remoto mudou durante o sync — rode taskframe sync de novo"},
	"cli.sync.status.repo":     {"clone:  %s", "clone:  %s"},
	"cli.sync.status.remote":   {"remote: %s", "remoto: %s"},
	"cli.sync.status.branch":   {"branch: %s", "branch: %s"},
	"cli.sync.status.state":    {"state:  %s", "estado: %s"},
	"cli.sync.status.dirty":    {"(working tree has uncommitted changes)", "(a árvore de trabalho tem mudanças não commitadas)"},
	"cli.sync.status.diverged": {"diverged (pull or push to resolve)", "divergiu (pull ou push para resolver)"},
	"cli.sync.status.toPush":   {"local changes to push", "mudanças locais para publicar"},
	"cli.sync.status.toPull":   {"remote changes to pull", "mudanças do remoto para baixar"},
	"cli.sync.status.clean":    {"up to date", "atualizado"},

	// --- TUI form ---
	"form.title":            {"Title", "Título"},
	"form.project":          {"Project", "Projeto"},
	"form.tags":             {"Tags", "Tags"},
	"form.priority":         {"Priority", "Prioridade"},
	"form.due":              {"Due", "Vencimento"},
	"form.waitUntil":        {"Wait until", "Aguardar até"},
	"form.scheduled":        {"Scheduled", "Agendada"},
	"form.recur":            {"Recurrence", "Recorrência"},
	"form.hint.project":     {"e.g. home.groceries", "ex: casa.mercado"},
	"form.hint.tags":        {"space-separated", "separadas por espaço"},
	"form.hint.priority":    {"H, M, L or empty", "H, M, L ou vazio"},
	"form.hint.due":         {"today, 3d, fri, 08/15...", "today, 3d, sex, 15/08..."},
	"form.hint.wait":        {"hide until the date", "esconder até a data"},
	"form.hint.scheduled":   {"don't nag before the date", "não cobrar antes da data"},
	"form.hint.recur":       {"daily, weekly, 3d...", "daily, weekly, 3d..."},
	"form.err.titleReq":     {"title is required", "título é obrigatório"},
	"form.err.priority":     {"priority must be H, M, L or empty", "prioridade deve ser H, M, L ou vazia"},
	"form.err.fieldInvalid": {"%s invalid: %s", "%s inválido: %s"},
	"form.field.due":        {"due", "vencimento"},
	"form.field.wait":       {"wait", "aguardar"},
	"form.field.scheduled":  {"scheduled", "agendada"},
	"form.err.recur":        {"invalid recurrence: %s", "recorrência inválida: %s"},
	"form.title.new":        {"New task", "Nova tarefa"},
	"form.title.edit":       {"Edit task", "Editar tarefa"},
	"form.title.newSub":     {"New subtask", "Nova subtarefa"},
	"form.footer":           {"Enter saves · Tab next field · Esc cancels", "Enter salva · Tab próximo campo · Esc cancela"},

	// --- TUI move ---
	"move.err.parentInvalid": {"invalid parent id: %s", "id do pai inválido: %s"},
	"move.err.selfParent":    {"a task cannot be its own parent", "a tarefa não pode ser pai dela mesma"},
	"move.field.project":     {"Project", "Projeto"},
	"move.field.parent":      {"Parent (id)", "Pai (id)"},
	"move.hint.root":         {"empty = root task", "vazio = tarefa raiz"},
	"move.footer":            {"Enter moves · Tab switches · Esc cancels", "Enter move · Tab alterna · Esc cancela"},
	"move.title":             {"Move task", "Mover tarefa"},

	// --- TUI confirm ---
	"confirm.yes":         {"[Y]es", "[S]im"},
	"confirm.no":          {"[N]o (Esc)", "[N]ão (Esc)"},
	"confirm.deleteTitle": {"Delete", "Deletar"},
	"confirm.deleteMsg":   {"Delete task %d — %s?", "Deletar tarefa %d — %s?"},

	// --- TUI note prompt ---
	"notePrompt.footer": {"Ctrl+D saves · Enter new line · Esc cancels", "Ctrl+D salva · Enter nova linha · Esc cancela"},
	"notePrompt.title":  {"New note", "Nova nota"},

	// --- TUI detail ---
	"detail.titleTui":  {"Task %d", "Tarefa %d"},
	"detail.footerTui": {"↑↓ scroll · Esc closes", "↑↓ rola · Esc fecha"},

	// --- read (markdown) view ---
	"read.noNotes": {"_No notes yet._", "_Sem notas ainda._"},
	"help.read.k":  {"read <id>", "read <id>"},
	"help.read.v":  {"render the task's notes as markdown", "renderiza as notas da tarefa em markdown"},

	// --- TUI sidebar ---
	"sb.all":          {"(all)", "(todas)"},
	"tab.all":         {"All", "Todas"},
	"sb.today":        {"Today", "Hoje"},
	"sb.overdue":      {"Overdue", "Atrasadas"},
	"sb.week":         {"Week", "Semana"},
	"sb.active":       {"Active", "Ativas"},
	"sb.next":         {"Next", "Próximas"},
	"sb.waiting":      {"Waiting", "Aguardando"},
	"sb.done":         {"Completed", "Concluídas"},
	"sb.deleted":      {"Deleted", "Deletadas"},
	"sb.title.tasks":  {"Tasks", "Tarefas"},
	"sb.title.of":     {"Tasks: ", "Tarefas: "},
	"sb.title.week":   {"Next 7 days", "Próximos 7 dias"},
	"sb.sec.filters":  {"filters", "filtros"},
	"sb.sec.tags":     {"tags", "tags"},
	"sb.sec.contexts": {"contexts", "contextos"},
	"sb.sec.archive":  {"archive", "arquivo"},
	"panel.projects":  {"Projects", "Projetos"},

	// --- TUI f-key bar ---
	"hint.list": {
		"? help · a add · e edit · d done · S start · / search · [ ] tab · q quit",
		"? ajuda · a nova · e editar · d concluir · S iniciar · / buscar · [ ] aba · q sair",
	},
	"hint.sidebar": {
		"? help · ↑↓ move · enter open/toggle · tab list · q quit",
		"? ajuda · ↑↓ move · enter abre/ativa · tab lista · q sair",
	},
	"hint.search": {"enter apply · esc cancel", "enter aplica · esc cancela"},
	"hint.modal":  {"esc close", "esc fecha"},

	// --- TUI help (F1) ---
	"tuihelp.title":     {"Help", "Ajuda"},
	"tuihelp.tab.v":     {"switch panels", "alterna entre painéis"},
	"tuihelp.move.v":    {"move the cursor", "move o cursor"},
	"tuihelp.fold.v":    {"collapse / expand subtasks", "recolhe / expande subtarefas"},
	"tuihelp.detail.v":  {"view details (notes + history)", "ver detalhes (notas + histórico)"},
	"tuihelp.read.v":    {"read notes as markdown", "ler notas como markdown"},
	"tuihelp.new.v":     {"new task", "nova tarefa"},
	"tuihelp.newsub.v":  {"new subtask under the cursor", "nova subtarefa sob o cursor"},
	"tuihelp.edit.v":    {"edit task", "editar tarefa"},
	"tuihelp.note.v":    {"add note", "adicionar nota"},
	"tuihelp.movedlg.v": {"move (project / parent)", "mover (projeto / pai)"},
	"tuihelp.done.v":    {"complete / reopen task", "concluir / reabrir tarefa"},
	"tuihelp.del.v":     {"delete (with confirmation)", "deletar (com confirmação)"},
	"tuihelp.search.v":  {"search by text", "buscar por texto"},
	"tuihelp.sort.v":    {"cycle sort mode", "alterna a ordenação"},
	"tuihelp.theme.v":   {"cycle theme", "alterna o tema"},
	"tuihelp.start.v":   {"start / stop task", "inicia / para a tarefa"},
	"tuihelp.ctx.v":     {"toggle context (sidebar row)", "ativa/desativa contexto"},
	"tuihelp.lang.v":    {"toggle language (en/pt-br)", "alterna o idioma (en/pt-br)"},
	"tuihelp.undo.v":    {"undo last operation", "desfaz a última operação"},
	"tuihelp.redo.v":    {"redo undone operation", "refaz operação desfeita"},
	"tuihelp.reload.v":  {"reload", "recarregar"},
	"tuihelp.quit.v":    {"quit", "sair"},
	"tuihelp.footer":    {"CLI: taskframe add/list/done/del/note/undo · any key closes", "CLI: taskframe add/list/done/del/note/undo · qualquer tecla fecha"},

	// --- TUI task list ---
	"tasklist.empty": {" no tasks — F2 to add", " nenhuma tarefa — F2 para adicionar"},

	// --- TUI app status ---
	"app.taskUpdated":        {"task %d updated", "tarefa %d atualizada"},
	"app.taskCreated":        {"task %d created", "tarefa %d criada"},
	"app.noteAdded":          {"note added to task %d", "nota adicionada à tarefa %d"},
	"app.taskMoved":          {"task %d moved", "tarefa %d movida"},
	"app.taskDeleted":        {"task %d deleted (u reverts)", "tarefa %d deletada (u desfaz)"},
	"app.undone":             {"undone: %s", "desfeito: %s"},
	"app.redone":             {"redone: %s", "refeito: %s"},
	"app.theme":              {"theme: %s", "tema: %s"},
	"app.sort":               {"sort: %s", "ordenação: %s"},
	"app.taskDoneRecur":      {"task %d completed · recurrence created %d (due %s)", "tarefa %d concluída · recorrência criou %d (vence %s)"},
	"app.taskDone":           {"task %d completed", "tarefa %d concluída"},
	"app.taskReopened":       {"task %d reopened", "tarefa %d reaberta"},
	"app.taskDeletedRestore": {"task deleted — use u to restore", "tarefa deletada — use u para restaurar"},
	"app.windowSmall":        {"Window too small for taskframe (min. 60x12).\nResize the terminal or press q to quit.\n", "Janela muito pequena para o taskframe (mín. 60x12).\nRedimensione o terminal ou pressione q para sair.\n"},
	"app.taskCount":          {" %d task(s)", " %d tarefa(s)"},
	"app.searchInfo":         {" · search: %q (F7 clears)", " · busca: %q (F7 limpa)"},
	"app.searchPrompt":       {"Search: ", "Busca: "},
	"app.lang":               {"language: %s", "idioma: %s"},
	"app.taskStarted":        {"task %d started", "tarefa %d iniciada"},
	"app.taskStopped":        {"task %d stopped", "tarefa %d parada"},
	"app.ctxActive":          {"context: %s", "contexto: %s"},
	"app.ctxCleared":         {"context cleared", "contexto desativado"},
}
