package editor

import (
	"gioui.org/text"
	"golang.org/x/image/math/fixed"
)

// Calculate the visual width of a tab based on the number of
// spaces it must expand to.
func (e *textView) calcTabWidth(shaper *text.Shaper, tabSize int) fixed.Int26_6 {
	shaper.LayoutString(e.params, " ")

	g, _ := shaper.NextGlyph()

	return g.Advance * fixed.Int26_6(tabSize)

}
