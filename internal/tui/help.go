package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Help is the F1 keybinding reference.
type Help struct{}

func (hp *Help) Update(msg tea.Msg) (Modal, tea.Cmd) {
	if _, ok := msg.(tea.KeyMsg); ok {
		return hp, func() tea.Msg { return modalCancelMsg{} }
	}
	return hp, nil
}

func (hp *Help) View(th Theme, w, h int) string {
	rows := [][2]string{
		{"Tab", "alterna entre painéis"},
		{"↑↓ / jk", "move o cursor"},
		{"←→ / hl", "recolhe / expande subtarefas"},
		{"Enter, F3", "ver detalhes (notas + histórico)"},
		{"F2, a", "nova tarefa"},
		{"F6, s", "nova subtarefa sob o cursor"},
		{"F4, e", "editar tarefa"},
		{"F5, n", "adicionar nota"},
		{"F9, d, Espaço", "concluir / reabrir tarefa"},
		{"F8, x, Del", "deletar (com confirmação)"},
		{"F7, /", "buscar por texto"},
		{"u", "desfazer última operação"},
		{"r", "recarregar"},
		{"F10, q", "sair"},
	}
	var lines []string
	lines = append(lines, "")
	for _, r := range rows {
		lines = append(lines, " "+th.TitleFocus.Render(padRowPlain(r[0], 16))+th.Text.Render(r[1]))
	}
	lines = append(lines, "", " "+th.Dim.Render("CLI: taskframe add/list/done/del/note/undo · qualquer tecla fecha"))

	bw := 64
	if bw > w-4 {
		bw = w - 4
	}
	return drawBox(th, "Ajuda", lines, bw, len(lines)+3, true)
}
