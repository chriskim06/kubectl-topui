package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/chriskim06/asciigraph"
	"github.com/chriskim06/kubectl-topui/internal/config"
	"github.com/chriskim06/kubectl-topui/internal/metrics"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/top"
)

type App struct {
	client     metrics.MetricsClient
	conf       config.Colors
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
	height     int
	width      int
	itemsPane  List
	graphsPane Graphs
	infoPane   Info
	loading    *spinner.Model
}

func New(resource metrics.Resource, interval int, options interface{}, showManagedFields bool, flags *genericclioptions.ConfigFlags) *App {
	conf := config.GetTheme()
	items := NewList(resource, conf)
	graphColor := asciigraph.White
	if !lipgloss.HasDarkBackground() {
		graphColor = asciigraph.Black
	}
	loading := spinner.New(spinner.WithSpinner(spinner.Dot))
	graphs := NewGraphs(conf, graphColor)
	var ns *string
	var allNs *bool
	if resource == metrics.POD {
		allNamespaces := options.(*top.TopPodOptions).AllNamespaces
		allNs = &allNamespaces
		ns = flags.Namespace
	}
	app := &App{
		client:     metrics.New(flags, showManagedFields, ns, allNs),
		conf:       conf,
		resource:   resource,
		options:    options,
		cpuData:    map[string][][]float64{},
		memData:    map[string][][]float64{},
		interval:   time.Duration(interval) * time.Second,
		itemsPane:  *items,
		graphsPane: *graphs,
		infoPane:   *NewInfo(conf),
		loading:    &loading,
	}
	return app
}

func (a App) Init() tea.Cmd {
	return tea.Batch(a.loading.Tick, a.updateData)
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.sizeReady = true
		a.height = msg.Height
		a.width = msg.Width
		half := msg.Height / 2
		thirdRounded := (msg.Width / 3) + (msg.Width % 3)
		a.graphsPane.SetSize(msg.Width, half)
		a.itemsPane.SetSize(msg.Width-thirdRounded-5, half)
		a.infoPane.SetSize(thirdRounded, half)
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c":
			return a, tea.Quit
		case "q":
			if a.err != nil {
				return a, tea.Quit
			}
			if !a.itemsPane.focused {
				a.itemsPane.focused = true
				a.infoPane.focused = false
				a.infoPane.SetContent("")
			} else {
				return a, tea.Quit
			}
		case "enter":
			if !a.ready || !a.sizeReady {
				return a, nil
			}
			if a.itemsPane.focused {
				a.itemsPane.focused = false
				var output string
				var err error
				if a.resource == metrics.POD {
					output, err = a.client.GetPod(a.itemsPane.GetSelected(), a.itemsPane.GetNamespace())
				} else {
					output, err = a.client.GetNode(a.itemsPane.GetSelected())
				}
				if err != nil {
					a.err = err
					return a, nil
				}
				a.infoPane.focused = true
				a.infoPane.SetContent(output)
			}
		case "j", "k", "h", "l", "g", "G", "up", "down", "left", "right", "tab", "shift+tab", "home", "end", "pgup", "pgdown":
			if !a.ready || !a.sizeReady {
				return a, nil
			}
			if !a.itemsPane.focused {
				a.infoPane, cmd = a.infoPane.Update(msg)
				return a, cmd
			}

			a.itemsPane, cmd = a.itemsPane.Update(msg)
			cmds = append(cmds, cmd)
			a.current = a.itemsPane.GetSelected()
			a.graphsPane.updateData(a.current, a.cpuData, a.memData)
		}
	case tickMsg:
		if msg.err != nil {
			a.err = msg.err
			return a, nil
		}
		a.ready = true
		if a.itemsPane.content.SelectedItem() != nil {
			msg.name = a.itemsPane.GetSelected()
		}
		var itemsCmd, graphsCmd tea.Cmd
		a.itemsPane, itemsCmd = a.itemsPane.Update(msg)
		a.graphsPane, graphsCmd = a.graphsPane.Update(msg)
		cmds = append(cmds, itemsCmd, graphsCmd, a.tickCmd())
	case spinner.TickMsg:
		if a.ready && a.sizeReady {
			a.loading = nil
			half := a.height / 2
			thirdRounded := (a.width / 3) + (a.width % 3)
			a.itemsPane.SetSize(a.width-thirdRounded-5, half)
			return a, nil
		}
		*a.loading, cmd = a.loading.Update(msg)
		return a, cmd
	}
	return a, tea.Batch(cmds...)
}

func (a App) View() string {
	if a.err != nil {
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, errStyle.Width(a.width/2).Height(a.height/2).Render("ERROR:\n\n"+a.err.Error()))
	}
	if !a.ready || !a.sizeReady {
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, a.loading.View()+"Initializing...")
	}
	bottom := lipgloss.JoinHorizontal(lipgloss.Top, a.itemsPane.View(), a.infoPane.View())
	return lipgloss.JoinVertical(lipgloss.Top, a.graphsPane.View(), bottom)
}

type tickMsg struct {
	m       []metrics.MetricValue
	name    string
	err     error
	cpuData map[string][][]float64
	memData map[string][][]float64
}

func (a *App) tickCmd() tea.Cmd {
	return tea.Tick(a.interval, func(t time.Time) tea.Msg {
		return a.updateData()
	})
}

func (a *App) updateData() tea.Msg {
	var err error
	var m []metrics.MetricValue
	if a.resource == metrics.POD {
		m, err = a.client.GetPodMetrics(a.options.(*top.TopPodOptions))
	} else {
		m, err = a.client.GetNodeMetrics(a.options.(*top.TopNodeOptions))
	}
	if err != nil {
		fmt.Println(err)
		return tickMsg{err: err}
	}
	for _, metric := range m {
		name := metric.Name
		if a.cpuData[name] == nil || a.memData[name] == nil {
			a.cpuData[name] = [][]float64{{}, {}}
			a.memData[name] = [][]float64{{}, {}}
		} else if len(a.cpuData[name][0]) == 50 {
			a.cpuData[name][0] = a.cpuData[name][0][1:]
			a.cpuData[name][1] = a.cpuData[name][1][1:]
			a.memData[name][0] = a.memData[name][0][1:]
			a.memData[name][1] = a.memData[name][1][1:]
		}
		a.cpuData[name][0] = append(a.cpuData[name][0], float64(metric.CPULimit.MilliValue()))
		a.cpuData[name][1] = append(a.cpuData[name][1], float64(metric.CPUCores.MilliValue()))
		a.memData[name][0] = append(a.memData[name][0], float64(metric.MemLimit))
		a.memData[name][1] = append(a.memData[name][1], float64(metric.MemCores))
	}
	return tickMsg{
		m:       m,
		name:    m[0].Name,
		cpuData: a.cpuData,
		memData: a.memData,
	}
}
