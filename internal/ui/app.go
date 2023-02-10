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

const helpPageName = "Help"

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
	pages    *tview.Pages
	resource metrics.Resource
	options  interface{}
	current  string
	tick     time.Ticker
}

func New(resource metrics.Resource, interval int, options interface{}, flags *genericclioptions.ConfigFlags) *App {
	info := tview.NewTextView().SetWrap(true)
	info.SetBorder(true)
	info.SetTitleAlign(tview.AlignLeft)
	app := &App{
		client:   metrics.New(flags),
		resource: resource,
		options:  options,
		items:    tview.NewList().ShowSecondaryText(false),
		info:     info,
		cpu:      NewPlot(),
		mem:      NewPlot(),
		cpuData:  map[string][][]float64{},
		memData:  map[string][][]float64{},
		view:     tview.NewApplication(),
		tick:     *time.NewTicker(time.Duration(interval) * time.Second),
	}
	app.init()
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

func (a *App) update() {
	var err error
	var m []metrics.MetricsValues
	if a.resource == metrics.POD {
		m, err = a.client.GetPodMetrics(a.options.(*top.TopPodOptions))
	} else {
		m, err = a.client.GetNodeMetrics(a.options.(*top.TopNodeOptions))
	}
	if err != nil {
		a.view.Stop()
		log.Fatal(err)
	}
	for _, metric := range m {
		name := metric.Name
		a.graphUpkeep(name)
		a.cpuData[name][0] = append(a.cpuData[name][0], float64(metric.CPULimit.MilliValue()))
		a.cpuData[name][1] = append(a.cpuData[name][1], float64(metric.CPUCores.MilliValue()))
		a.memData[name][0] = append(a.memData[name][0], float64(metric.MemLimit.Value()/(1024*1024)))
		a.memData[name][1] = append(a.memData[name][1], float64(metric.MemCores.Value()/(1024*1024)))
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

func (a *App) graphUpkeep(name string) {
	if a.cpuData[name] == nil || a.memData[name] == nil {
		a.cpuData[name] = [][]float64{{}, {}}
		a.memData[name] = [][]float64{{}, {}}
		return
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
}

func (a *App) updateGraphs() {
	a.cpu.SetData(a.cpuData[a.current])
	a.mem.SetData(a.memData[a.current])
	a.cpu.SetTitle(fmt.Sprintf("CPU - %s", a.current))
	a.mem.SetTitle(fmt.Sprintf("MEM - %s", a.current))
}

func (a *App) init() {
	a.update()
	a.initInfo()
	a.initItems()

	grid := tview.NewGrid().
		SetRows(0, 0).
		SetColumns(0, 0, 0, 0, 0, 0).
		AddItem(a.cpu, 0, 0, 1, 3, 0, 0, false).
		AddItem(a.mem, 0, 3, 1, 3, 0, 0, false).
		AddItem(a.frame, 1, 0, 1, 4, 0, 0, true).
		AddItem(a.info, 1, 4, 1, 2, 0, 0, false)

	help := tview.NewTextView()
	help.SetTextAlign(tview.AlignLeft).SetBorder(true).SetBorderColor(tcell.ColorPink).SetBorderPadding(1, 1, 1, 1).SetTitle(helpPageName).SetTitleAlign(tview.AlignLeft)
	help.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || event.Rune() == 'q' || event.Rune() == '?' {
			a.pages.HidePage(helpPageName)
			a.view.SetFocus(a.items)
			return nil
		}
		return event
	})
	help.SetText(helpText())
	helpBox := tview.NewGrid().SetRows(0, 0, 0).SetColumns(0, 0, 0).AddItem(help, 1, 1, 1, 1, 0, 0, true)
	pages := tview.NewPages().
		AddPage("app", grid, true, true).
		AddPage(helpPageName, helpBox, true, false)

	a.pages = pages
	a.view.SetRoot(pages, true).SetFocus(a.items)
}

func (a *App) initInfo() {
	a.info.SetFocusFunc(func() {
		a.info.SetBorderColor(tcell.ColorPink)
	})
	a.info.SetBlurFunc(func() {
		a.info.SetBorderColor(tcell.ColorWhite)
	})
	a.info.SetChangedFunc(func() {
		a.view.Draw()
	})
	a.info.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			a.info.SetTitle("")
			a.info.Clear()
			a.view.SetFocus(a.items)
		}
	})
	a.info.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' {
			a.info.SetTitle("")
			a.info.Clear()
			a.view.SetFocus(a.items)
		}
		return event
	})
}

func (a *App) initItems() {
	a.items.SetFocusFunc(func() {
		a.frame.SetBorderColor(tcell.ColorPink)
	})
	a.items.SetBlurFunc(func() {
		a.frame.SetBorderColor(tcell.ColorWhite)
	})
	a.items.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		a.setCurrent()
		a.updateGraphs()
	})
	a.items.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			if a.current == "" {
				return event
			}
			var output string
			var err error
			if a.resource == metrics.POD {
				output, err = a.client.GetPod(a.current)
			} else {
				output, err = a.client.GetNode(a.current)
			}
			if err != nil {
				a.info.SetText(err.Error())
			} else {
				a.info.SetText(output)
			}
			a.info.SetTitle(a.current)
			a.view.SetFocus(a.info)
			return nil
		}
		switch event.Rune() {
		case 'q':
			a.view.Stop()
			return nil
		case 'j':
			i := a.items.GetCurrentItem() + 1
			if i > a.items.GetItemCount()-1 {
				i = 0
			}
			a.items.SetCurrentItem(i)
			a.setCurrent()
			a.updateGraphs()
			return nil
		case 'k':
			a.items.SetCurrentItem(a.items.GetCurrentItem() - 1)
			a.setCurrent()
			a.updateGraphs()
			return nil
		case '?':
			a.pages.ShowPage(helpPageName)
			a.view.SetFocus(a.pages)
			return nil
		}
		return event
	})
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
