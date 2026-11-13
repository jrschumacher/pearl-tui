package main

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

// opDoneMsg carries the result of a fake check or connect.
type opDoneMsg struct {
	op  string
	err error
}

type model struct {
	w         *wizard
	svc       Service
	form      *huh.Form
	width     int
	height    int
	cancelled bool
}

func newModel(svc Service, store Store) *model {
	return &model{w: newWizard(store), svc: svc}
}

func (m *model) Init() tea.Cmd { return nil }

func (m *model) rebuild() tea.Cmd {
	m.form = m.buildForm()
	if m.form == nil {
		return nil
	}
	return m.form.Init()
}

func (m *model) run(op string) tea.Cmd {
	svc := m.svc
	switch op {
	case "check":
		ep := m.w.endpoint()
		return func() tea.Msg { return opDoneMsg{op, svc.Check(ep)} }
	case "connect":
		p := m.w.profile()
		return func() tea.Msg { return opDoneMsg{op, svc.Connect(p)} }
	}
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if m.form == nil {
			return m, m.rebuild()
		}
		m.form.WithWidth(m.formWidth()).WithHeight(m.formHeight())

	case opDoneMsg:
		m.w.finish(msg.op, msg.err)
		return m, m.rebuild()

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		}
		if m.w.busy != "" {
			return m, nil
		}
		if m.w.step == stepDone {
			switch msg.String() {
			case "enter", "q", "esc":
				return m, tea.Quit
			}
			return m, nil
		}
		switch msg.String() {
		case "esc":
			m.cancelled = true
			return m, tea.Quit
		case "ctrl+b":
			if m.w.back() {
				return m, m.rebuild()
			}
			return m, nil
		}
	}

	if m.form == nil {
		return m, nil
	}
	f, cmd := m.form.Update(msg)
	m.form = f.(*huh.Form)
	if m.form.State == huh.StateCompleted {
		if op := m.w.complete(); op != "" {
			return m, m.run(op)
		}
		return m, m.rebuild()
	}
	return m, cmd
}
