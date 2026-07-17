# TaskFrame classic TUI demo — regenerate with `vhs demo-classic.tape`.
# CI renders this automatically (see .github/workflows/vhs.yml).
Output demo-classic.gif

Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 720
Set Padding 22
Set TypingSpeed 55ms

# Seed a throwaway database so the demo is self-contained. The fixed past due
# date keeps the Overdue tab lit no matter when the tape renders.
Hide
Type "export TASKFRAME_DB=$(mktemp -u)" Enter
Type "export TASKFRAME_THEME=charm" Enter
Type "taskframe add 'buy milk' pro:home.groceries +urgent due:fri prio:H" Enter
Type "taskframe add 'review the quarterly report' pro:work prio:M due:today" Enter
Type "taskframe add 'pay the electricity bill' pro:home due:2026-01-01" Enter
Type "taskframe add 'write the sync tests' pro:work sub:2" Enter
Type "taskframe note 2 'waiting on **Marcos** for the final numbers'" Enter
Type "taskframe start 2" Enter
Type "clear" Enter
Show

# The classic interface: wordmark, report tabs, open columns, checklist marks.
Type "taskframe classic" Enter
Sleep 2.5s

# Jump between report tabs by number: Overdue (lit red) and Active (started
# task with its elapsed time), then back to All with ].
Type "3"
Sleep 1.8s
Type "5"
Sleep 1.5s
Type "]"
Sleep 800ms
Type "1"
Sleep 1.2s

# Open the detail of the task under the cursor (the started report, first by
# urgency) — the dialog floats over the dimmed live view, showing the subtask
# progress bar and the note rendered as markdown.
Enter
Sleep 2.5s
Escape
Sleep 800ms

Type "q"
Sleep 600ms
