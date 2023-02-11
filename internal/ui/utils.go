package ui

import (
	"bytes"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/chriskim06/kubectl-ptop/internal/metrics"
	"github.com/gdamore/tcell/v2"
	"github.com/navidys/tvxwidgets"
	"github.com/rivo/tview"
)

const helpText = `This app shows metrics for pods and nodes! The graphs display the limit and usage for the cpu and memory of whichever item is selected.

Keyboard Shortcuts
  - j: move selection down or scroll down spec
  - k: move selection up or scroll up spec
  - q: quit application or clear pod/node spec
  - ?: open/close this help menu`

var headers = map[metrics.Resource]string{
	metrics.POD:  "NAMESPACE\tNAME\tREADY\tSTATUS\tNODE\tCPU USAGE\tCPU LIMIT\tMEM USAGE\tMEM LIMIT\tRESTARTS\tAGE",
	metrics.NODE: "NAME\tCPU USAGE\tCPU AVAILABLE\tCPU %\tMEM USAGE\tMEM AVAILABLE\tMEM %",
}

func tabStrings(data []metrics.MetricsValues, resource metrics.Resource) (string, []string) {
	var b bytes.Buffer
	w := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, headers[resource])
	for _, m := range data {
		fmt.Fprintln(w, fmtStr(m, resource))
	}
	w.Flush()
	strs := strings.Split(b.String(), "\n")
	header := strs[0]
	items := strs[1 : len(strs)-1]
	return header, items
}

func fmtStr(m metrics.MetricsValues, resource metrics.Resource) string {
	if resource == metrics.POD {
		return fmt.Sprintf(
			"%s\t%s\t%s\t%s\t%s\t%vm\t%vm\t%vMi\t%vMi\t%d\t%s",
			m.Namespace,
			m.Name,
			fmt.Sprintf("%d/%d", m.Ready, m.Total),
			m.Status,
			m.Node,
			m.CPUCores.MilliValue(),
			m.CPULimit.MilliValue(),
			m.MemCores.Value()/(1024*1024),
			m.MemLimit.Value()/(1024*1024),
			m.Restarts,
			m.Age,
		)
	} else {
		return fmt.Sprintf(
			"%s\t%vm\t%vm\t%0.2f%%\t%vMi\t%vMi\t%0.2f%%",
			m.Name,
			m.CPUCores.MilliValue(),
			m.CPULimit.MilliValue(),
			m.CPUPercent,
			m.MemCores.Value()/(1024*1024),
			m.MemLimit.Value()/(1024*1024),
			m.MemPercent,
		)
	}
}

func NewPlot() *tvxwidgets.Plot {
	plot := tvxwidgets.NewPlot()
	plot.SetMarker(tvxwidgets.PlotMarkerBraille)
	plot.SetTitleAlign(tview.AlignLeft)
	plot.SetBorder(true)
	plot.SetBorderPadding(1, 1, 1, 1)
	plot.SetLineColor([]tcell.Color{
		tcell.ColorRed,
		tcell.ColorDarkCyan,
	})
	return plot
}
