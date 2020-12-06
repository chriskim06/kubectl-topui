package cmd

import (
	"os"

	"github.com/chriskim06/kubectl-ptop/internal/view"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/top"
)

var (
	nodeOpts = &top.TopNodeOptions{
		IOStreams: genericclioptions.IOStreams{
			In:     os.Stdin,
			Out:    os.Stdout,
			ErrOut: os.Stderr,
		},
	}
	nodeCmd = &cobra.Command{
		Use:     "node",
		Aliases: []string{"nodes"},
		Short:   "Show node metrics",
		Long:    `Show various widgets for node metrics.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return view.Render(nodeOpts, flags, view.NODE)
		},
	}
)

func init() {
	nodeCmd.Flags().StringVarP(&nodeOpts.Selector, "selector", "l", nodeOpts.Selector, "Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	nodeCmd.Flags().StringVar(&nodeOpts.SortBy, "sort-by", nodeOpts.Selector, "If non-empty, sort nodes list using specified field. The field can be either 'cpu' or 'memory'.")
	flags.AddFlags(nodeCmd.Flags())
	rootCmd.AddCommand(nodeCmd)
}
