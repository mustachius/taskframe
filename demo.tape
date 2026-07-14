# TaskFrame demo — regenerate with `vhs demo.tape`.
# CI renders this automatically (see .github/workflows/vhs.yml).
Output demo.gif

Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 720
Set Padding 22
Set TypingSpeed 55ms

# Seed a throwaway database so the demo is self-contained.
# Start in the Dracula theme so the gradient logo and output render in color.
Hide
Type "export TASKFRAME_DB=$(mktemp -u)" Enter
Type "export TASKFRAME_THEME=dracula" Enter
Type "taskframe add 'buy milk' pro:home.groceries +urgent due:fri prio:H" Enter
Type "taskframe add 'review the quarterly report' pro:work prio:M" Enter
Type "taskframe note 1 'waiting on **Marcos** — check the supplier list'" Enter
Type "clear" Enter
Show

# Launch the inline REPL — Dracula theme, gradient logo in color.
Type "taskframe" Enter
Sleep 1.8s

# Render the note as markdown in the scrollback.
Type "read 1" Enter
Sleep 2.2s

# Open the navigable list and inspect a task's detail.
Type "list" Enter
Sleep 1.2s
Down
Sleep 600ms
Enter
Sleep 2.2s
Escape
Escape
Sleep 500ms

Type "/quit" Enter
Sleep 900ms
