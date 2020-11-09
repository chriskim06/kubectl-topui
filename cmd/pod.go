package cmd

import (
	"github.com/chriskim06/kubectl-ptop/internal/view"
	"github.com/spf13/cobra"
)

var podCmd = &cobra.Command{
	Use:   "pod",
	Short: "Show pod metrics",
	Long: `Show various widgets for pod metrics.

CPU and memory percentages are calculated by getting the sum of the container
limits/requests for a given pod.`,
	RunE: func(_ *cobra.Command, args []string) error {
		return view.Render(view.POD)
	},
}

func init() {
	rootCmd.AddCommand(podCmd)
}
