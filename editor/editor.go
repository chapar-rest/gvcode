// This file is based on the Editor part of package gioui.org/widget.
//
// Copyright (c) 2018-2025 Elias Naur and Gio contributors

package editor

import (
	"image"
	"time"
	"unicode"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/text"
	"gioui.org/unit"
	"github.com/oligo/gvcode/buffer"
)

// Editor implements an editable and scrollable text area.
type Editor struct {
	// Alignment controls the alignment of text within the editor.
	Alignment text.Alignment
	// LineHeight determines the gap between baselines of text. If zero, a sensible
	// default will be used.
	LineHeight unit.Sp
	// LineHeightScale is multiplied by LineHeight to determine the final gap
	// between baselines. If zero, a sensible default will be used.
	LineHeightScale float32
	// WrapPolicy configures how displayed text will be broken into lines.
	WrapPolicy text.WrapPolicy
	// ReadOnly controls whether the contents of the editor can be altered by
	// user interaction. If set to true, the editor will allow selecting text
	// and copying it interactively, but not modifying it.
	ReadOnly bool
	// InputHint specifies the type of on-screen keyboard to be displayed.
	InputHint key.InputHint

	// text manages the text buffer and provides shaping and cursor positioning
	// services.
	text       textView
	buffer     buffer.TextSource
	textStyles []*TextStyle
	// Match ranges in rune offset, for text search.
	matches []TextRange
	// Index of the current [TextRange].
	currentMatch int
	// scratch is a byte buffer that is reused to efficiently read portions of text
	// from the textView.
	scratch []byte
	// regions is a region buffer.
	regions []Region

	blinkStart time.Time

	// ime tracks the state relevant to input methods.
	ime struct {
		imeState
		scratch []byte
	}

	dragging    bool
	dragger     gesture.Drag
	scroller    gesture.Scroll
	scrollCaret bool
	showCaret   bool
	clicker     gesture.Click
	pending     []EditorEvent
}

type imeState struct {
	selection struct {
		rng   key.Range
		caret key.Caret
	}
	snippet    key.Snippet
	start, end int
}

type selectionAction int

const (
	selectionExtend selectionAction = iota
	selectionClear
)

type EditorEvent interface {
	isEditorEvent()
}

// A ChangeEvent is generated for every user change to the text.
type ChangeEvent struct{}

// A SelectEvent is generated when the user selects some text, or changes the
// selection (e.g. with a shift-click), including if they remove the
// selection. The selected text is not part of the event, on the theory that
// it could be a relatively expensive operation (for a large editor), most
// applications won't actually care about it, and those that do can call
// Editor.SelectedText() (which can be empty).
type SelectEvent struct{}

const (
	blinksPerSecond  = 1
	maxBlinkDuration = 10 * time.Second
)

// initBuffer should be invoked first in every exported function that accesses
// text state. It ensures that the underlying text widget is both ready to use
// and has its fields synced with the editor.
func (e *Editor) initBuffer() {
	if e.buffer == nil {
		e.buffer = buffer.NewTextSource()
		e.text.SetSource(e.buffer)
	}

	e.text.CaretWidth = unit.Dp(1)
	e.text.Alignment = e.Alignment
	e.text.LineHeight = e.LineHeight
	e.text.LineHeightScale = e.LineHeightScale
	e.text.WrapPolicy = e.WrapPolicy
}

// Update the state of the editor in response to input events. Update consumes editor
// input events until there are no remaining events or an editor event is generated.
// To fully update the state of the editor, callers should call Update until it returns
// false.
func (e *Editor) Update(gtx layout.Context) (EditorEvent, bool) {
	e.initBuffer()
	event, ok := e.processEvents(gtx)
	// Notify IME of selection if it changed.
	newSel := e.ime.selection
	start, end := e.text.Selection()
	newSel.rng = key.Range{
		Start: start,
		End:   end,
	}
	caretPos, carAsc, carDesc := e.text.CaretInfo()
	newSel.caret = key.Caret{
		Pos:     layout.FPt(caretPos),
		Ascent:  float32(carAsc),
		Descent: float32(carDesc),
	}
	if newSel != e.ime.selection {
		e.ime.selection = newSel
		gtx.Execute(key.SelectionCmd{Tag: e, Range: newSel.rng, Caret: newSel.caret})
	}

	e.updateSnippet(gtx, e.ime.start, e.ime.end)
	return event, ok
}

// Layout lays out the editor using the provided textMaterial as the paint material
// for the text glyphs+caret and the selectMaterial as the paint material for the
// selection rectangle.
func (e *Editor) Layout(gtx layout.Context, lt *text.Shaper, font font.Font, size unit.Sp,
	textMaterial, selectMaterial op.CallOp, lineMaterial op.CallOp, matchMaterial op.CallOp, lineNumberMaterial op.CallOp) layout.Dimensions {
	for {
		_, ok := e.Update(gtx)
		if !ok {
			break
		}
	}

	// Adjust scrolling for new viewport and layout.
	e.text.ScrollRel(0, 0)

	if e.scrollCaret {
		e.scrollCaret = false
		e.text.ScrollToCaret()
	}
	// visibleDims := e.text.Dimensions()

	defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops).Pop()
	e.scroller.Add(gtx.Ops)

	return layout.Flex{
		Axis: layout.Horizontal,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return e.text.PaintLineNumber(gtx, lt, lineNumberMaterial)
		}),

		layout.Rigid(layout.Spacer{Width: unit.Dp(24)}.Layout),

		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			e.text.Layout(gtx, lt, font, size)
			return e.layout(gtx, textMaterial, selectMaterial, lineMaterial, matchMaterial)
		}),
	)

}

func (e *Editor) layout(gtx layout.Context, textMaterial, selectMaterial op.CallOp, lineMaterial op.CallOp, rangeMaterial op.CallOp) layout.Dimensions {
	defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops).Pop()
	pointer.CursorText.Add(gtx.Ops)
	event.Op(gtx.Ops, e)
	key.InputHintOp{Tag: e, Hint: e.InputHint}.Add(gtx.Ops)

	e.scroller.Add(gtx.Ops)

	e.clicker.Add(gtx.Ops)
	e.dragger.Add(gtx.Ops)
	e.showCaret = false
	if gtx.Focused(e) {
		now := gtx.Now
		dt := now.Sub(e.blinkStart)
		blinking := dt < maxBlinkDuration
		const timePerBlink = time.Second / blinksPerSecond
		nextBlink := now.Add(timePerBlink/2 - dt%(timePerBlink/2))
		if blinking {
			gtx.Execute(op.InvalidateCmd{At: nextBlink})
		}
		e.showCaret = !blinking || dt%timePerBlink < timePerBlink/2
	}
	semantic.Editor.Add(gtx.Ops)
	if e.Len() > 0 {
		e.paintSelection(gtx, selectMaterial)
		e.paintLineHighlight(gtx, lineMaterial)
		e.paintTextRanges(gtx, rangeMaterial)
		e.paintText(gtx, textMaterial)
	}
	if gtx.Enabled() {
		e.paintCaret(gtx, textMaterial)
	}
	return layout.Dimensions{Size: gtx.Constraints.Max}
}

// paintSelection paints the contrasting background for selected text using the provided
// material to set the painting material for the selection.
func (e *Editor) paintSelection(gtx layout.Context, material op.CallOp) {
	e.initBuffer()
	e.text.PaintSelection(gtx, material)
}

// paintText paints the text glyphs using the provided material to set the fill of the
// glyphs.
func (e *Editor) paintText(gtx layout.Context, material op.CallOp) {
	e.initBuffer()
	e.text.PaintText(gtx, material, e.textStyles)
}

// paintCaret paints the text glyphs using the provided material to set the fill material
// of the caret rectangle.
func (e *Editor) paintCaret(gtx layout.Context, material op.CallOp) {
	e.initBuffer()
	if !e.showCaret || e.ReadOnly {
		return
	}
	e.text.PaintCaret(gtx, material)
}

func (e *Editor) paintLineHighlight(gtx layout.Context, material op.CallOp) {
	e.initBuffer()
	e.text.paintLineHighlight(gtx, material)
}

// SetMatches sets the matched text ranges after a find operation.
func (e *Editor) SetMatches(matches []TextRange) {
	e.matches = matches
	e.ClearSelection()
	if len(matches) > 0 {
		e.currentMatch = 0
	}
}

func (e *Editor) paintTextRanges(gtx layout.Context, material op.CallOp) {
	e.initBuffer()

	e.regions = e.regions[:0]
	rg := make([]Region, 0)
	for _, match := range e.matches {
		rg = rg[:0]
		e.regions = append(e.regions, e.text.Regions(match.Start, match.End, rg)...)
	}

	e.text.PaintRegions(gtx, e.regions, material)
}

// NextMatch is used to switch between the [TextRange]s. This also selects the next match and causes
// the selection background drawn under the matched text.
func (e *Editor) NextMatch(index int) {
	if index < 0 || index >= len(e.matches) {
		return
	}

	e.currentMatch = index
	e.SetCaret(e.matches[e.currentMatch].Start, e.matches[e.currentMatch].End)
}

// Len is the length of the editor contents, in runes.
func (e *Editor) Len() int {
	e.initBuffer()
	return e.buffer.Len()
}

// Text returns the contents of the editor.
func (e *Editor) Text() string {
	e.initBuffer()
	e.scratch = e.buffer.Text(e.scratch)
	return string(e.scratch)
}

func (e *Editor) SetText(s string) {
	e.initBuffer()

	e.text.SetText(s)
	e.ime.start = 0
	e.ime.end = 0
	// Reset xoff and move the caret to the beginning.
	e.SetCaret(0, 0)
}

// CaretPos returns the line & column numbers of the caret.
func (e *Editor) CaretPos() (line, col int) {
	e.initBuffer()
	return e.text.CaretPos()
}

// CaretCoords returns the coordinates of the caret, relative to the
// editor itself.
func (e *Editor) CaretCoords() f32.Point {
	e.initBuffer()
	return e.text.CaretCoords()
}

// Delete runes from the caret position. The sign of the argument specifies the
// direction to delete: positive is forward, negative is backward.
//
// If there is a selection, it is deleted and counts as a single grapheme
// cluster.
func (e *Editor) Delete(graphemeClusters int) (deletedRunes int) {
	e.initBuffer()
	if graphemeClusters == 0 {
		return 0
	}

	start, end := e.text.Selection()
	if start != end {
		graphemeClusters -= sign(graphemeClusters)
	}

	// Move caret by the target quantity of clusters.
	e.text.MoveCaret(0, graphemeClusters)
	// Get the new rune offsets of the selection.
	start, end = e.text.Selection()
	e.replace(start, end, "")
	// Reset xoff.
	e.text.MoveCaret(0, 0)
	e.ClearSelection()
	return end - start
}

func (e *Editor) Insert(s string) (insertedRunes int) {
	e.initBuffer()

	start, end := e.text.Selection()
	moves := e.replace(start, end, s)
	if end < start {
		start = end
	}
	// Reset xoff.
	e.text.MoveCaret(0, 0)
	e.SetCaret(start+moves, start+moves)
	e.scrollCaret = true
	return moves
}

// undo revert the last operation(s).
func (e *Editor) undo() (EditorEvent, bool) {
	e.initBuffer()

	positions, ok := e.text.Undo()
	if !ok {
		return nil, false
	}

	var start, end int
	for _, pos := range positions {
		start = pos.Start
		end = pos.End
	}

	e.SetCaret(end, start)
	return ChangeEvent{}, true
}

// redo revert the last undo operation.
func (e *Editor) redo() (EditorEvent, bool) {
	e.initBuffer()

	positions, ok := e.text.Redo()
	if !ok {
		return nil, false
	}

	var start, end int
	for _, pos := range positions {
		start = pos.Start
		end = pos.End
	}

	e.SetCaret(end, start)
	return ChangeEvent{}, true
}

// replace the text between start and end with s. Indices are in runes.
// It returns the number of runes inserted.
func (e *Editor) replace(start, end int, s string) int {
	length := e.text.Len()
	if start > end {
		start, end = end, start
	}
	start = min(start, length)
	end = min(end, length)

	sc := e.text.Replace(start, end, s)
	newEnd := start + sc
	adjust := func(pos int) int {
		switch {
		case newEnd < pos && pos <= end:
			pos = newEnd
		case end < pos:
			diff := newEnd - end
			pos = pos + diff
		}
		return pos
	}
	e.ime.start = adjust(e.ime.start)
	e.ime.end = adjust(e.ime.end)
	return sc
}

// ReplaceAll assumes a context of "Find & Replace". newStr applies
// to a list of text [TextRange], and the matched text is replaced
// with newStr one by one. The number of replacement is saved to be
// used during undo/redo.
// It returns the number of occurrences replaced.
func (e *Editor) ReplaceAll(newStr string) int {
	if len(e.matches) <= 0 {
		return 0
	}

	// Traverse in reverse order to prevent match offset changes after
	// each replace.
	e.buffer.GroupOp()
	finalPos := 0
	for idx := len(e.matches) - 1; idx >= 0; idx-- {
		start, end := e.matches[idx].Start, e.matches[idx].End
		e.replace(start, end, newStr)
		finalPos = start
	}
	e.buffer.UnGroupOp()

	e.SetCaret(finalPos, finalPos)
	return len(e.matches)
}

// MoveCaret moves the caret (aka selection start) and the selection end
// relative to their current positions. Positive distances moves forward,
// negative distances moves backward. Distances are in grapheme clusters,
// which closely match what users perceive as "characters" even when the
// characters are multiple code points long.
func (e *Editor) MoveCaret(startDelta, endDelta int) {
	e.initBuffer()
	e.text.MoveCaret(startDelta, endDelta)
}

// deleteWord deletes the next word(s) in the specified direction.
// Unlike moveWord, deleteWord treats whitespace as a word itself.
// Positive is forward, negative is backward.
// Absolute values greater than one will delete that many words.
// The selection counts as a single word.
func (e *Editor) deleteWord(distance int) (deletedRunes int) {
	if distance == 0 {
		return
	}

	start, end := e.text.Selection()
	if start != end {
		deletedRunes = e.Delete(1)
		distance -= sign(distance)
	}
	if distance == 0 {
		return deletedRunes
	}

	// split the distance information into constituent parts to be
	// used independently.
	words, direction := distance, 1
	if distance < 0 {
		words, direction = distance*-1, -1
	}
	caret, _ := e.text.Selection()
	// atEnd if offset is at or beyond either side of the buffer.
	atEnd := func(runes int) bool {
		idx := caret + runes*direction
		return idx <= 0 || idx >= e.Len()
	}
	// next returns the appropriate rune given the direction and offset in runes).
	next := func(runes int) rune {
		idx := caret + runes*direction
		if idx < 0 {
			idx = 0
		} else if idx > e.Len() {
			idx = e.Len()
		}
		off := e.text.ByteOffset(idx)
		var r rune
		if direction < 0 {
			r, _, _ = e.buffer.ReadRuneBeforeBytes(int64(off))
		} else {
			r, _, _ = e.buffer.ReadRuneAtBytes(int64(off))
		}
		return r
	}
	runes := 1
	for ii := 0; ii < words; ii++ {
		r := next(runes)
		wantSpace := unicode.IsSpace(r)
		for r := next(runes); unicode.IsSpace(r) == wantSpace && !atEnd(runes); r = next(runes) {
			runes += 1
		}
	}
	deletedRunes += e.Delete(runes * direction)
	return deletedRunes
}

// SelectionLen returns the length of the selection, in runes; it is
// equivalent to utf8.RuneCountInString(e.SelectedText()).
func (e *Editor) SelectionLen() int {
	e.initBuffer()
	return e.text.SelectionLen()
}

// Selection returns the start and end of the selection, as rune offsets.
// start can be > end.
func (e *Editor) Selection() (start, end int) {
	e.initBuffer()
	return e.text.Selection()
}

// SetCaret moves the caret to start, and sets the selection end to end. start
// and end are in runes, and represent offsets into the editor text.
func (e *Editor) SetCaret(start, end int) {
	e.initBuffer()
	e.text.SetCaret(start, end)
	e.scrollCaret = true
	e.scroller.Stop()
}

// SelectedText returns the currently selected text (if any) from the editor.
func (e *Editor) SelectedText() string {
	e.initBuffer()
	e.scratch = e.text.SelectedText(e.scratch)
	return string(e.scratch)
}

// ClearSelection clears the selection, by setting the selection end equal to
// the selection start.
func (e *Editor) ClearSelection() {
	e.initBuffer()
	e.text.ClearSelection()
}

// Regions returns visible regions covering the rune range [start,end).
func (e *Editor) Regions(start, end int, regions []Region) []Region {
	e.initBuffer()
	return e.text.Regions(start, end, regions)
}

// returns start and end offset ratio of viewport
func (e *Editor) ViewPortRatio() (float32, float32) {
	textDims := e.text.FullDimensions()
	visibleDims := e.text.Dimensions()
	scrollOffY := e.text.ScrollOff().Y

	return float32(scrollOffY) / float32(textDims.Size.Y),
		float32(scrollOffY+visibleDims.Size.Y) / float32(textDims.Size.Y)
}

func (e *Editor) ScrollByRatio(gtx layout.Context, ratio float32) {
	textDims := e.text.FullDimensions()
	sdist := int(float32(textDims.Size.Y) * ratio)
	e.text.ScrollRel(0, sdist)
}

func (e *Editor) UpdateTextStyles(styles []*TextStyle) {
	e.textStyles = styles
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	default:
		return 0
	}
}

func (s ChangeEvent) isEditorEvent() {}
func (s SelectEvent) isEditorEvent() {}
