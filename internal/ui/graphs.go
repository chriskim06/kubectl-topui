package ui

import (
	"fmt"
	"math"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/chriskim06/asciigraph"
	"github.com/chriskim06/kubectl-ptop/internal/config"
)

type Graphs struct {
	Height     int
	Width      int
	conf       config.Colors
	graphColor asciigraph.AnsiColor
	name       string
	cpuData    map[string][][]float64
	memData    map[string][][]float64
	cpuMax     float64
	memMax     float64
	cpuMin     float64
	memMin     float64
	style      lipgloss.Style
}

func NewGraphs(conf config.Colors, graphColor asciigraph.AnsiColor) *Graphs {
	return &Graphs{
		conf:       conf,
		graphColor: graphColor,
		style:      border.Copy().Align(lipgloss.Top).BorderForeground(adaptive.Copy().GetForeground()),
	}
}

func (g Graphs) Init() tea.Cmd {
	return nil
}

func (g *Graphs) Update(msg tea.Msg) (Graphs, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		g.updateData(msg.name, msg.cpuData, msg.memData)
	}
	return *g, nil
}

func (g *Graphs) View() string {
	cpuColors := asciigraph.SeriesColors(asciigraph.ColorNames[string(g.conf.CPULimit)], asciigraph.ColorNames[string(g.conf.CPUUsage)])
	memColors := asciigraph.SeriesColors(asciigraph.ColorNames[string(g.conf.MemLimit)], asciigraph.ColorNames[string(g.conf.MemUsage)])
	cpuPlot := g.plot(g.cpuData[g.name], "CPU", asciigraph.Min(g.cpuMin), asciigraph.Max(g.cpuMax), cpuColors)
	memPlot := g.plot(g.memData[g.name], "MEM", asciigraph.Min(g.memMin), asciigraph.Max(g.memMax), memColors)
	g.style = g.style.
		MaxWidth(g.Width / 2).
		MaxHeight(g.Height).
		Width(g.Width/2 - 2)
	return lipgloss.JoinHorizontal(lipgloss.Top, g.style.Render(cpuPlot), g.style.Render(memPlot))
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

func (g Graphs) plot(data [][]float64, caption string, o ...asciigraph.Option) string {
	options := []asciigraph.Option{
		asciigraph.Height(g.Height - 6),
		asciigraph.Width(0),
		asciigraph.Offset(6),
		asciigraph.Caption(fmt.Sprintf("%s - %s", caption, g.name)),
		asciigraph.CaptionColor(g.graphColor),
		asciigraph.AxisColor(g.graphColor),
		asciigraph.LabelColor(g.graphColor),
	}
	options = append(options, o...)
	return asciigraph.PlotMany(data, options...)
}

func (g *Graphs) SetSize(width, height int) {
	g.Width = width
	g.Height = height
}
