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
)

const (
	selectorHelpStr          = "Selector (label query) to filter on, supports '=', '==', and '!=' (e.g. -l key1=value1,key2=value2)."
	intervalHelpStr          = "The interval in seconds between getting metrics (defaults to 3)."
	showManagedFieldsHelpStr = "Display managed fields when viewing pod or node manifests."
	allNsHelpStr             = "If present, list the requested object(s) across all namespaces. Namespace in current context is ignored even if specified with --namespace."
	keyboardShortcuts        = `
Keyboard Shortcuts:
  - q: quit
  - j: scroll down
  - k: scroll up
  - enter: view spec for selected item`
)

func addKeyboardShortcutsToDescription(usage string) string {
	return fmt.Sprintf("%s\n%s", usage, keyboardShortcuts)
}
