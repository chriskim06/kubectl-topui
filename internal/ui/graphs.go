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
}

func (g Graphs) Init() tea.Cmd {
	return nil
}

func (g *Graphs) Update(msg tea.Msg) (Graphs, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// nothing
	case tickMsg:
		g.name = msg.name
		g.cpuData = msg.cpuData
		g.memData = msg.memData
	}
	return *g, nil
}

func (g Graphs) View() string {
	var cpuMax, memMax float64
	cpuMin, memMin := math.MaxFloat64, math.MaxFloat64
	for _, metrics := range g.cpuData[g.name] {
		for _, value := range metrics {
			if value < cpuMin {
				cpuMin = value
			}
			if value > cpuMax {
				cpuMax = value
			}
		}
	}
	for _, metrics := range g.memData[g.name] {
		for _, value := range metrics {
			if value < memMin {
				memMin = value
			}
			if value > memMax {
				memMax = value
			}
		}
	}
	cpuColors := asciigraph.SeriesColors(asciigraph.ColorNames[string(g.conf.CPULimit)], asciigraph.ColorNames[string(g.conf.CPUUsage)])
	memColors := asciigraph.SeriesColors(asciigraph.ColorNames[string(g.conf.MemLimit)], asciigraph.ColorNames[string(g.conf.MemUsage)])
	cpuPlot := g.plot(g.cpuData[g.name], "CPU", asciigraph.Min(cpuMin), asciigraph.Max(cpuMax), cpuColors)
	memPlot := g.plot(g.memData[g.name], "MEM", asciigraph.Min(memMin), asciigraph.Max(memMax), memColors)
	style := lipgloss.NewStyle().Align(lipgloss.Top).BorderStyle(lipgloss.NormalBorder()).BorderBackground(adaptive.GetBackground()).Width(g.Width/2 - 2).Height(g.Height - 6)
	return lipgloss.JoinHorizontal(lipgloss.Top, style.Render(cpuPlot), style.Render(memPlot))
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
