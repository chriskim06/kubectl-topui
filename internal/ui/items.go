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
	"github.com/chriskim06/kubectl-topui/internal/ui/utils"
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
		line = utils.Truncate(line, m.Width())
	}
	if index == m.Index() {
		fmt.Fprintf(w, Adaptive.Copy().Background(lipgloss.Color("245")).Bold(true).Render(line))
	} else {
		fmt.Fprintf(w, Adaptive.Copy().Render(line))
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
	maxLen   int
}

func NewList(resource metrics.Resource, conf config.Colors) *List {
	itemList := list.New([]list.Item{}, itemDelegate{}, 0, 0)
	itemList.ItemNamePlural = resource.LowerCase()
	itemList.Styles.Title = lipgloss.NewStyle().Bold(true).Padding(0)
	itemList.Styles.TitleBar = lipgloss.NewStyle().Padding(0)
	return &List{
		resource: resource,
		conf:     conf,
		content:  itemList,
		focused:  true,
		style:    Border.Copy().Padding(0, 1),
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
		header, items := utils.TabStrings(msg.m, l.resource)
		max := 0
		listItems := []list.Item{}
		for _, item := range items {
			listItems = append(listItems, listItem(item))
			if len(item) > max {
				max = len(item)
			}
		}
		l.maxLen = max
		l.content.Title = header
		l.content.SetItems(listItems)
	}
	return *l, cmd
}

func (l List) View() string {
	if l.focused {
		l.style.BorderForeground(lipgloss.Color(fmt.Sprintf("%d", l.conf.Selected)))
	} else {
		l.style.BorderForeground(Adaptive.Copy().GetForeground())
	}
	l.style.Width(l.Width).Height(l.Height)
	h, v := l.style.GetFrameSize()
	l.content.Styles.TitleBar.Width(l.Width - h)
	l.content.SetSize(l.Width-h, l.Height-v)
	return l.style.Render(l.content.View())
}

func (l *List) SetSize(width, height int) {
	l.Width = width
	l.Height = height
}

func (l List) GetSelected() string {
	sections := l.getSections()
	x := 0
	if l.resource == metrics.POD {
		x = 1
	}
	return sections[x]
}

func (l List) GetNamespace() string {
	sections := l.getSections()
	return sections[0]
}

func (l List) getSections() []string {
	current := l.content.SelectedItem().(listItem)
	return strings.Fields(string(current))
}
