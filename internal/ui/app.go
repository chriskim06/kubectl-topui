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
	info     *tview.TextView
	cpu      *tvxwidgets.Plot
	mem      *tvxwidgets.Plot
	resource metrics.Resource
	options  interface{}
	current  string
	tick     time.Ticker
}

func New(resource metrics.Resource, interval int, options interface{}, flags *genericclioptions.ConfigFlags) *App {
	app := &App{
		client:   metrics.New(flags),
		resource: resource,
		options:  options,
		items:    tview.NewList().ShowSecondaryText(false),
		info:     tview.NewTextView().SetWrap(true),
		cpu:      NewPlot(),
		mem:      NewPlot(),
		cpuData:  map[string][][]float64{},
		memData:  map[string][][]float64{},
		view:     tview.NewApplication(),
		tick:     *time.NewTicker(time.Duration(interval) * time.Second),
	}
	app.info.SetBorder(true)
	app.info.SetTitleAlign(tview.AlignLeft)
	app.update()

	grid := tview.NewGrid().
		SetRows(0, 0).
		SetColumns(0, 0, 0, 0, 0, 0).
		AddItem(app.cpu, 0, 0, 1, 3, 0, 0, false).
		AddItem(app.mem, 0, 3, 1, 3, 0, 0, false).
		AddItem(app.frame, 1, 0, 1, 4, 0, 0, true).
		AddItem(app.info, 1, 4, 1, 2, 0, 0, false)

	app.frame.SetFocusFunc(func() {
		app.frame.SetBorderColor(tcell.ColorPink)
	})
	app.frame.SetBlurFunc(func() {
		app.frame.SetBorderColor(tcell.ColorWhite)
	})
	app.info.SetFocusFunc(func() {
		app.info.SetBorderColor(tcell.ColorPink)
	})
	app.info.SetBlurFunc(func() {
		app.info.SetBorderColor(tcell.ColorWhite)
	})
	app.info.SetChangedFunc(func() {
		app.view.Draw()
	})
	app.info.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			app.info.SetTitle("")
			app.info.Clear()
			app.view.SetFocus(app.items)
		}
	})
	app.info.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' {
			app.info.SetTitle("")
			app.info.Clear()
			app.view.SetFocus(app.items)
		}
		return event
	})
	app.items.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		app.setCurrent()
		app.updateGraphs()
	})
	app.items.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			if app.current == "" {
				return event
			}
			pod, err := app.client.GetPod(app.current)
			if err != nil {
				app.info.SetText(err.Error())
			} else {
				app.info.SetText(pod)
			}
			app.info.SetTitle(app.current)
			app.view.SetFocus(app.info)
			return nil
		}
		switch event.Rune() {
		case 'q':
			app.view.Stop()
		case 'j':
			i := app.items.GetCurrentItem() + 1
			if i > app.items.GetItemCount()-1 {
				i = 0
			}
			app.items.SetCurrentItem(i)
			app.setCurrent()
			app.updateGraphs()
			return nil
		case 'k':
			app.items.SetCurrentItem(app.items.GetCurrentItem() - 1)
			app.setCurrent()
			app.updateGraphs()
			return nil
		}
		return event
	})
	app.view.SetRoot(grid, true).SetFocus(app.items)
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
				a.view.QueueUpdateDraw(func() {
					a.update()
				})
			}
		}
	}()
	defer func() {
		a.tick.Stop()
		done <- true
	}()
	return a.view.Run()
}

func (a *App) setCurrent() {
	line, _ := a.items.GetItemText(a.items.GetCurrentItem())
	sections := strings.Fields(line)
	x := 0
	if a.resource == metrics.POD {
		x = 1
	}
	a.current = sections[x]
}

func (a *App) update() {
	var err error
	var m []metrics.MetricsValues
	if a.resource == metrics.POD {
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
	a.frame = tview.NewFrame(a.items).AddText(header, true, tview.AlignLeft, tcell.Color(tcell.AttrBold))
	a.frame.SetBorder(true).SetTitle(string(a.resource)).SetTitleAlign(tview.AlignLeft)
	a.updateGraphs()
}

func (a *App) updateGraphs() {
	a.cpu.SetData(a.cpuData[a.current])
	a.mem.SetData(a.memData[a.current])
	a.cpu.SetTitle(fmt.Sprintf("CPU - %s", a.current))
	a.mem.SetTitle(fmt.Sprintf("MEM - %s", a.current))
}
