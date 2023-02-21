package ui

import (
	"bytes"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/charmbracelet/lipgloss"
	"github.com/chriskim06/asciigraph"
	"github.com/chriskim06/kubectl-ptop/internal/metrics"
)

const helpText = `This app shows metrics for pods and nodes! The graphs display the limit and usage for the cpu and memory of whichever item is selected.

Keyboard Shortcuts
  - j: move selection down or scroll down spec
  - k: move selection up or scroll up spec
  - q: quit application or clear pod/node spec
  - ?: open/close this help menu`

var (
	adaptive = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "0", Dark: "15"})
	border   = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder())
	headers  = map[metrics.Resource]string{
		metrics.POD:  "NAMESPACE\tNAME\tREADY\tSTATUS\tNODE\tCPU USAGE\tCPU LIMIT\tMEM USAGE\tMEM LIMIT\tRESTARTS\tAGE",
		metrics.NODE: "NAME\tCPU USAGE\tCPU AVAILABLE\tCPU %\tMEM USAGE\tMEM AVAILABLE\tMEM %",
	}
)

type graphData struct {
	limit []float64
	usage []float64
}

func tabStrings(data []metrics.MetricValue, resource metrics.Resource) (string, []string) {
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

func fmtStr(m metrics.MetricValue, resource metrics.Resource) string {
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
			m.MemCores,
			m.MemLimit,
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
			m.MemCores,
			m.MemLimit,
			m.MemPercent,
		)
	}
}

func toColor(s string) lipgloss.Color {
	b, ok := asciigraph.ColorNames[s]
	if !ok {
		return adaptive.GetForeground().(lipgloss.Color)
	}
	return lipgloss.Color(fmt.Sprintf("%d", b))
}
