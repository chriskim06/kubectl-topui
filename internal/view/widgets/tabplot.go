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
package widgets

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

func (self *TabPlot) FocusNext() {
	lastIdx := len(self.TabNames) - 1
	if self.ActiveTabIndex < lastIdx {
		self.ActiveTabIndex++
	} else if self.ActiveTabIndex == lastIdx {
		self.ActiveTabIndex = 0
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
