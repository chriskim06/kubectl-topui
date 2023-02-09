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

	"github.com/chriskim06/kubectl-ptop/internal/ui"
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
		Long:    addKeyboardShortcutsToDescription("Show various widgets for node metrics."),
		RunE: func(_ *cobra.Command, args []string) error {
			app := ui.New("node", interval, nodeOpts, flags)
			return app.Run()
		},
	}
)

func init() {
	nodeCmd.Flags().StringVarP(&nodeOpts.Selector, "selector", "l", nodeOpts.Selector, selectorHelpStr)
	nodeCmd.Flags().IntVar(&interval, "interval", 3, intervalHelpStr)
	flags.AddFlags(nodeCmd.Flags())
	rootCmd.AddCommand(nodeCmd)
}
