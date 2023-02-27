package ui

import (
	"fmt"
	"math"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	plot "github.com/chriskim06/bubble-plot"
	"github.com/chriskim06/kubectl-topui/internal/config"
)

var (
	plotStyle = Border.Copy().Margin(0, 1).Padding(0, 1)
	ph, pv    = plotStyle.GetFrameSize()
)

type Graphs struct {
	Height  int
	Width   int
	conf    config.Colors
	name    string
	cpuData map[string][][]float64
	memData map[string][][]float64
	cpuMax  float64
	memMax  float64
	cpuMin  float64
	memMin  float64
	style   lipgloss.Style
	cpuPlot *plot.Model
	memPlot *plot.Model
}

func NewGraphs(conf config.Colors) *Graphs {
	cpuPlot := plot.New()
	memPlot := plot.New()
	cpuPlot.Styles.Container = plotStyle
	memPlot.Styles.Container = plotStyle
	return &Graphs{
		conf:    conf,
		style:   Border.Copy().Align(lipgloss.Top).BorderForeground(Adaptive.Copy().GetForeground()),
		cpuPlot: cpuPlot,
		memPlot: memPlot,
	}
}

func (g Graphs) Init() tea.Cmd {
	return nil
}

func (g *Graphs) Update(msg tea.Msg) (Graphs, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		g.SetSize(msg.Width, msg.Height)
		m := tea.WindowSizeMsg{
			Width:  (msg.Width / 2) - ph,
			Height: msg.Height - pv,
		}
		//         g.cpuPlot.Styles.Container.MaxHeight(msg.Height - 5).Height(msg.Height - 5)
		//         g.memPlot.Styles.Container.MaxHeight(msg.Height - 5).Height(msg.Height - 5)
		g.cpuPlot.Update(m)
		g.memPlot.Update(m)
	case tickMsg:
		g.updateData(msg.name, msg.cpuData, msg.memData)
		g.cpuPlot.Title = fmt.Sprintf("CPU - %s", g.name)
		g.memPlot.Title = fmt.Sprintf("MEM - %s", g.name)
		g.cpuPlot.Update(plot.GraphUpdateMsg{
			Data: g.cpuData[g.name],
		})
		g.memPlot.Update(plot.GraphUpdateMsg{
			Data: g.memData[g.name],
		})
	}
	return *g, nil
}

func (g *Graphs) View() string {
	return lipgloss.JoinHorizontal(lipgloss.Top, g.cpuPlot.View(), g.memPlot.View())
}

func (g *Graphs) updateData(name string, cpuData, memData map[string][][]float64) {
	g.name = name
	g.cpuData = cpuData
	g.memData = memData
	g.cpuMax, g.memMax = 0, 0
	g.cpuMin, g.memMin = math.MaxFloat64, math.MaxFloat64
	for _, metrics := range g.cpuData[g.name] {
		for _, value := range metrics {
			if value < g.cpuMin {
				g.cpuMin = value
			}
			if value > g.cpuMax {
				g.cpuMax = value
			}
		}
	}
	for _, metrics := range g.memData[g.name] {
		for _, value := range metrics {
			if value < g.memMin {
				g.memMin = value
			}
			if value > g.memMax {
				g.memMax = value
			}
		}
	}
}

// func (g Graphs) plot(data [][]float64, caption string, o ...asciigraph.Option) string {
//     options := []asciigraph.Option{
//         asciigraph.Width(0),
//         asciigraph.Height(g.Height - 7),
//         asciigraph.AxisColor(g.graphColor),
//         asciigraph.LabelColor(g.graphColor),
//     }
//     options = append(options, o...)
//     return asciigraph.PlotMany(data, options...)
// }

func (g *Graphs) SetSize(width, height int) {
	g.Width = width
	g.Height = height
}
