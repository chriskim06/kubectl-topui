package ui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/chriskim06/asciigraph"
	"github.com/chriskim06/kubectl-ptop/internal/config"
	"github.com/chriskim06/kubectl-ptop/internal/metrics"
	"github.com/muesli/reflow/padding"
	"github.com/muesli/reflow/wrap"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/top"
)

type App struct {
	client     metrics.MetricsClient
	conf       config.Colors
	data       []metrics.MetricsValues
	cpuData    map[string][][]float64
	memData    map[string][][]float64
	resource   metrics.Resource
	options    interface{}
	current    string
	tick       time.Ticker
	interval   time.Duration
	ready      bool
	sizeReady  bool
	err        error
	itemsPane  List
	graphsPane Graphs
	yamlPane   viewport.Model
}

func New(resource metrics.Resource, interval int, options interface{}, showManagedFields bool, flags *genericclioptions.ConfigFlags) *App {
	conf := config.GetTheme()
	items := NewList(resource, conf)
	items.content.SetShowStatusBar(false)
	items.content.SetFilteringEnabled(false)
	items.content.SetShowHelp(false)
	graphColor := asciigraph.White
	if !lipgloss.HasDarkBackground() {
		graphColor = asciigraph.Black
	}
	app := &App{
		client:     metrics.New(flags, showManagedFields),
		conf:       conf,
		resource:   resource,
		options:    options,
		cpuData:    map[string][][]float64{},
		memData:    map[string][][]float64{},
		interval:   time.Duration(interval) * time.Second,
		itemsPane:  *items,
		graphsPane: Graphs{conf: conf, graphColor: graphColor},
	}
	return app
}

func (a App) Init() tea.Cmd {
	return a.immediateCmd()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.sizeReady = true
		third := msg.Width / 3
		half := msg.Height / 2
		a.itemsPane.SetSize(msg.Width-third, half)
		a.graphsPane.SetSize(msg.Width, half)
		a.yamlPane = viewport.New(third, half+2)
		a.yamlPane.SetContent(strings.Repeat(" ", a.yamlPane.Width-5))
		a.yamlPane.Style = lipgloss.NewStyle().Border(lipgloss.NormalBorder())
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c":
			return a, tea.Quit
		case "q":
			if !a.itemsPane.focused {
				a.itemsPane.focused = true
				a.yamlPane.SetContent(strings.Repeat(" ", a.yamlPane.Width-5))
				a.yamlPane.Style.BorderForeground(adaptive.GetForeground())
			} else {
				return a, tea.Quit
			}
		case "enter":
			if a.itemsPane.focused {
				a.itemsPane.focused = false
				var output string
				var err error
				if a.resource == metrics.POD {
					output, err = a.client.GetPod(a.itemsPane.GetSelected())
				} else {
					output, err = a.client.GetNode(a.itemsPane.GetSelected())
				}
				if err != nil {
					a.err = err
					return a, tea.Quit
				}
				s := wrap.String(padding.String(output, uint(a.yamlPane.Width-5)), a.yamlPane.Width-5)
				a.yamlPane.SetContent(s)
				a.yamlPane.Style.BorderForeground(toColor(string(a.conf.Selected)))
			}
		case "j", "k", "up", "down":
			var cmd tea.Cmd
			if !a.itemsPane.focused {
				a.yamlPane, cmd = a.yamlPane.Update(msg)
				cmds = append(cmds, cmd)
				break
			}

			a.itemsPane.content, cmd = a.itemsPane.content.Update(msg)
			cmds = append(cmds, cmd)
			a.current = a.itemsPane.GetSelected()
			a.graphsPane.name = a.current
			a.graphsPane.cpuData = a.cpuData
			a.graphsPane.memData = a.memData
		}
	case tickMsg:
		// update items and graphs
		if msg.err != nil {
			a.err = msg.err
			return a, tea.Quit
		}
		a.ready = true
		msg.name = msg.m[0].Name
		if a.itemsPane.content.SelectedItem() != nil {
			msg.name = a.itemsPane.GetSelected()
		}
		var itemsCmd, graphsCmd tea.Cmd
		a.itemsPane, itemsCmd = a.itemsPane.Update(msg)
		a.graphsPane, graphsCmd = a.graphsPane.Update(msg)
		cmds = append(cmds, a.tickCmd(), itemsCmd, graphsCmd)
	}
	return a, tea.Batch(cmds...)
}

func (a App) View() string {
	if !a.ready || !a.sizeReady {
		return "Initializing..."
	}
	//     items := lipgloss.NewStyle().Height(a.itemsPane.Height).Width(a.itemsPane.Width).Render(a.itemsPane.View())
	//     yaml := lipgloss.NewStyle().Height(a.yamlPane.Height).Width(a.yamlPane.Width).Render(a.yamlPane.View())
	//     bottom := lipgloss.JoinHorizontal(lipgloss.Top, items, yaml)
	//     bottom := lipgloss.JoinHorizontal(lipgloss.Top, a.itemsPane.View(), a.yamlPane.View())
	yaml := lipgloss.NewStyle().Height(a.yamlPane.Height).Width(a.yamlPane.Width).Render(a.yamlPane.View())
	bottom := lipgloss.JoinHorizontal(lipgloss.Top, a.itemsPane.View(), yaml)
	return lipgloss.JoinVertical(lipgloss.Top, a.graphsPane.View(), bottom)
}

type tickMsg struct {
	m       []metrics.MetricsValues
	name    string
	err     error
	cpuData map[string][][]float64
	memData map[string][][]float64
}

func (a *App) immediateCmd() tea.Cmd {
	return func() tea.Msg {
		return a.updateData()
	}
}

func (a *App) tickCmd() tea.Cmd {
	return tea.Tick(a.interval, func(t time.Time) tea.Msg {
		return a.updateData()
	})
}

func (a *App) updateData() tickMsg {
	var err error
	var m []metrics.MetricsValues
	if a.resource == metrics.POD {
		m, err = a.client.GetPodMetrics(a.options.(*top.TopPodOptions))
	} else {
		m, err = a.client.GetNodeMetrics(a.options.(*top.TopNodeOptions))
	}
	if err != nil {
		return tickMsg{err: err}
	}
	a.data = m
	for _, metric := range m {
		name := metric.Name
		if a.cpuData[name] == nil || a.memData[name] == nil {
			a.cpuData[name] = [][]float64{{}, {}}
			a.memData[name] = [][]float64{{}, {}}
		} else {
			if len(a.cpuData[name][0]) == 100 {
				a.cpuData[name][0] = a.cpuData[name][0][1:]
			}
			if len(a.cpuData[name][1]) == 100 {
				a.cpuData[name][1] = a.cpuData[name][1][1:]
			}
			if len(a.memData[name][0]) == 100 {
				a.memData[name][0] = a.memData[name][0][1:]
			}
			if len(a.memData[name][1]) == 100 {
				a.memData[name][1] = a.memData[name][1][1:]
			}
		}
		a.cpuData[name][0] = append(a.cpuData[name][0], float64(metric.CPULimit.MilliValue()))
		a.cpuData[name][1] = append(a.cpuData[name][1], float64(metric.CPUCores.MilliValue()))
		a.memData[name][0] = append(a.memData[name][0], float64(metric.MemLimit.Value()/(1024*1024)))
		a.memData[name][1] = append(a.memData[name][1], float64(metric.MemCores.Value()/(1024*1024)))
	}
	return tickMsg{
		m:       m,
		cpuData: a.cpuData,
		memData: a.memData,
	}
}
