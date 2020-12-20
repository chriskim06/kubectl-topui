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
		Long:    addKeyboardShortcutsToDescription("Show various widgets for node metrics."),
		RunE: func(_ *cobra.Command, args []string) error {
			if !isValidSortKey(nodeOpts.SortBy) {
				return errors.New("Error: --sort-by can be either 'cpu', 'memory', 'cpu-percent', or 'memory-percent'")
			}
			return view.Render(nodeOpts, flags, view.NODE, interval)
		},
	}
)

func init() {
	nodeCmd.Flags().StringVarP(&nodeOpts.Selector, "selector", "l", nodeOpts.Selector, selectorHelpStr)
	nodeCmd.Flags().StringVar(&nodeOpts.SortBy, "sort-by", nodeOpts.Selector, sortHelpStr)
	nodeCmd.Flags().IntVar(&interval, "interval", 5, intervalHelpStr)
	flags.AddFlags(nodeCmd.Flags())
	rootCmd.AddCommand(nodeCmd)
}
