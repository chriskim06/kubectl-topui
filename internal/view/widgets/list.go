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
	"fmt"
	"image"

	. "github.com/gizak/termui/v3"
)

const (
	rowHeight = 3
)

// GaugeListItem is a single resource gauge in the gauge list
type GaugeListItem struct {
	Percent float64
	Label   string
}

// NewGaugeListItem instantiates a resource gauge
func NewGaugeListItem(percent float64, label string) *GaugeListItem {
	return &GaugeListItem{
		Percent: percent,
		Label:   label,
	}
}

// GaugeList is a custom widget of GaugeListItems with other metadata
type GaugeList struct {
	Block

	Rows             []*GaugeListItem
	SelectedRow      int
	SelectedRowStyle Style
	topRow           int
}

// NewGaugeList instantiates a new gauge list
func NewGaugeList() *GaugeList {
	return &GaugeList{
		Block:            *NewBlock(),
		SelectedRowStyle: Theme.List.Text,
	}
}

// Draw renders the gauge list
func (self *GaugeList) Draw(buf *Buffer) {
	self.Block.Draw(buf)

	// adjusts view into widget
	numRows := self.Inner.Dy() / rowHeight
	if self.SelectedRow > numRows-1 {
		self.topRow = self.SelectedRow - numRows + 1
	} else if self.SelectedRow < self.topRow {
		self.topRow = self.SelectedRow
	}

	// draw rows
	point := self.Inner.Min
	for row := self.topRow; row < len(self.Rows) && point.Y < self.Inner.Max.Y-2; row++ {
		gauge := self.Rows[row]

		// draw border
		self.drawBorder(buf, point, gauge.Label)

		// draw bar
		barWidth := int((float64(gauge.Percent) / 100) * float64(self.Inner.Dx()))
		c := ColorGreen
		if gauge.Percent >= 90 {
			c = ColorRed
		}
		buf.Fill(
			NewCell(' ', NewStyle(ColorClear, c)),
			image.Rect(point.X+2, point.Y+1, point.X+2+barWidth, point.Y+2),
		)

		// add percentage label
		label := fmt.Sprintf("%.2f%%", gauge.Percent)
		labelXCoordinate := point.X + 1 + (self.Inner.Dx() / 2) - int(float64(len(label))/2)
		labelYCoordinate := point.Y + 1
		if labelYCoordinate < self.Inner.Max.Y {
			for i, char := range label {
				style := NewStyle(ColorClear, ColorClear)
				if labelXCoordinate+i+1 <= point.X+barWidth {
					style = NewStyle(c, ColorClear, ModifierReverse)
				}
				buf.SetCell(NewCell(char, style), image.Pt(labelXCoordinate+i, labelYCoordinate))
			}
		}

		// add indicator if this is the selected row
		if row == self.SelectedRow {
			buf.SetCell(NewCell('*', NewStyle(ColorClear, ColorClear, ModifierBold)), image.Pt(point.X, point.Y+1))
		}

		// update the starting point for the next row
		point = image.Pt(self.Inner.Min.X, point.Y+rowHeight)
	}

	// draw UP_ARROW if needed
	if self.topRow > 0 {
		buf.SetCell(
			NewCell(UP_ARROW, NewStyle(ColorWhite)),
			image.Pt(self.Inner.Max.X-1, self.Inner.Min.Y),
		)
	}

	// draw DOWN_ARROW if needed
	if self.topRow+numRows < len(self.Rows) {
		buf.SetCell(
			NewCell(DOWN_ARROW, NewStyle(ColorWhite)),
			image.Pt(self.Inner.Max.X-1, self.Inner.Max.Y-1),
		)
	}
}

func (self *GaugeList) drawBorder(buf *Buffer, point image.Point, label string) {
	verticalCell := NewCell(VERTICAL_LINE, NewStyle(ColorClear, ColorClear))
	horizontalCell := NewCell(HORIZONTAL_LINE, NewStyle(ColorClear, ColorClear))
	buf.Fill(horizontalCell, image.Rect(point.X+1, point.Y, self.Inner.Max.X, point.Y+1))
	buf.Fill(horizontalCell, image.Rect(point.X+1, point.Y+2, self.Inner.Max.X, point.Y+3))
	buf.Fill(verticalCell, image.Rect(point.X+2, point.Y+1, point.X+1, point.Y+2))
	buf.Fill(verticalCell, image.Rect(self.Inner.Max.X-1, point.Y+1, self.Inner.Max.X, point.Y+2))
	buf.SetCell(NewCell(TOP_LEFT, NewStyle(ColorClear, ColorClear)), image.Pt(point.X+1, point.Y))
	buf.SetCell(NewCell(TOP_RIGHT, NewStyle(ColorClear, ColorClear)), image.Pt(self.Inner.Max.X-1, point.Y))
	buf.SetCell(NewCell(BOTTOM_LEFT, NewStyle(ColorClear, ColorClear)), image.Pt(point.X+1, point.Y+2))
	buf.SetCell(NewCell(BOTTOM_RIGHT, NewStyle(ColorClear, ColorClear)), image.Pt(self.Inner.Max.X-1, point.Y+2))
	buf.SetString(
		" "+label+" ",
		NewStyle(ColorClear),
		image.Pt(point.X+2, point.Y),
	)
}

// ScrollAmount scrolls by amount given. If amount is < 0, then scroll up.
// There is no need to set self.topRow, as this will be set automatically when drawn,
// since if the selected item is off screen then the topRow variable will change accordingly.
func (self *GaugeList) ScrollAmount(amount int) {
	if len(self.Rows)-int(self.SelectedRow) <= amount {
		self.SelectedRow = len(self.Rows) - 1
	} else if int(self.SelectedRow)+amount < 0 {
		self.SelectedRow = 0
	} else {
		self.SelectedRow += amount
	}
}

// ScrollUp scrolls up by 1
func (self *GaugeList) ScrollUp() {
	self.ScrollAmount(-1)
}

// ScrollDown scrolls down by 1
func (self *GaugeList) ScrollDown() {
	self.ScrollAmount(1)
}
