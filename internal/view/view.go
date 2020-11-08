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

func Something() error {
	m, err := metrics.GetNodeMetrics()
	if err != nil {
		return err
	}

	// initialize termui
	if err := ui.Init(); err != nil {
		return err
	}
	defer ui.Close()

	// start a new ticker
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	quit := make(chan struct{})

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
	memPlot := widgets.NewKubePlot()
	memPlot.Border = false

	// custom gauge list widgets
	cpuGaugeList, memGaugeList := widgets.NewGaugeList(), widgets.NewGaugeList()
	cpuGaugeList.Title = "CPU"
	memGaugeList.Title = "Memory"
	cpuGaugeList.TitleStyle = ui.Style{Fg: ui.ColorClear, Bg: ui.ColorClear, Modifier: ui.ModifierBold}
	memGaugeList.TitleStyle = ui.Style{Fg: ui.ColorClear, Bg: ui.ColorClear, Modifier: ui.ModifierBold}
	for i, item := range m {
		cpuItem := widgets.NewGaugeListItem(item.CPUPercent, item.Name)
		memItem := widgets.NewGaugeListItem(item.MemPercent, item.Name)
		cpuGaugeList.Rows = append(cpuGaugeList.Rows, cpuItem)
		memGaugeList.Rows = append(memGaugeList.Rows, memItem)
		lists[0].Rows = append(lists[0].Rows, " "+item.Name)
		lists[1].Rows = append(lists[1].Rows, fmt.Sprintf(" %vm", item.CPUCores))
		lists[2].Rows = append(lists[2].Rows, fmt.Sprintf(" %v%%", item.CPUPercent))
		lists[3].Rows = append(lists[3].Rows, fmt.Sprintf(" %vMi", item.MemCores/(1024*1024)))
		lists[4].Rows = append(lists[4].Rows, fmt.Sprintf(" %v%%", item.MemPercent))
		cpuPlot.LineColors = append(cpuPlot.LineColors, ui.Color(i))
		cpuPlot.Data = append(cpuPlot.Data, []float64{0, float64(item.CPUPercent)})
		memPlot.LineColors = append(memPlot.LineColors, ui.Color(i))
		memPlot.Data = append(memPlot.Data, []float64{0, float64(item.MemPercent)})
	}

	// tab pane that holds the cpu/mem plots
	tabplot := widgets.NewTabPlot([]string{"CPU Percent", "Mem Percent"}, []*widgets.KubePlot{cpuPlot, memPlot})

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
				cpuGaugeList.Rows = nil
				memGaugeList.Rows = nil
				for i := 0; i < 5; i++ {
					lists[i].Rows = nil
				}
				for i, v := range values {
					cpuItem := widgets.NewGaugeListItem(v.CPUPercent, v.Name)
					memItem := widgets.NewGaugeListItem(v.MemPercent, v.Name)
					cpuGaugeList.Rows = append(cpuGaugeList.Rows, cpuItem)
					memGaugeList.Rows = append(memGaugeList.Rows, memItem)
					lists[0].Rows = append(lists[0].Rows, " "+v.Name)
					lists[1].Rows = append(lists[1].Rows, fmt.Sprintf(" %vm", v.CPUCores))
					lists[2].Rows = append(lists[2].Rows, fmt.Sprintf(" %v%%", v.CPUPercent))
					lists[3].Rows = append(lists[3].Rows, fmt.Sprintf(" %vMi", v.MemCores/(1024*1024)))
					lists[4].Rows = append(lists[4].Rows, fmt.Sprintf(" %v%%", v.MemPercent))
					cpuPlot.Data[i] = append(cpuPlot.Data[i], float64(v.CPUPercent))
					memPlot.Data[i] = append(memPlot.Data[i], float64(v.MemPercent))
				}
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
