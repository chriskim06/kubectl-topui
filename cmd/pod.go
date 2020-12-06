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
