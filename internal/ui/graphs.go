package ui

import (
	"fmt"
	"math"
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
	conf    config.Colors
	name    string
	cpuData map[string][][]float64
	memData map[string][][]float64
	cpuMax  float64
	memMax  float64
	cpuMin  float64
	memMin  float64
	cpuPlot *plot.Model
	memPlot *plot.Model
}

func NewGraphs(conf config.Colors) *Graphs {
	cpuPlot := plot.New()
	memPlot := plot.New()
	cpuPlot.Styles.Container = plotStyle.Copy()
	memPlot.Styles.Container = plotStyle.Copy()
	cpuPlot.Styles.LineColors = []int{conf.CPULimit, conf.CPUUsage}
	cpuPlot.Styles.AxisColor = conf.Axis
	cpuPlot.Styles.LabelColor = conf.Labels
	memPlot.Styles.LineColors = []int{conf.MemLimit, conf.MemUsage}
	memPlot.Styles.AxisColor = conf.Axis
	memPlot.Styles.LabelColor = conf.Labels
	return &Graphs{
		conf:    conf,
		cpuPlot: cpuPlot,
		memPlot: memPlot,
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
	return lipgloss.JoinHorizontal(lipgloss.Left, g.cpuPlot.View(), strings.Repeat(" ", g.extra), g.memPlot.View())
}

func (g *Graphs) SetSize(width, height int) {
	m := tea.WindowSizeMsg{
		Width:  (width / 2) - ph,
		Height: height - pv,
	}
	g.cpuPlot.Update(m)
	g.memPlot.Update(m)
	g.extra = width % 2
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
	g.cpuPlot.Title = fmt.Sprintf("CPU - %s", g.name)
	g.memPlot.Title = fmt.Sprintf("MEM - %s", g.name)
	g.cpuPlot.Update(plot.GraphUpdateMsg{
		Data: g.cpuData[g.name],
	})
	g.memPlot.Update(plot.GraphUpdateMsg{
		Data: g.memData[g.name],
	})
}
