package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/freehold-digital/rwr/internal/reporting"
)

type formTaskDone struct {
	request *reporting.FormReq
	err     error
}

func (m *Model) initForm() tea.Cmd {
	req := m.form
	req.Form.WithTheme(huh.ThemeFunc(func(dark bool) *huh.Styles {
		s := huh.ThemeBase(dark)
		for _, field := range []*huh.FieldStyles{&s.Focused, &s.Blurred} {
			field.Title = style(m.theme.Text).Bold(true)
			field.NoteTitle = style(m.theme.Modal).Bold(true)
			field.Description = style(m.theme.Subtext)
			field.Option = style(m.theme.Text)
			field.SelectedOption = style(m.theme.Accent).Bold(true)
			field.SelectSelector = style(m.theme.Accent).SetString("> ")
			field.ErrorMessage = style(m.theme.Danger)
			field.ErrorIndicator = style(m.theme.Danger).SetString(" *")
			field.TextInput.Text = style(m.theme.Text)
			field.TextInput.Cursor = style(m.theme.Accent).Reverse(true)
		}
		return s
	}))
	// An embedded form must never send tea.Quit to the parent program.
	req.Form.SubmitCmd = nil
	req.Form.CancelCmd = nil
	m.resizeForm()
	init := req.Form.Init()
	if req.Task == nil {
		return init
	}
	return tea.Batch(init, func() tea.Msg {
		return formTaskDone{request: req, err: req.Task(req.Context)}
	})
}

func (m *Model) resizeForm() tea.Cmd {
	m.form.Form.WithWidth(m.promptContentWidth())
	_, cmd := m.form.Form.Update(tea.WindowSizeMsg{Width: m.promptContentWidth(), Height: max(1, m.height-8)})
	return cmd
}

func (m *Model) finishForm(err error) {
	if reporting.TryClaim(m.form.Claim) {
		m.form.Done <- err
	}
	m.form = nil
	m.resumePopup()
}

// Closing an overlay must not hide an interaction still waiting underneath it.
func (m *Model) resumePopup() {
	if m.state == SummaryState {
		return
	}
	if m.secret != nil || m.confirm != nil || m.form != nil || m.command != nil || m.halt != nil {
		m.state = Prompting
	} else {
		m.state = Running
	}
}

func (m *Model) updateForm(msg tea.Msg) tea.Cmd {
	if done, ok := msg.(formTaskDone); ok {
		if done.request == m.form {
			m.finishForm(done.err)
		}
		return nil
	}
	if key, ok := msg.(tea.KeyPressMsg); ok {
		if key.String() == "esc" || key.String() == "ctrl+c" {
			m.finishForm(reporting.ErrPromptCancelled)
			return nil
		}
		if m.form.Task != nil {
			return nil // the OAuth popup waits for the task, not Enter
		}
	}
	_, cmd := m.form.Form.Update(msg)
	switch m.form.Form.State {
	case huh.StateCompleted:
		m.finishForm(nil)
		return nil
	case huh.StateAborted:
		m.finishForm(reporting.ErrPromptCancelled)
		return nil
	}
	return cmd
}
