package main

import (
	"image"

	. "github.com/gizak/termui/v3"
)

type TabPlot struct {
	Block

	TabNames         []string
	Plots            []*KubePlot
	ActiveTabIndex   int
	ActiveTabStyle   Style
	InactiveTabStyle Style
}

func NewTabPlot(names []string, plots []*KubePlot) *TabPlot {
	return &TabPlot{
		Block:            *NewBlock(),
		TabNames:         names,
		Plots:            plots,
		ActiveTabStyle:   Style{Fg: ColorClear, Bg: ColorClear, Modifier: ModifierBold},
		InactiveTabStyle: Style{Fg: ColorClear, Bg: ColorClear},
	}
}

func (self *TabPlot) FocusLeft() {
	if self.ActiveTabIndex > 0 {
		self.ActiveTabIndex--
	}
}

func (self *TabPlot) FocusRight() {
	if self.ActiveTabIndex < len(self.TabNames)-1 {
		self.ActiveTabIndex++
	}
}

func (self *TabPlot) Draw(buf *Buffer) {
	self.Block.Draw(buf)

	// draw tab names
	xCoordinate := self.Inner.Min.X
	for i, name := range self.TabNames {
		ColorPair := self.InactiveTabStyle
		if i == self.ActiveTabIndex {
			ColorPair = self.ActiveTabStyle
		}
		buf.SetString(
			TrimString(name, self.Inner.Max.X-xCoordinate),
			ColorPair,
			image.Pt(xCoordinate, self.Inner.Min.Y),
		)

		xCoordinate += 1 + len(name)

		if i < len(self.TabNames)-1 && xCoordinate < self.Inner.Max.X {
			buf.SetCell(
				NewCell(VERTICAL_LINE, NewStyle(ColorWhite)),
				image.Pt(xCoordinate, self.Inner.Min.Y),
			)
		}

		xCoordinate += 2
	}

	// draw active panel
	activeTab := self.Plots[self.ActiveTabIndex]
	activeTab.SetRect(
		self.Inner.Min.X,
		self.Inner.Min.Y+1,
		self.Inner.Max.X,
		self.Inner.Max.Y,
	)
	activeTab.Draw(buf)
}
