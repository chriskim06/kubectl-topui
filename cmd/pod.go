package cmd

import (
	"os"

	"github.com/chriskim06/kubectl-ptop/internal/view"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/top"
)

var (
	podOpts = &top.TopPodOptions{
		IOStreams: genericclioptions.IOStreams{
			In:     os.Stdin,
			Out:    os.Stdout,
			ErrOut: os.Stderr,
		},
	}
	podCmd = &cobra.Command{
		Use:     "pod",
		Aliases: []string{"pods"},
		Short:   "Show pod metrics",
		Long: `Show various widgets for pod metrics.

CPU and memory percentages are calculated by getting the sum of the container
limits/requests for a given pod.`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			return view.Render(podOpts, flags, view.POD)
		},
	}
)

func init() {
	podCmd.Flags().StringVarP(&podOpts.Selector, "selector", "l", podOpts.Selector, "Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	flags.AddFlags(podCmd.Flags())
	rootCmd.AddCommand(podCmd)
}
