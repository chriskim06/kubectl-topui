package main

import (
	"fmt"
	"log"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

var cpuData [][]float64 = [][]float64{}

func main() {
	m, err := getNodeMetrics()
	if err != nil {
		log.Fatalln(err)
	}

	// initialize termui
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	// start a new ticker
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	quit := make(chan struct{})

	// create widgets
	lists := make([]*widgets.List, 5)
	for i := 0; i < 5; i++ {
		lists[i] = widgets.NewList()
		lists[i].Title = NodeColumns[i]
		lists[i].TitleStyle = ui.Style{Fg: ui.ColorClear, Bg: ui.ColorClear, Modifier: ui.ModifierBold}
		lists[i].TextStyle = ui.Style{Fg: ui.ColorClear, Bg: ui.ColorClear}
		lists[i].SelectedRowStyle = ui.Style{Bg: ui.ColorClear, Modifier: ui.ModifierBold}
		lists[i].Border = false
	}

	// custom gauge list widget
	cpuGaugeList, memGaugeList := NewGaugeList(), NewGaugeList()
	cpuGaugeList.Title = "CPU"
	memGaugeList.Title = "Memory"
	cpuGaugeList.TitleStyle = ui.Style{Fg: ui.ColorClear, Bg: ui.ColorClear, Modifier: ui.ModifierBold}
	memGaugeList.TitleStyle = ui.Style{Fg: ui.ColorClear, Bg: ui.ColorClear, Modifier: ui.ModifierBold}
	for _, item := range m {
		cpuItem := NewGaugeListItem(item.CPUPercent, item.Name)
		memItem := NewGaugeListItem(item.MemPercent, item.Name)
		cpuGaugeList.Rows = append(cpuGaugeList.Rows, cpuItem)
		memGaugeList.Rows = append(memGaugeList.Rows, memItem)
		lists[0].Rows = append(lists[0].Rows, " "+item.Name)
		lists[1].Rows = append(lists[1].Rows, fmt.Sprintf(" %vm", item.CPUCores))
		lists[2].Rows = append(lists[2].Rows, fmt.Sprintf(" %v%%", item.CPUPercent))
		lists[3].Rows = append(lists[3].Rows, fmt.Sprintf(" %vMi", item.MemCores/(1024*1024)))
		lists[4].Rows = append(lists[4].Rows, fmt.Sprintf(" %v%%", item.MemPercent))
	}

	// use grid to keep relative height and width of terminal
	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)
	grid.Set(
		ui.NewCol(
			2.0/5,
			ui.NewCol(2.0/5, lists[0]),
			ui.NewCol(0.75/5, lists[1]),
			ui.NewCol(0.75/5, lists[2]),
			ui.NewCol(0.75/5, lists[3]),
			ui.NewCol(0.75/5, lists[4]),
		),
		ui.NewCol(
			3.0/5,
			ui.NewRow(1.0/2, cpuGaugeList),
			ui.NewRow(1.0/2, memGaugeList),
		),
	)

	// render something initially
	ui.Render(grid)

	// create a goroutine that redraws the grid at each tick
	go func(cpuGaugeList, memGaugeList *GaugeList, lists []*widgets.List) {
		for {
			select {
			case <-ticker.C:
				// update the widgets and render the grid with new node metrics
				values, err := getNodeMetrics()
				if err != nil {
					log.Println(err)
					return
				}
				cpuGaugeList.Rows = nil
				memGaugeList.Rows = nil
				for i := 0; i < 5; i++ {
					lists[i].Rows = nil
				}
				for _, v := range values {
					cpuItem := NewGaugeListItem(v.CPUPercent, v.Name)
					memItem := NewGaugeListItem(v.MemPercent, v.Name)
					cpuGaugeList.Rows = append(cpuGaugeList.Rows, cpuItem)
					memGaugeList.Rows = append(memGaugeList.Rows, memItem)
					lists[0].Rows = append(lists[0].Rows, " "+v.Name)
					lists[1].Rows = append(lists[1].Rows, fmt.Sprintf(" %vm", v.CPUCores))
					lists[2].Rows = append(lists[2].Rows, fmt.Sprintf(" %v%%", v.CPUPercent))
					lists[3].Rows = append(lists[3].Rows, fmt.Sprintf(" %vMi", v.MemCores/(1024*1024)))
					lists[4].Rows = append(lists[4].Rows, fmt.Sprintf(" %v%%", v.MemPercent))
				}
				ui.Render(grid)
			case <-quit:
				return
			}
		}
	}(cpuGaugeList, memGaugeList, lists)

	uiEvents := ui.PollEvents()
	for {
		e := <-uiEvents
		switch e.ID {
		case "q", "<C-c>":
			close(quit)
			return
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
		case "<Resize>":
			payload := e.Payload.(ui.Resize)
			grid.SetRect(0, 0, payload.Width, payload.Height)
			ui.Clear()
			ui.Render(grid)
		}
	}
}
