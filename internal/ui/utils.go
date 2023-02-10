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

var headers = map[metrics.Resource]string{
	metrics.POD:  "NAMESPACE\tNAME\tREADY\tSTATUS\tNODE\tCPU USAGE\tCPU LIMIT\tCPU %\tMEM USAGE\tMEM LIMIT\tMEM %\tRESTARTS\tAGE",
	metrics.NODE: "NAME\tCPU USAGE\tCPU AVAILABLE\tCPU %\tMEM USAGE\tMEM AVAILABLE\tMEM %",
}

func tabStrings(data []metrics.MetricsValues, resource metrics.Resource) (string, []string) {
	var b bytes.Buffer
	w := tabwriter.NewWriter(&b, 0, 0, 3, ' ', 0)
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
			"%s\t%s\t%s\t%s\t%s\t%dm\t%dm\t%0.2f%%\t%dMi\t%dm\t%0.2f%%\t%d\t%s",
			m.Namespace,
			m.Name,
			fmt.Sprintf("%d/%d", m.Ready, m.Total),
			m.Status,
			m.Node,
			m.CPUCores,
			m.CPULimit,
			m.CPUPercent,
			m.MemCores,
			m.MemLimit,
			m.MemPercent,
			m.Restarts,
			m.Age,
		)
	} else {
		return fmt.Sprintf(
			"%s\t%dm\t%dm\t%0.2f%%\t%dMi\t%dMi\t%0.2f%%",
			m.Name,
			m.CPUCores,
			m.CPULimit,
			m.CPUPercent,
			m.MemCores,
			m.MemLimit,
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
