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
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/chriskim06/kubectl-ptop/internal/metrics"
	"github.com/chriskim06/kubectl-ptop/internal/view/widgets"
	ui "github.com/gizak/termui/v3"
	uiWidgets "github.com/gizak/termui/v3/widgets"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/top"
)

const (
	POD                 = "pod"
	NODE                = "node"
	SortByCpu           = "cpu"
	SortByMemory        = "memory"
	SortByCpuPercent    = "cpu-percent"
	SortByMemoryPercent = "memory-percent"
)

var columns = []string{"NAME", "CPU(cores)", "CPU%", "MEMORY(bytes)", "MEMORY%"}

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
		return fmt.Errorf("unrecognized resource")
	}
	if err != nil {
		return err
	}

	// initialize termui
	if err := ui.Init(); err != nil {
		return err
	}
	defer ui.Close()

	// create widgets
	lists := make([]*uiWidgets.List, 5)
	for i := 0; i < 5; i++ {
		lists[i] = uiWidgets.NewList()
		lists[i].Title = columns[i]
		lists[i].TitleStyle = ui.Style{Fg: ui.ColorClear, Bg: ui.ColorClear, Modifier: ui.ModifierBold}
		lists[i].TextStyle = ui.Style{Fg: ui.ColorClear, Bg: ui.ColorClear}
		lists[i].SelectedRowStyle = ui.Style{Fg: ui.ColorClear, Bg: ui.ColorClear, Modifier: ui.ModifierBold}
		lists[i].Border = false
	}

	// cpu and mem plots
	cpuPlot := widgets.NewKubePlot()
	cpuPlot.Border = false
	cpuPlot.AxisMetric = "%"
	memPlot := widgets.NewKubePlot()
	memPlot.Border = false
	memPlot.AxisMetric = "%"
	for i := 0; i < len(m); i++ {
		cpuPlot.Data = append(cpuPlot.Data, []float64{0})
		memPlot.Data = append(memPlot.Data, []float64{0})
		cpuPlot.LineColors = append(cpuPlot.LineColors, ui.Color(i))
		memPlot.LineColors = append(memPlot.LineColors, ui.Color(i))
	}

	// custom gauge list widgets
	cpuGaugeList, memGaugeList := widgets.NewGaugeList(), widgets.NewGaugeList()
	cpuGaugeList.Title = "CPU"
	memGaugeList.Title = "Memory"
	cpuGaugeList.TitleStyle = ui.Style{Fg: ui.ColorClear, Bg: ui.ColorClear, Modifier: ui.ModifierBold}
	memGaugeList.TitleStyle = ui.Style{Fg: ui.ColorClear, Bg: ui.ColorClear, Modifier: ui.ModifierBold}

	// tab pane that holds the cpu/mem plots
	tabplot := widgets.NewTabPlot([]string{"CPU Percent", "Mem Percent"}, []*widgets.KubePlot{cpuPlot, memPlot})

	// populate widgets initially
	fillWidgetData(m, lists, cpuGaugeList, memGaugeList, cpuPlot, memPlot, sortBy)

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
			ui.NewCol(1.5/10, lists[0]),
			ui.NewCol(0.75/10, lists[1]),
			ui.NewCol(0.75/10, lists[2]),
			ui.NewCol(0.75/10, lists[3]),
			ui.NewCol(1.25/10, lists[4]),
			ui.NewCol(5.0/10, tabplot),
		),
	)

	// render something initially
	ui.Render(grid)

	// start a new ticker
	duration, _ := time.ParseDuration(fmt.Sprintf("%ds", interval))
	ticker := time.NewTicker(duration)
	defer ticker.Stop()
	quit := make(chan struct{})

	// create a goroutine that redraws the grid at each tick
	go func(flags *genericclioptions.ConfigFlags, resource string, cpuGaugeList, memGaugeList *widgets.GaugeList, lists []*uiWidgets.List, cpuPlot, memPlot *widgets.KubePlot) {
		for {
			select {
			case <-ticker.C:
				// update the widgets and render the grid with new node metrics
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
				fillWidgetData(values, lists, cpuGaugeList, memGaugeList, cpuPlot, memPlot, sortBy)
				ui.Render(grid)
			case <-quit:
				return
			}
		}
	}(flags, resource, cpuGaugeList, memGaugeList, lists, cpuPlot, memPlot)

	uiEvents := ui.PollEvents()
	for {
		e := <-uiEvents
		switch e.ID {
		case "q", "<C-c>":
			close(quit)
			return nil
		case "j", "<Down>":
			for i := 0; i < 5; i++ {
				lists[i].ScrollDown()
			}
			cpuGaugeList.ScrollDown()
			memGaugeList.ScrollDown()
			ui.Render(grid)
		case "k", "<Up>":
			for i := 0; i < 5; i++ {
				lists[i].ScrollUp()
			}
			cpuGaugeList.ScrollUp()
			memGaugeList.ScrollUp()
			ui.Render(grid)
		case "<Tab>":
			tabplot.FocusNext()
			ui.Render(grid)
		case "h", "<Left>":
			tabplot.FocusLeft()
			ui.Render(grid)
		case "l", "<Right>":
			tabplot.FocusRight()
			ui.Render(grid)
		case "<Resize>":
			payload := e.Payload.(ui.Resize)
			grid.SetRect(0, 0, payload.Width, payload.Height)
			ui.Clear()
			ui.Render(grid)
		}
	}
}

func fillWidgetData(metrics []metrics.MetricsValues, resourceLists []*uiWidgets.List, cpuGaugeList, memGaugeList *widgets.GaugeList, cpuPlot, memPlot *widgets.KubePlot, sortBy string) {
	cpuGaugeList.Rows = nil
	memGaugeList.Rows = nil
	for i := 0; i < 5; i++ {
		resourceLists[i].Rows = nil
	}
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
	for i, v := range metrics {
		cpuItem := widgets.NewGaugeListItem(v.CPUPercent, v.Name)
		memItem := widgets.NewGaugeListItem(v.MemPercent, v.Name)
		cpuGaugeList.Rows = append(cpuGaugeList.Rows, cpuItem)
		memGaugeList.Rows = append(memGaugeList.Rows, memItem)
		resourceLists[0].Rows = append(resourceLists[0].Rows, " "+v.Name)
		resourceLists[1].Rows = append(resourceLists[1].Rows, fmt.Sprintf(" %vm", v.CPUCores))
		resourceLists[2].Rows = append(resourceLists[2].Rows, fmt.Sprintf(" %.2f%%", v.CPUPercent))
		resourceLists[3].Rows = append(resourceLists[3].Rows, fmt.Sprintf(" %vMi", v.MemCores/(1024*1024)))
		resourceLists[4].Rows = append(resourceLists[4].Rows, fmt.Sprintf(" %.2f%%", v.MemPercent))
		cpuPlot.Data[i] = append(cpuPlot.Data[i], float64(v.CPUPercent))
		memPlot.Data[i] = append(memPlot.Data[i], float64(v.MemPercent))
	}
}
