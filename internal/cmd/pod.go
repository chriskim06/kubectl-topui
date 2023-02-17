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

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/top"

	"github.com/chriskim06/kubectl-ptop/internal/metrics"
	"github.com/chriskim06/kubectl-ptop/internal/ui"
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
		Long: addKeyboardShortcutsToDescription(`Show pod metrics.

CPU and memory percentages are calculated by getting the sum of the container
limits for a given pod.`),
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			app := ui.New(metrics.POD, interval, podOpts, showManagedFields, flags)
			_, err := tea.NewProgram(app, tea.WithAltScreen()).Run()
			return err
		},
	}
)

func init() {
	podCmd.Flags().StringVarP(&podOpts.Selector, "selector", "l", podOpts.Selector, selectorHelpStr)
	podCmd.Flags().IntVar(&interval, "interval", 3, intervalHelpStr)
	podCmd.Flags().BoolVarP(&showManagedFields, "show-managed-fields", "m", false, showManagedFieldsHelpStr)
	flags.AddFlags(podCmd.Flags())
	rootCmd.AddCommand(podCmd)
}
