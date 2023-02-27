package ui

import "github.com/charmbracelet/lipgloss"

var (
	//                     w = 56 = w_a+w_b or w_c+w_d
	//        ╭───────────────────────────────────────────────────────╮
	//        │╭─────────────────────────╮ ╭─────────────────────────╮│
	//        ││     w_a=(w/2)-2         │ │   w_b=(w/2)-2           ││
	//        ││                         │ │                         ││
	//        ││                         │ │                         ││
	//        ││                         │ │                         ││
	//        ││                         │ │                         ││
	//        ││                         │ │                         ││
	// h = 17 │└─────────────────────────┘ └─────────────────────────┘│
	//        │╭────────────────────────────────╮ ╭──────────────────╮│
	//        ││    w_c=(2w/3)-2                │ │ w_d=(w/3)-2      ││
	//        ││                                │ │                  ││
	//        ││                                │ │                  ││
	//        ││                                │ │                  ││
	//        ││                                │ │                  ││
	//        ││                                │ │                  ││
	//        │└────────────────────────────────┘ └──────────────────┘│
	//        └───────────────────────────────────────────────────────┘
	// right/left margins = 1 for all cells

	Adaptive = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "0", Dark: "15"})
	Border   = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).Margin(1)
	ErrStyle = lipgloss.NewStyle().BorderStyle(lipgloss.DoubleBorder()).BorderForeground(lipgloss.Color("9"))
)
