package ui

import (
	"log"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/chriskim06/asciigraph"
	"github.com/chriskim06/kubectl-ptop/internal/config"
	"github.com/chriskim06/kubectl-ptop/internal/metrics"
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
	}
	return app
}

func (a App) Init() tea.Cmd {
	return a.tickCmd()
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
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c":
			return a, tea.Quit
		case "q":
			if !a.itemsFocused {
				a.itemsFocused = true
				a.yamlPane.SetContent("")
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
					a.yamlPane.SetContent(output)
				} else {
					a.yamlPane.SetContent(err.Error())
				}
			}
		case "j", "k", "h", "l":
			// figure out selected item and set it on items and graphs
			if a.itemsFocused {
				cmds = a.updatePanes(msg)
			} else {
				a.yamlPane, cmd = a.yamlPane.Update(msg)
				cmds = append(cmds, cmd)
			}
		}
	case tickMsg:
		// update items and graphs
		a.ready = true
		cmds = append(cmds, a.updatePanes(msg)...)
		cmds = append(cmds, a.tickCmd())
	}
	return a, tea.Batch(cmds...)
}

func (a *App) updatePanes(msg tea.Msg) []tea.Cmd {
	var cmd tea.Cmd
	cmds := []tea.Cmd{}
	a.itemsPane, cmd = a.itemsPane.Update(msg)
	cmds = append(cmds, cmd)
	a.setCurrent()
	a.graphsPane, cmd = a.graphsPane.Update(msg)
	cmds = append(cmds, cmd)
	return cmds
}

func (a App) View() string {
	if !a.ready || !a.sizeReady {
		return "Initializing..."
	}
	//     bottom := lipgloss.JoinHorizontal(lipgloss.Top, a.itemsPane.View(), a.yamlPane.View())
	// return lipgloss.JoinVertical(lipgloss.Top, a.graphsPane.View(), bottom)
	//     graphs := lipgloss.NewStyle().Render(a.graphsPane.View())
	return lipgloss.JoinVertical(lipgloss.Top, a.graphsPane.View(), a.itemsPane.View())
}

type tickMsg struct {
	m       []metrics.MetricsValues
	name    string
	cpuData [][]float64
	memData [][]float64
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
		log.Fatal(err)
	}
	for _, metric := range m {
		name := metric.Name
		a.graphUpkeep(name)
		a.cpuData[name][0] = append(a.cpuData[name][0], float64(metric.CPULimit.MilliValue()))
		a.cpuData[name][1] = append(a.cpuData[name][1], float64(metric.CPUCores.MilliValue()))
		a.memData[name][0] = append(a.memData[name][0], float64(metric.MemLimit.Value()/(1024*1024)))
		a.memData[name][1] = append(a.memData[name][1], float64(metric.MemCores.Value()/(1024*1024)))
	}
	a.data = m
	if a.current == "" {
		a.current = m[0].Name
		a.graphsPane.name = a.current
	}
	return tickMsg{
		m:       m,
		cpuData: a.cpuData[a.current],
		memData: a.memData[a.current],
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
	current := a.itemsPane.content.SelectedItem().(listItem)
	sections := strings.Fields(string(current))
	x := 0
	if a.resource == metrics.POD {
		x = 1
	}
	a.current = sections[x]
	a.graphsPane.name = a.current
	a.graphsPane.cpuData = a.cpuData[a.current]
	a.graphsPane.memData = a.memData[a.current]
}
