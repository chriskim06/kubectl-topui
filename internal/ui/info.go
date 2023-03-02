package ui

import (
	"bytes"
	"fmt"

	"github.com/alecthomas/chroma/quick"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/chriskim06/kubectl-topui/internal/config"
	"github.com/muesli/reflow/padding"
	"github.com/muesli/reflow/wrap"
)

type Info struct {
	Height  int
	Width   int
	focused bool
	yaml    string
	conf    config.Colors
	content viewport.Model
	style   lipgloss.Style
}

func NewInfo(conf config.Colors) *Info {
	return &Info{
		conf:    conf,
		style:   lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).Padding(0),
		content: viewport.New(0, 0),
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
		i.style.BorderForeground(lipgloss.Color(fmt.Sprintf("%d", i.conf.Selected)))
	} else {
		i.style.BorderForeground(Adaptive.Copy().GetForeground())
	}
	return i.style.Render(i.content.View())
}

func (i *Info) SetContent(s string) {
	i.yaml = s
	i.setText()
}

func (i *Info) SetSize(width, height int) {
	i.Width = width - 4
	i.Height = height
	i.style.Width(i.Width).Height(i.Height)
	h, v := i.style.GetFrameSize()
	i.content.Width = i.Width - h
	i.content.Height = i.Height - v
	if i.yaml != "" {
		i.setText()
	}
}

func (i *Info) setText() {
	h, v := i.style.GetFrameSize()
	i.content.Width = i.Width - h
	i.content.Height = i.Height - v
	content := wrap.String(padding.String(i.yaml, uint(i.content.Width)), i.content.Width)
	var b bytes.Buffer
	if err := quick.Highlight(&b, content, "yaml", "terminal256", "friendly"); err == nil {
		i.content.SetContent(b.String())
	} else {
		i.content.SetContent(content)
	}
}
