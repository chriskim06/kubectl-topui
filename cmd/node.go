package cmd

import (
	"github.com/chriskim06/kubectl-ptop/internal/view"
	"github.com/spf13/cobra"
)

var nodeCmd = &cobra.Command{
	Use:     "node",
	Aliases: []string{"nodes"},
	Short:   "Show node metrics",
	Long:    `Show various widgets for node metrics.`,
	RunE: func(_ *cobra.Command, args []string) error {
		return view.Render(flags, view.NODE)
	},
}

func init() {
	flags.AddFlags(nodeCmd.Flags())
	rootCmd.AddCommand(nodeCmd)
}
