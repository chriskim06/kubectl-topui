package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	plot "github.com/chriskim06/bubble-plot"
	"github.com/chriskim06/kubectl-topui/internal/config"
)

var (
	plotStyle = Border.Copy()
	ph, pv    = plotStyle.GetFrameSize()
)

type Graphs struct {
	Height  int
	Width   int
	extra   int
	name    string
	cpuData map[string][][]float64
	memData map[string][][]float64
	labels  []string
	cpuPlot *plot.Model
	memPlot *plot.Model
}

func NewGraphs(conf config.Colors) *Graphs {
	options := []plot.Option{
		plot.WithMaxDataPoints(50),
		plot.WithAxisColor(conf.Axis),
		plot.WithLabelColor(conf.Labels),
	}
	cpuPlot := plot.New(append(options, plot.WithLineColors([]int{conf.CPULimit, conf.CPUUsage}))...)
	memPlot := plot.New(append(options, plot.WithLineColors([]int{conf.MemLimit, conf.MemUsage}))...)
	return &Graphs{
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
	case tickMsg:
		g.updateData(msg.name, msg.cpuData, msg.memData, msg.xAxisLabels)
	}
	return *g, nil
}

func (g *Graphs) View() string {
	return lipgloss.JoinHorizontal(lipgloss.Left, g.cpuPlot.View(), strings.Repeat(" ", g.extra), g.memPlot.View())
}

func (g *Graphs) SetSize(width, height int) {
	m := tea.WindowSizeMsg{
		Width:  (width / 2) - ph,
		Height: height - pv - 2,
	}
	g.memPlot.Update(m)
	g.cpuPlot.Update(m)
	g.extra = width % 2
}

func (g *Graphs) updateData(name string, cpuData, memData map[string][][]float64, labels []string) {
	g.name = name
	g.cpuData = cpuData
	g.memData = memData
	g.labels = labels
	g.cpuPlot.Title = fmt.Sprintf("CPU - %s", g.name)
	g.memPlot.Title = fmt.Sprintf("MEM - %s", g.name)
	g.cpuPlot.Update(plot.GraphUpdateMsg{
		Data:   g.cpuData[g.name],
		Labels: g.labels,
	})
	g.memPlot.Update(plot.GraphUpdateMsg{
		Data:   g.memData[g.name],
		Labels: g.labels,
	})
}
