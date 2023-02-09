package ui

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"text/tabwriter"
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
	m := metrics.New(flags)
	var err error
	var d []metrics.MetricsValues
	if resource == "pod" {
		d, err = m.GetPodMetrics(options.(*top.TopPodOptions))
	}
	if err != nil {
		log.Fatal(err)
	}
	cpu := tvxwidgets.NewPlot()
	cpu.SetMarker(tvxwidgets.PlotMarkerBraille)
	cpu.SetBorder(true)
	cpu.SetLineColor([]tcell.Color{
		tcell.ColorRed,
		tcell.ColorDarkCyan,
	})
	mem := tvxwidgets.NewPlot()
	mem.SetMarker(tvxwidgets.PlotMarkerBraille)
	mem.SetBorder(true)
	mem.SetLineColor([]tcell.Color{
		tcell.ColorRed,
		tcell.ColorDarkCyan,
	})
	items := tview.NewList().ShowSecondaryText(false)
	app := &App{
		client:   m,
		resource: resource,
		options:  options,
		items:    items,
		cpu:      cpu,
		mem:      mem,
		cpuData:  map[string][][]float64{},
		memData:  map[string][][]float64{},
		view:     tview.NewApplication(),
		tick:     *time.NewTicker(3 * time.Second),
	}
	app.update(d)
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(
			tview.NewFlex().
				AddItem(app.cpu, 0, 1, false).
				AddItem(app.mem, 0, 1, false),
			0,
			1,
			false,
		).
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
				var err error
				var d []metrics.MetricsValues
				if a.resource == "pod" {
					d, err = a.client.GetPodMetrics(a.options.(*top.TopPodOptions))
					if err != nil {
						log.Fatal(err)
					}
				}
				a.update(d)
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
	if a.items.GetCurrentItem()+i >= a.items.GetItemCount() {
		x = 0
	} else if a.items.GetCurrentItem()+i < 0 {
		x = a.items.GetItemCount() - 1
	} else {
		x = a.items.GetCurrentItem() + i
	}
	go func() {
		a.view.QueueUpdateDraw(func() {
			a.items.SetCurrentItem(x)
			line, _ := a.items.GetItemText(a.items.GetCurrentItem())
			sections := strings.Fields(line)
			a.current = sections[1]
			a.updateGraphs()
		})
	}()
}

func (a *App) updateList() {
	var b bytes.Buffer
	w := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAMESPACE\tNAME\tREADY\tSTATUS\tCPU\tMEM\tRESTARTS\tAGE")
	for _, m := range a.data {
		fmt.Fprintln(w, fmt.Sprintf(
			"%s\t%s\t%s\t%s\t%dm\t%dMi\t%d\t%s",
			m.Namespace,
			m.Name,
			fmt.Sprintf("%d/%d", m.Ready, m.Total),
			m.Status,
			m.CPUCores,
			m.MemCores,
			m.Restarts,
			m.Age,
		))
	}
	w.Flush()
	strs := strings.Split(b.String(), "\n")
	header := strs[0]
	items := strs[1 : len(strs)-1]
	if a.items.GetItemCount() == 0 {
		for _, item := range items {
			a.items.AddItem(item, "", 0, nil)
		}
	} else {
		for i, item := range items {
			a.items.SetItemText(i, item, "")
		}
	}
	line, _ := a.items.GetItemText(a.items.GetCurrentItem())
	sections := strings.Fields(line)
	a.current = sections[1]
	a.frame = tview.NewFrame(a.items).AddText(header, true, tview.AlignLeft, tcell.ColorWhite|tcell.Color(tcell.AttrBold))
	a.frame.SetBorder(true)
}

func (a *App) updateGraphs() {
	a.cpu.SetData(a.cpuData[a.current])
	a.mem.SetData(a.memData[a.current])
	a.cpu.SetTitle(fmt.Sprintf("CPU - %s", a.current))
	a.mem.SetTitle(fmt.Sprintf("MEM - %s", a.current))
}

func (a *App) update(m []metrics.MetricsValues) {
	for _, metric := range m {
		if a.cpuData[metric.Name] == nil {
			a.cpuData[metric.Name] = [][]float64{{}, {}}
		}
		if a.memData[metric.Name] == nil {
			a.memData[metric.Name] = [][]float64{{}, {}}
		}
		if len(a.cpuData[metric.Name][0]) == 100 {
			a.cpuData[metric.Name][0] = a.cpuData[metric.Name][0][1:]
		}
		if len(a.cpuData[metric.Name][1]) == 100 {
			a.cpuData[metric.Name][1] = a.cpuData[metric.Name][1][1:]
		}
		if len(a.memData[metric.Name][0]) == 100 {
			a.memData[metric.Name][0] = a.memData[metric.Name][0][1:]
		}
		if len(a.memData[metric.Name][1]) == 100 {
			a.memData[metric.Name][1] = a.memData[metric.Name][1][1:]
		}
		a.cpuData[metric.Name][0] = append(a.cpuData[metric.Name][0], float64(metric.CPULimit))
		a.cpuData[metric.Name][1] = append(a.cpuData[metric.Name][1], float64(metric.CPUCores))
		a.memData[metric.Name][0] = append(a.memData[metric.Name][0], float64(metric.MemLimit))
		a.memData[metric.Name][1] = append(a.memData[metric.Name][1], float64(metric.MemCores))
	}
	a.data = m
	a.updateList()
	a.updateGraphs()
}
