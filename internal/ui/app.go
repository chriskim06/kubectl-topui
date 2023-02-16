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

var adaptive = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "0", Dark: "15"})

type App struct {
	client    metrics.MetricsClient
	conf      config.Colors
	data      []metrics.MetricsValues
	cpuData   map[string][][]float64
	memData   map[string][][]float64
	resource  metrics.Resource
	options   interface{}
	current   string
	tick      time.Ticker
	interval  time.Duration
	ready     bool
	sizeReady bool
	err       error

	height       int
	width        int
	itemsFocused bool
	itemsPane    List
	graphsPane   Graphs
	yamlPane     viewport.Model
}

func New(resource metrics.Resource, interval int, options interface{}, flags *genericclioptions.ConfigFlags) *App {
	conf := config.GetTheme()
	items := NewList(resource, conf)
	items.content.SetShowStatusBar(false)
	items.content.SetFilteringEnabled(false)
	items.content.SetShowHelp(false)
	graphColor := asciigraph.White
	if !lipgloss.HasDarkBackground() {
		graphColor = asciigraph.Black
	}
	yamlPane := viewport.New(0, 0)
	yamlPane.Style = lipgloss.NewStyle().Border(lipgloss.NormalBorder())
	app := &App{
		client:       metrics.New(flags),
		conf:         conf,
		resource:     resource,
		options:      options,
		cpuData:      map[string][][]float64{},
		memData:      map[string][][]float64{},
		interval:     time.Duration(interval) * time.Second,
		itemsFocused: true,
		itemsPane:    *items,
		graphsPane:   Graphs{conf: conf, graphColor: graphColor},
		yamlPane:     yamlPane,
	}
	return app
}

func (a App) Init() tea.Cmd {
	return a.immediateCmd()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.sizeReady = true
		a.width = msg.Width
		a.height = msg.Height
		a.itemsPane.Width = msg.Width * (2 / 3)
		a.itemsPane.Height = msg.Height / 2
		a.itemsPane.content.SetWidth(msg.Width * (2 / 3))
		a.itemsPane.content.SetHeight(msg.Height / 2)
		a.graphsPane.Width = msg.Width
		a.graphsPane.Height = msg.Height / 2
		a.yamlPane = viewport.New((msg.Width/3)+1, msg.Height/2+2)
		a.yamlPane.SetContent(strings.Repeat(" ", a.yamlPane.Width))
		a.yamlPane.Style = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Height(a.yamlPane.Height).Width(a.yamlPane.Width)
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c":
			return a, tea.Quit
		case "q":
			if !a.itemsFocused {
				a.itemsFocused = true
				a.yamlPane.SetContent(strings.Repeat(" ", a.yamlPane.Width))
			} else {
				return a, tea.Quit
			}
		case "enter":
			if a.itemsFocused {
				a.itemsFocused = false
				var output string
				var err error
				if a.resource == metrics.POD {
					output, err = a.client.GetPod(a.current)
				} else {
					output, err = a.client.GetNode(a.current)
				}
				if err != nil {
					a.err = err
					return a, tea.Quit
				}
				s := wrap.String(padding.String(output, uint(a.yamlPane.Width)), a.yamlPane.Width)
				a.yamlPane.SetContent(s)
			}
		case "j", "k", "up", "down":
			// figure out selected item and handle updating list cursor
			if !a.itemsFocused {
				a.yamlPane, cmd = a.yamlPane.Update(msg)
				cmds = append(cmds, cmd)
				break
			}

			a.itemsPane.content, cmd = a.itemsPane.content.Update(msg)
			cmds = append(cmds, cmd)
			a.setCurrent()
		}
	case tickMsg:
		// update items and graphs
		if msg.err != nil {
			a.err = msg.err
			return a, tea.Quit
		}
		if !a.ready {
			a.ready = true
		}
		if a.itemsPane.content.SelectedItem() != nil {
			msg.name = a.itemsPane.GetSelected()
		} else {
			msg.name = msg.m[0].Name
		}
		cmds = append(cmds, a.updatePanes(msg)...)
		cmds = append(cmds, a.tickCmd())
	}
	return a, tea.Batch(cmds...)
}

func (a *App) updatePanes(msg tea.Msg) []tea.Cmd {
	cmds := []tea.Cmd{}
	var itemsCmd, graphsCmd tea.Cmd
	a.itemsPane, itemsCmd = a.itemsPane.Update(msg)
	a.graphsPane, graphsCmd = a.graphsPane.Update(msg)
	return append(cmds, itemsCmd, graphsCmd)
}

func (a App) View() string {
	if !a.ready || !a.sizeReady {
		return "Initializing..."
	}
	bottom := lipgloss.JoinHorizontal(lipgloss.Top, a.itemsPane.View(), a.yamlPane.View())
	return lipgloss.JoinVertical(lipgloss.Top, a.graphsPane.View(), bottom)
	// return lipgloss.JoinVertical(lipgloss.Top, a.graphsPane.View(), a.itemsPane.View())
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
		return a.update()
	}
}

func (a *App) tickCmd() tea.Cmd {
	return tea.Tick(a.interval, func(t time.Time) tea.Msg {
		return a.update()
	})
}

func (a *App) update() tickMsg {
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
		a.graphUpkeep(name)
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

func (a *App) graphUpkeep(name string) {
	if a.cpuData[name] == nil || a.memData[name] == nil {
		a.cpuData[name] = [][]float64{{}, {}}
		a.memData[name] = [][]float64{{}, {}}
		return
	}
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

func (a *App) setCurrent() {
	a.current = a.itemsPane.GetSelected()
	a.graphsPane.name = a.current
	a.graphsPane.cpuData = a.cpuData
	a.graphsPane.memData = a.memData
}
