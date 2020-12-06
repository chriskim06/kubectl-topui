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
	yAxisLabelsWidth = 4
	yAxisLabelsGap   = 1
)

// KubePlot is a custom widget that plots the resources
type KubePlot struct {
	Block

	Data            [][]float64
	MaxVal          float64
	LineColors      []Color
	HorizontalScale int
	AxisMetric      string
}

// NewKubePlot instantiates a new plot
func NewKubePlot() *KubePlot {
	return &KubePlot{
		Block:           *NewBlock(),
		LineColors:      []Color{},
		Data:            [][]float64{},
		HorizontalScale: 1,
	}
}

func (self *KubePlot) renderBraille(buf *Buffer, drawArea image.Rectangle, maxVal float64) {
	canvas := NewCanvas()
	canvas.Rectangle = drawArea

	for i, l := range self.Data {
		line := l
		if len(line) > 100 {
			line = line[100:]
		}
		previousHeight := int((line[1] / maxVal) * float64(drawArea.Dy()-1))
		for j, val := range line[1:] {
			height := int((val / maxVal) * float64(drawArea.Dy()-1))
			canvas.SetLine(
				image.Pt(
					(drawArea.Min.X+(j*self.HorizontalScale))*2,
					(drawArea.Max.Y-previousHeight-1)*4,
				),
				image.Pt(
					(drawArea.Min.X+((j+1)*self.HorizontalScale))*2,
					(drawArea.Max.Y-height-1)*4,
				),
				SelectColor(self.LineColors, i),
			)
			previousHeight = height
		}
	}

	canvas.Draw(buf)
}

func (self *KubePlot) plotAxes(buf *Buffer, maxVal float64) {
	// draw origin cell
	buf.SetCell(
		NewCell(BOTTOM_LEFT, NewStyle(ColorClear)),
		image.Pt(self.Inner.Min.X+yAxisLabelsWidth, self.Inner.Max.Y-1),
	)
	// draw x axis line
	for i := yAxisLabelsWidth + 1; i < self.Inner.Dx(); i++ {
		buf.SetCell(
			NewCell(HORIZONTAL_DASH, NewStyle(ColorClear)),
			image.Pt(i+self.Inner.Min.X, self.Inner.Max.Y-1),
		)
	}
	// draw y axis line
	for i := 0; i < self.Inner.Dy()-1; i++ {
		buf.SetCell(
			NewCell(VERTICAL_DASH, NewStyle(ColorClear)),
			image.Pt(self.Inner.Min.X+yAxisLabelsWidth, i+self.Inner.Min.Y),
		)
	}
	// draw y axis labels
	verticalScale := maxVal / float64(self.Inner.Dy()-1)
	for i := 0; i*(yAxisLabelsGap+1) < self.Inner.Dy()-1; i++ {
		buf.SetString(
			fmt.Sprintf("%.2f%s", float64(i)*verticalScale*(yAxisLabelsGap+1), self.AxisMetric),
			NewStyle(ColorClear),
			image.Pt(self.Inner.Min.X, self.Inner.Max.Y-(i*(yAxisLabelsGap+1))-2),
		)
	}
}

// Draw renders the plot
func (self *KubePlot) Draw(buf *Buffer) {
	self.Block.Draw(buf)

	maxVal := self.MaxVal
	if maxVal == 0 {
		maxVal, _ = GetMaxFloat64From2dSlice(self.Data)
	}

	self.plotAxes(buf, maxVal)

	drawArea := image.Rect(
		self.Inner.Min.X+yAxisLabelsWidth+1, self.Inner.Min.Y,
		self.Inner.Max.X, self.Inner.Max.Y-1,
	)
	self.renderBraille(buf, drawArea, maxVal)
}
