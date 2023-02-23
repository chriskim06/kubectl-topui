package ui

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/chriskim06/asciigraph"
	"github.com/chriskim06/kubectl-topui/internal/metrics"
	"k8s.io/cli-runtime/pkg/printers"
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
	errStyle = lipgloss.NewStyle().BorderStyle(lipgloss.DoubleBorder()).BorderForeground(lipgloss.Color("9"))
	headers  = map[metrics.Resource]string{
		metrics.POD:  "NAMESPACE\tNAME\tREADY\tSTATUS\tNODE\tCPU USAGE\tCPU LIMIT\tMEM USAGE\tMEM LIMIT\tRESTARTS\tAGE",
		metrics.NODE: "NAME\tCPU USAGE\tCPU AVAILABLE\tCPU PERCENT\tMEM USAGE\tMEM AVAILABLE\tMEM PERCENT",
	}
)

func tabStrings(data []metrics.MetricValue, resource metrics.Resource) (string, []string) {
	var b bytes.Buffer
	w := printers.GetNewTabWriter(&b)
	fmt.Fprintln(w, headers[resource])
	for i, m := range data {
		writeMetric(w, m, resource)
		if i != len(data)-1 {
			fmt.Fprint(w, "\n")
		}
	}
	w.Flush()
	strs := strings.Split(b.String(), "\n")
	header := strs[0]
	items := strs[1:]
	return header, items
}

func writeMetric(w io.Writer, m metrics.MetricValue, resource metrics.Resource) {
	if resource == metrics.POD {
		fmt.Fprintf(w, "%v\t", m.Namespace)
		fmt.Fprintf(w, "%v\t", m.Name)
		fmt.Fprintf(w, "%s\t", fmt.Sprintf("%d/%d", m.Ready, m.Total))
		fmt.Fprintf(w, "%v\t", m.Status)
		fmt.Fprintf(w, "%v\t", m.Node)
		fmt.Fprintf(w, "%vm\t", m.CPUCores.MilliValue())
		fmt.Fprintf(w, "%vm\t", m.CPULimit.MilliValue())
		fmt.Fprintf(w, "%vMi\t", m.MemCores)
		fmt.Fprintf(w, "%vMi\t", m.MemLimit)
		fmt.Fprintf(w, "%v\t", m.Restarts)
		fmt.Fprintf(w, "%v", m.Age)
	} else {
		fmt.Fprintf(w, "%v\t", m.Name)
		fmt.Fprintf(w, "%vm\t", m.CPUCores.MilliValue())
		fmt.Fprintf(w, "%vm\t", m.CPULimit.MilliValue())
		fmt.Fprintf(w, "%.2f", m.CPUPercent)
		w.Write([]byte("%%\t"))
		fmt.Fprintf(w, " %vMi\t", m.MemCores)
		fmt.Fprintf(w, " %vMi\t", m.MemLimit)
		fmt.Fprintf(w, " %.2f", m.MemPercent)
		w.Write([]byte("%%"))
	}
}

func toColor(s string) lipgloss.Color {
	b, ok := asciigraph.ColorNames[s]
	if !ok {
		return adaptive.GetForeground().(lipgloss.Color)
	}
	return lipgloss.Color(fmt.Sprintf("%d", b))
}
