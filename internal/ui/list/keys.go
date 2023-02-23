package list

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines keybindings. It satisfies to the help.KeyMap interface, which
// is used to render the menu.
type KeyMap struct {
	// Keybindings used when browsing the list.
	CursorUp    key.Binding
	CursorDown  key.Binding
	CursorLeft  key.Binding
	CursorRight key.Binding
	NextPage    key.Binding
	PrevPage    key.Binding
	GoToStart   key.Binding
	GoToEnd     key.Binding

	// The quit keybinding. This won't be caught when filtering.
	Quit key.Binding

	// The quit-no-matter-what keybinding. This will be caught when filtering.
	ForceQuit key.Binding
}

// DefaultKeyMap returns a default set of keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		// Browsing.
		CursorUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		CursorDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		CursorLeft: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "left"),
		),
		CursorRight: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "right"),
		),
		PrevPage: key.NewBinding(
			key.WithKeys("shift+tab", "pgup"),
			key.WithHelp("shift+tab", "prev page"),
		),
		NextPage: key.NewBinding(
			key.WithKeys("tab", "pgdown"),
			key.WithHelp("tab", "next page"),
		),
		GoToStart: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("g", "go to start"),
		),
		GoToEnd: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("G", "go to end"),
		),

		// Quitting.
		Quit: key.NewBinding(
			key.WithKeys("q", "esc"),
			key.WithHelp("q", "quit"),
		),
		ForceQuit: key.NewBinding(key.WithKeys("ctrl+c")),
	}
}
