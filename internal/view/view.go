/*
Copyright © 2020 Chris Kim

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package view

import (
	"context"
	"log"
	"sort"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/pkg/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/top"

	"github.com/chriskim06/kubectl-ptop/internal/metrics"
	"github.com/chriskim06/kubectl-ptop/internal/view/widgets"
)

type scrollDirection int

const (
	UP     scrollDirection = -1
	DOWN   scrollDirection = 1
	TOP    scrollDirection = 0
	BOTTOM scrollDirection = 2
)

const (
	POD                 = "pod"
	NODE                = "node"
	SortByCpu           = "cpu"
	SortByMemory        = "memory"
	SortByCpuPercent    = "cpu-percent"
	SortByMemoryPercent = "memory-percent"
)

var (
	columns      = []string{"NAME", "CPU(cores)", "CPU%", "MEMORY(bytes)", "MEMORY%"}
	defaultStyle = ui.Style{Fg: ui.ColorClear, Bg: ui.ColorClear, Modifier: ui.ModifierBold}
	validColors  = []int{}
)

// Render gets the resource metrics and initializes the termui widgets
func Render(options interface{}, flags *genericclioptions.ConfigFlags, resource string, interval int) error {
	var m []metrics.MetricsValues
	var err error
	var sortBy string
	switch resource {
	case POD:
		o := options.(*top.TopPodOptions)
		sortBy = o.SortBy
		m, err = metrics.GetPodMetrics(o, flags)
	case NODE:
		o := options.(*top.TopNodeOptions)
		sortBy = o.SortBy
		m, err = metrics.GetNodeMetrics(o, flags)
	default:
		return errors.New("unrecognized resource")
	}
	if err != nil {
		return errors.Wrap(err, "error getting metrics")
	}

	// initialize termui
	if err := ui.Init(); err != nil {
		return errors.Wrap(err, "error initializing termui")
	}
	defer ui.Close()

	// resource list
	rl := widgets.NewResourceList()
	rl.Headers = columns
	rl.TitleStyle = defaultStyle
	rl.SelectedRowStyle = defaultStyle
	rl.Border = false
	colors := map[string]ui.Color{}

	// cpu and mem plots
	cpuPlot := widgets.NewKubePlot()
	cpuPlot.Border = false
	memPlot := widgets.NewKubePlot()
	memPlot.Border = false
	for i := 0; i < len(m); i++ {
		color := ui.Color(validColors[i%len(validColors)])
		cpuPlot.Data = append(cpuPlot.Data, []float64{0})
		memPlot.Data = append(memPlot.Data, []float64{0})
		cpuPlot.LineColors = append(cpuPlot.LineColors, color)
		memPlot.LineColors = append(memPlot.LineColors, color)
		cpuPlot.NameMapping[m[i].Name] = i
		memPlot.NameMapping[m[i].Name] = i
		colors[m[i].Name] = color
	}
	rl.Colors = colors

	// gauge list widgets
	cpuGaugeList, memGaugeList := widgets.NewGaugeList(), widgets.NewGaugeList()
	cpuGaugeList.Title = "CPU"
	memGaugeList.Title = "Memory"
	cpuGaugeList.TitleStyle = defaultStyle
	memGaugeList.TitleStyle = defaultStyle

	// tab pane that holds the cpu/mem plots
	tabplot := widgets.NewTabPlot([]string{"CPU Percent", "Mem Percent"}, []*widgets.KubePlot{cpuPlot, memPlot})

	// populate widgets initially
	fillWidgetData(m, rl, cpuGaugeList, memGaugeList, cpuPlot, memPlot, sortBy)

	// use grid to keep relative height and width of terminal
	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)
	grid.Set(
		ui.NewRow(
			3.0/5,
			ui.NewCol(1.0/2, cpuGaugeList),
			ui.NewCol(1.0/2, memGaugeList),
		),
		ui.NewRow(
			2.0/5,
			ui.NewCol(1.0/2, rl),
			ui.NewCol(1.0/2, tabplot),
		),
	)

	// render something initially
	ui.Render(grid)

	// start a new ticker
	duration := time.Duration(interval) * time.Second
	metricsTicker := time.NewTicker(duration)
	uiTicker := time.NewTicker(duration)
	defer metricsTicker.Stop()
	defer uiTicker.Stop()

	ctx, cancel := context.WithCancel(context.Background())

	// put results into separate channel
	results := make(chan []metrics.MetricsValues)
	go func() {
		for {
			select {
			case <-metricsTicker.C:
				var values []metrics.MetricsValues
				var err error
				if resource == POD {
					o := options.(*top.TopPodOptions)
					values, err = metrics.GetPodMetrics(o, flags)
				} else {
					o := options.(*top.TopNodeOptions)
					values, err = metrics.GetNodeMetrics(o, flags)
				}
				if err != nil {
					log.Println(err)
					return
				}
				results <- values
			case <-ctx.Done():
				return
			}
		}
	}()

	// create a goroutine that redraws the grid at each tick
	go func(cpuGaugeList, memGaugeList *widgets.GaugeList, rl *widgets.ResourceList, cpuPlot, memPlot *widgets.KubePlot) {
		for {
			select {
			case <-uiTicker.C:
				// update the widgets and render the grid with new metrics
				select {
				case values := <-results:
					fillWidgetData(values, rl, cpuGaugeList, memGaugeList, cpuPlot, memPlot, sortBy)
				default:
				}
				ui.Render(grid)
			case <-ctx.Done():
				return
			}
		}
	}(cpuGaugeList, memGaugeList, rl, cpuPlot, memPlot)

	previousKey := ""
	uiEvents := ui.PollEvents()
	for {
		e := <-uiEvents
		switch e.ID {
		case "q", "<C-c>":
			cancel()
			return nil
		case "j", "<Down>":
			scroll(DOWN, rl, cpuGaugeList, memGaugeList)
		case "k", "<Up>":
			scroll(UP, rl, cpuGaugeList, memGaugeList)
		case "g":
			if previousKey == "g" {
				scroll(TOP, rl, cpuGaugeList, memGaugeList)
			}
		case "<Home>":
			scroll(TOP, rl, cpuGaugeList, memGaugeList)
		case "G", "<End>":
			scroll(BOTTOM, rl, cpuGaugeList, memGaugeList)
		case "<Tab>":
			tabplot.FocusNext()
		case "h", "<Left>":
			tabplot.FocusLeft()
		case "l", "<Right>":
			tabplot.FocusRight()
		case "<Resize>":
			payload := e.Payload.(ui.Resize)
			grid.SetRect(0, 0, payload.Width, payload.Height)
			ui.Clear()
		}

		if previousKey == "g" {
			previousKey = ""
		} else {
			previousKey = e.ID
		}

		ui.Render(grid)
	}
}

func scroll(dir scrollDirection, l *widgets.ResourceList, c, m *widgets.GaugeList) {
	switch dir {
	case UP:
		l.ScrollUp()
		c.ScrollUp()
		m.ScrollUp()
	case DOWN:
		l.ScrollDown()
		c.ScrollDown()
		m.ScrollDown()
	case TOP:
		l.ScrollTop()
		c.ScrollTop()
		m.ScrollTop()
	case BOTTOM:
		l.ScrollBottom()
		c.ScrollBottom()
		m.ScrollBottom()
	}
}

func fillWidgetData(metrics []metrics.MetricsValues, resourceList *widgets.ResourceList, cpuGaugeList, memGaugeList *widgets.GaugeList, cpuPlot, memPlot *widgets.KubePlot, sortBy string) {
	resourceList.Metrics = metrics
	cpuGaugeList.Rows = nil
	memGaugeList.Rows = nil
	switch sortBy {
	case SortByCpu:
		sort.Slice(metrics, func(i, j int) bool {
			return metrics[i].CPUCores > metrics[j].CPUCores
		})
	case SortByMemory:
		sort.Slice(metrics, func(i, j int) bool {
			return metrics[i].MemCores > metrics[j].MemCores
		})
	case SortByCpuPercent:
		sort.Slice(metrics, func(i, j int) bool {
			return metrics[i].CPUPercent > metrics[j].CPUPercent
		})
	case SortByMemoryPercent:
		sort.Slice(metrics, func(i, j int) bool {
			return metrics[i].MemPercent > metrics[j].MemPercent
		})
	}
	for _, v := range metrics {
		cpuItem := widgets.NewGaugeListItem(v.CPUPercent, v.Name)
		memItem := widgets.NewGaugeListItem(v.MemPercent, v.Name)
		cpuGaugeList.Rows = append(cpuGaugeList.Rows, cpuItem)
		memGaugeList.Rows = append(memGaugeList.Rows, memItem)
		cpuIdx := cpuPlot.NameMapping[v.Name]
		memIdx := memPlot.NameMapping[v.Name]
		cpuPlot.Data[cpuIdx] = append(cpuPlot.Data[cpuIdx], float64(v.CPUPercent))
		memPlot.Data[memIdx] = append(memPlot.Data[memIdx], float64(v.MemPercent))
	}
}

func init() {
	// exclude white/black colors from the graph to hopefully provide better
	// contrast on a variety of terminal backgrounds
	for i := 1; i < 231; i++ {
		if i != 7 && i != 15 && i != 16 {
			validColors = append(validColors, i)
		}
	}
}
