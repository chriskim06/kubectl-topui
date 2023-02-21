package ui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/chriskim06/kubectl-ptop/internal/config"
	"github.com/muesli/reflow/padding"
	"github.com/muesli/reflow/wrap"
)

type Info struct {
	Height  int
	Width   int
	focused bool
	conf    config.Colors
	content viewport.Model
	style   lipgloss.Style
}

func NewInfo(conf config.Colors) *Info {
	return &Info{
		conf:  conf,
		style: border.Copy(),
	}
}

func (i Info) Init() tea.Cmd {
	return nil
}

func (i *Info) Update(msg tea.Msg) (Info, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		var cmd tea.Cmd
		i.content, cmd = i.content.Update(msg)
		return *i, cmd
	}
	return *i, nil
}

func (i Info) View() string {
	if i.focused {
		i.style.BorderForeground(toColor(string(i.conf.Selected)))
	} else {
		i.style.BorderForeground(adaptive.Copy().GetForeground())
	}
	return i.style.Render(i.content.View())
}

func (i *Info) SetContent(s string) {
	v, h := i.style.GetFrameSize()
	i.content = viewport.New(i.Width-h, i.Height-v)
	i.content.SetContent(wrap.String(padding.String(s, uint(i.Width-h)), i.Width-h))
}

func (i *Info) SetSize(width, height int) {
	i.Width = width
	i.Height = height
	i.style = i.style.Width(i.Width).Height(i.Height)
}
