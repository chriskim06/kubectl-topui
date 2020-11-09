package view

import (
	"fmt"
	"log"
	"time"

	"github.com/chriskim06/kubectl-ptop/internal/metrics"
	"github.com/chriskim06/kubectl-ptop/internal/view/widgets"
	ui "github.com/gizak/termui/v3"
	uiWidgets "github.com/gizak/termui/v3/widgets"
)

const (
	POD  = "pod"
	NODE = "node"
)

func Render(resource string) error {
	m, err := metrics.GetNodeMetrics()
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
		lists[i].Title = metrics.NodeColumns[i]
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
	fillWidgetData(m, lists, cpuGaugeList, memGaugeList, cpuPlot, memPlot)

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
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	quit := make(chan struct{})

	// create a goroutine that redraws the grid at each tick
	go func(cpuGaugeList, memGaugeList *widgets.GaugeList, lists []*uiWidgets.List, cpuPlot, memPlot *widgets.KubePlot) {
		for {
			select {
			case <-ticker.C:
				// update the widgets and render the grid with new node metrics
				values, err := metrics.GetNodeMetrics()
				if err != nil {
					log.Println(err)
					return
				}
				fillWidgetData(values, lists, cpuGaugeList, memGaugeList, cpuPlot, memPlot)
				ui.Render(grid)
			case <-quit:
				return
			}
		}
	}(cpuGaugeList, memGaugeList, lists, cpuPlot, memPlot)

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

func fillWidgetData(metrics []metrics.MetricsValues, resourceLists []*uiWidgets.List, cpuGaugeList, memGaugeList *widgets.GaugeList, cpuPlot, memPlot *widgets.KubePlot) {
	cpuGaugeList.Rows = nil
	memGaugeList.Rows = nil
	for i := 0; i < 5; i++ {
		resourceLists[i].Rows = nil
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
