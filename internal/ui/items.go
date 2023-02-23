package ui

import (
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/chriskim06/kubectl-topui/internal/config"
	"github.com/chriskim06/kubectl-topui/internal/metrics"
	"github.com/chriskim06/kubectl-topui/internal/ui/list"
	"github.com/muesli/reflow/truncate"
)

type listItem string

func (li listItem) FilterValue() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(listItem)
	if !ok {
		return
	}

	line := string(i)
	if m.GetOffset() >= len(line) {
		line = ""
	} else {
		line = line[m.GetOffset():]
		line = truncate.StringWithTail(line, uint(m.Width()), "…")
	}
	if index == m.Index() {
		fmt.Fprintf(w, adaptive.Copy().Background(lipgloss.Color("245")).Bold(true).Render(line))
	} else {
		fmt.Fprintf(w, adaptive.Copy().Render(line))
	}
}

type List struct {
	Height   int
	Width    int
	focused  bool
	conf     config.Colors
	resource metrics.Resource
	content  list.Model
	style    lipgloss.Style
}

func NewList(resource metrics.Resource, conf config.Colors) *List {
	itemList := list.New([]list.Item{}, itemDelegate{}, 0, 0)
	itemList.Styles.Title = lipgloss.NewStyle().Bold(true).Padding(0)
	itemList.Styles.TitleBar = lipgloss.NewStyle().Padding(0)
	return &List{
		resource: resource,
		conf:     conf,
		content:  itemList,
		focused:  true,
		style:    border.Copy(),
	}
}

func (l List) Init() tea.Cmd {
	return nil
}

func (l *List) Update(msg tea.Msg) (List, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		l.content, cmd = l.content.Update(msg)
	case tickMsg:
		header, items := tabStrings(msg.m, l.resource)
		listItems := []list.Item{}
		for _, item := range items {
			listItems = append(listItems, listItem(item))
		}
		l.content.Title = header
		l.content.SetItems(listItems)
	}
	return *l, cmd
}

func (l List) View() string {
	if l.focused {
		l.style.BorderForeground(toColor(string(l.conf.Selected)))
	} else {
		l.style.BorderForeground(adaptive.Copy().GetForeground())
	}
	return l.style.Render(l.content.View())
}

func (l *List) SetSize(width, height int) {
	l.Width = width
	l.Height = height
	l.style = l.style.Width(l.Width).Height(l.Height).Padding(0, 1)
	v, h := l.style.GetFrameSize()
	l.content.Styles.TitleBar.Width(l.Width - h).MaxWidth(l.Width - h)
	l.content.SetSize(l.Width-h, l.Height-v)
}

func (l List) GetSelected() string {
	current := l.content.SelectedItem().(listItem)
	sections := strings.Fields(string(current))
	x := 0
	if l.resource == metrics.POD {
		x = 1
	}
	return sections[x]
}
