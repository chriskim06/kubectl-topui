package widgets

import (
	"fmt"
	"image"

	. "github.com/gizak/termui/v3"
)

// Plot has two modes: line(default) and scatter.
// Plot also has two marker types: braille(default) and dot.
// A single braille character is a 2x4 grid of dots, so using braille
// gives 2x X resolution and 4x Y resolution over dot mode.
type KubePlot struct {
	Block

	Data            [][]float64
	MaxVal          float64
	LineColors      []Color
	HorizontalScale int
	AxisMetric      string
}

const (
	yAxisLabelsWidth = 4
	yAxisLabelsGap   = 1
)

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
