package ui

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/chriskim06/kubectl-ptop/internal/metrics"
	"github.com/gdamore/tcell/v2"
	"github.com/navidys/tvxwidgets"
	"github.com/rivo/tview"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/top"
)

type App struct {
	client   metrics.MetricsClient
	data     []metrics.MetricsValues
	cpuData  map[string][][]float64
	memData  map[string][][]float64
	view     *tview.Application
	frame    *tview.Frame
	items    *tview.List
	cpu      *tvxwidgets.Plot
	mem      *tvxwidgets.Plot
	resource string
	options  interface{}
	current  string
	tick     time.Ticker
}

func New(resource string, options interface{}, flags *genericclioptions.ConfigFlags) *App {
	app := &App{
		client:   metrics.New(flags),
		resource: resource,
		options:  options,
		items:    tview.NewList().ShowSecondaryText(false),
		cpu:      NewPlot(),
		mem:      NewPlot(),
		cpuData:  map[string][][]float64{},
		memData:  map[string][][]float64{},
		view:     tview.NewApplication(),
		tick:     *time.NewTicker(3 * time.Second),
	}
	app.update()

	graphs := tview.NewFlex().
		AddItem(app.cpu, 0, 1, false).
		AddItem(app.mem, 0, 1, false)
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(graphs, 0, 1, false).
		AddItem(app.frame, 0, 3, false)

	app.view.SetRoot(flex, true).SetFocus(flex)
	app.view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'q':
			app.view.Stop()
		case 'j':
			app.move(1)
		case 'k':
			app.move(-1)
		}
		return event
	})
	return app
}

func (a App) Run() error {
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-a.tick.C:
				a.update()
				a.view.Draw()
			}
		}
	}()
	err := a.view.Run()
	done <- true
	return err
}

func (a *App) move(i int) {
	x := 0
	cur := a.items.GetCurrentItem()
	if cur+i >= a.items.GetItemCount() {
		x = 0
	} else if cur+i < 0 {
		x = a.items.GetItemCount() - 1
	} else {
		x = cur + i
	}
	go func() {
		a.view.QueueUpdateDraw(func() {
			a.items.SetCurrentItem(x)
			a.setCurrent()
			a.updateGraphs()
		})
	}()
}

func (a *App) setCurrent() {
	line, _ := a.items.GetItemText(a.items.GetCurrentItem())
	sections := strings.Fields(line)
	x := 0
	if a.resource == "pod" {
		x = 1
	}
	a.current = sections[x]
}

func (a *App) update() {
	var err error
	var m []metrics.MetricsValues
	if a.resource == "pod" {
		m, err = a.client.GetPodMetrics(a.options.(*top.TopPodOptions))
	} else {
		m, err = a.client.GetNodeMetrics(a.options.(*top.TopNodeOptions))
	}
	if err != nil {
		log.Fatal(err)
	}
	for _, metric := range m {
		name := metric.Name
		if a.cpuData[name] == nil {
			a.cpuData[name] = [][]float64{{}, {}}
		}
		if a.memData[name] == nil {
			a.memData[name] = [][]float64{{}, {}}
		}
		if len(a.cpuData[name][0]) == 100 {
			a.cpuData[name][0] = a.cpuData[name][0][1:]
		}
		if len(a.cpuData[name][1]) == 100 {
			a.cpuData[name][1] = a.cpuData[name][1][1:]
		}
		if len(a.memData[name][0]) == 100 {
			a.memData[name][0] = a.memData[name][0][1:]
		}
		if len(a.memData[name][1]) == 100 {
			a.memData[name][1] = a.memData[name][1][1:]
		}
		a.cpuData[name][0] = append(a.cpuData[name][0], float64(metric.CPULimit))
		a.cpuData[name][1] = append(a.cpuData[name][1], float64(metric.CPUCores))
		a.memData[name][0] = append(a.memData[name][0], float64(metric.MemLimit))
		a.memData[name][1] = append(a.memData[name][1], float64(metric.MemCores))
	}
	a.data = m
	a.updateList()
	a.updateGraphs()
}

func (a *App) updateList() {
	header, items := tabStrings(a.data, a.resource)
	if a.items.GetItemCount() == 0 {
		for _, item := range items {
			a.items.AddItem(item, "", 0, nil)
		}
	} else {
		for i, item := range items {
			a.items.SetItemText(i, item, "")
		}
	}
	a.setCurrent()
	a.frame = tview.NewFrame(a.items).AddText(header, true, tview.AlignLeft, tcell.ColorWhite|tcell.Color(tcell.AttrBold))
	a.frame.SetBorder(true).SetTitle(a.resource).SetTitleAlign(tview.AlignLeft)
}

func (a *App) updateGraphs() {
	a.cpu.SetData(a.cpuData[a.current])
	a.mem.SetData(a.memData[a.current])
	a.cpu.SetTitle(fmt.Sprintf("CPU - %s", a.current))
	a.mem.SetTitle(fmt.Sprintf("MEM - %s", a.current))
}
