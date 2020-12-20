/*
Copyright © 2020 Chris Kim

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/top"

	"github.com/chriskim06/kubectl-ptop/internal/view"
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
		Long: addKeyboardShortcutsToDescription(`Show various widgets for pod metrics.

CPU and memory percentages are calculated by getting the sum of the container
limits/requests for a given pod.`),
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			if !isValidSortKey(podOpts.SortBy) {
				return errors.New("Error: --sort-by can be either 'cpu', 'memory', 'cpu-percent', or 'memory-percent'")
			}
			return view.Render(podOpts, flags, view.POD, interval)
		},
	}
)

func init() {
	podCmd.Flags().StringVarP(&podOpts.Selector, "selector", "l", podOpts.Selector, selectorHelpStr)
	podCmd.Flags().StringVar(&podOpts.SortBy, "sort-by", podOpts.Selector, sortHelpStr)
	podCmd.Flags().IntVar(&interval, "interval", 5, intervalHelpStr)
	flags.AddFlags(podCmd.Flags())
	rootCmd.AddCommand(podCmd)
}
