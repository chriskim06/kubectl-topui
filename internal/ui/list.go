package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/chriskim06/kubectl-ptop/internal/config"
	"github.com/chriskim06/kubectl-ptop/internal/metrics"
)

var itemStyle = adaptive.PaddingLeft(2)

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

	fn := itemStyle.Bold(false).Render
	if index == m.Index() {
		fn = func(s string) string {
			return itemStyle.Bold(true).Render(s)
		}
	}

	fmt.Fprint(w, fn(string(i)))
}

type List struct {
	Height   int
	Width    int
	focused  bool
	conf     config.Colors
	resource metrics.Resource
	data     []metrics.MetricsValues
	content  list.Model
}

func NewList(resource metrics.Resource, conf config.Colors) *List {
	return &List{
		resource: resource,
		conf:     conf,
		content:  list.New([]list.Item{}, itemDelegate{}, 0, 0),
	}
}

func (l List) Init() tea.Cmd {
	return nil
}

func (l List) Update(msg tea.Msg) (List, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		l.content, cmd = l.content.Update(msg)
		return l, cmd
	case tickMsg:
		l.data = msg.m
		header, items := tabStrings(l.data, l.resource)
		listItems := []list.Item{}
		for _, item := range items {
			listItems = append(listItems, listItem(item))
		}
		l.content.Title = header
		l.content.Styles.Title = lipgloss.NewStyle().Bold(true)
		l.content.SetItems(listItems)
	}
	return l, nil
}

func (l List) View() string {
	style := lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder())
	if l.focused {
		style = style.BorderBackground(lipgloss.Color(l.conf.Selected))
	} else {
		style = style.BorderBackground(adaptive.GetBackground())
	}
	return lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).Margin(0).Width(l.Width).Height(l.Height).Render(l.content.View())
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
