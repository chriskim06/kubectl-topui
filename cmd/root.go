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
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var (
	flags    = genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	interval = 5
	rootCmd  = &cobra.Command{
		Use:   "ptop",
		Short: "Prettier kubectl top output",
		Long: `Render kubectl top output with fancier widgets!

This shows separate lists of gauges for the CPU and memory. It also has a panel
that displays the CPU and memory percentage graphs for the lifespan of the
command invocation. The standard top output is also displayed.

Keyboard Shortcuts:
  - q: quit
  - j: scroll down
  - k: scroll up
  - h: move to left graph panel
  - l: move to right graph panel`,
		Args: cobra.NoArgs,
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	flags.AddFlags(rootCmd.Flags())
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
