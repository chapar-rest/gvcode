package gvcode

import (
	"image"
	"io"
	"math"
	"strings"

	"gioui.org/gesture"
	"gioui.org/io/clipboard"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/io/transfer"
	"gioui.org/layout"
)

func (e *Editor) processEvents(gtx layout.Context) (ev EditorEvent, ok bool) {
	if len(e.pending) > 0 {
		out := e.pending[0]
		e.pending = e.pending[:copy(e.pending, e.pending[1:])]
		return out, true
	}
	selStart, selEnd := e.Selection()
	defer func() {
		afterSelStart, afterSelEnd := e.Selection()
		if selStart != afterSelStart || selEnd != afterSelEnd {
			if ok {
				e.pending = append(e.pending, SelectEvent{})
			} else {
				ev = SelectEvent{}
				ok = true
			}
		}
	}()

	ev, ok = e.processPointer(gtx)
	if ok {
		return ev, ok
	}
	ev, ok = e.processKey(gtx)
	if ok {
		return ev, ok
	}
	return nil, false
}

func (e *Editor) processPointer(gtx layout.Context) (EditorEvent, bool) {
	var scrollX, scrollY pointer.ScrollRange
	textDims := e.text.FullDimensions()
	visibleDims := e.text.Dimensions()

	scrollOffX := e.text.ScrollOff().X
	scrollX.Min = min(-scrollOffX, 0)
	scrollX.Max = max(0, textDims.Size.X-(scrollOffX+visibleDims.Size.X))

	scrollOffY := e.text.ScrollOff().Y
	scrollY.Min = -scrollOffY
	scrollY.Max = max(0, textDims.Size.Y-(scrollOffY+visibleDims.Size.Y))

	sbounds := e.text.ScrollBounds()
	var soff int
	var smin, smax int

	sdist := e.scroller.Update(gtx.Metric, gtx.Source, gtx.Now, gesture.Vertical, scrollX, scrollY)
	// Have to wait for the patch to be accepted by Gio dev team.
	// if e.scroller.Direction() == gesture.Horizontal {
	// 	e.text.ScrollRel(sdist, 0)
	// 	soff = e.text.ScrollOff().X
	// 	smin, smax = sbounds.Min.X, sbounds.Max.X
	// } else {
	e.text.ScrollRel(0, sdist)
	soff = e.text.ScrollOff().Y
	smin, smax = sbounds.Min.Y, sbounds.Max.Y
	//}

	for {
		evt, ok := e.clicker.Update(gtx.Source)
		if !ok {
			break
		}
		ev, ok := e.processPointerEvent(gtx, evt)
		if ok {
			return ev, ok
		}
	}
	for {
		evt, ok := e.dragger.Update(gtx.Metric, gtx.Source, gesture.Both)
		if !ok {
			break
		}
		ev, ok := e.processPointerEvent(gtx, evt)
		if ok {
			return ev, ok
		}
	}

	if (sdist > 0 && soff >= smax) || (sdist < 0 && soff <= smin) {
		e.scroller.Stop()
	}
	return nil, false
}

func (e *Editor) processPointerEvent(gtx layout.Context, ev event.Event) (EditorEvent, bool) {
	switch evt := ev.(type) {
	case gesture.ClickEvent:
		switch {
		case evt.Kind == gesture.KindPress && evt.Source == pointer.Mouse,
			evt.Kind == gesture.KindClick && evt.Source != pointer.Mouse:
			prevCaretPos, _ := e.text.Selection()
			e.blinkStart = gtx.Now
			e.text.MoveCoord(image.Point{
				X: int(math.Round(float64(evt.Position.X))),
				Y: int(math.Round(float64(evt.Position.Y))),
			})
			gtx.Execute(key.FocusCmd{Tag: e})
			if !e.readOnly {
				gtx.Execute(key.SoftKeyboardCmd{Show: true})
			}
			if e.scroller.State() != gesture.StateFlinging {
				e.scrollCaret = true
			}

			if evt.Modifiers == key.ModShift {
				start, end := e.text.Selection()
				// If they clicked closer to the end, then change the end to
				// where the caret used to be (effectively swapping start & end).
				if abs(end-start) < abs(start-prevCaretPos) {
					e.text.SetCaret(start, prevCaretPos)
				}
			} else {
				e.text.ClearSelection()
			}
			e.dragging = true

			// Process multi-clicks.
			switch {
			case evt.NumClicks == 2:
				e.text.MoveWords(-1, selectionClear)
				e.text.MoveWords(1, selectionExtend)
				e.dragging = false
			case evt.NumClicks >= 3:
				e.text.MoveLineStart(selectionClear)
				e.text.MoveLineEnd(selectionExtend)
				e.dragging = false
			}
		}
	case pointer.Event:
		release := false
		switch {
		case evt.Kind == pointer.Release && evt.Source == pointer.Mouse:
			release = true
			fallthrough
		case evt.Kind == pointer.Drag && evt.Source == pointer.Mouse:
			if e.dragging {
				e.blinkStart = gtx.Now
				e.text.MoveCoord(image.Point{
					X: int(math.Round(float64(evt.Position.X))),
					Y: int(math.Round(float64(evt.Position.Y))),
				})
				e.scrollCaret = true

				if release {
					e.dragging = false
				}
			}
		}
	}
	return nil, false
}

func condFilter(pred bool, f key.Filter) event.Filter {
	if pred {
		return f
	} else {
		return nil
	}
}

func (e *Editor) processKey(gtx layout.Context) (EditorEvent, bool) {
	if e.text.Changed() {
		return ChangeEvent{}, true
	}
	caret, _ := e.text.Selection()
	atBeginning := caret == 0
	atEnd := caret == e.text.Len()
	if gtx.Locale.Direction.Progression() != system.FromOrigin {
		atEnd, atBeginning = atBeginning, atEnd
	}
	filters := []event.Filter{
		key.FocusFilter{Target: e},
		transfer.TargetFilter{Target: e, Type: "application/text"},
		key.Filter{Focus: e, Name: key.NameEnter, Optional: key.ModShift},
		key.Filter{Focus: e, Name: key.NameReturn, Optional: key.ModShift},

		key.Filter{Focus: e, Name: "Z", Required: key.ModShortcut, Optional: key.ModShift},
		key.Filter{Focus: e, Name: "C", Required: key.ModShortcut},
		key.Filter{Focus: e, Name: "V", Required: key.ModShortcut},
		key.Filter{Focus: e, Name: "X", Required: key.ModShortcut},
		key.Filter{Focus: e, Name: "A", Required: key.ModShortcut},

		key.Filter{Focus: e, Name: key.NameDeleteBackward, Optional: key.ModShortcutAlt | key.ModShift},
		key.Filter{Focus: e, Name: key.NameDeleteForward, Optional: key.ModShortcutAlt | key.ModShift},

		key.Filter{Focus: e, Name: key.NameHome, Optional: key.ModShortcut | key.ModShift},
		key.Filter{Focus: e, Name: key.NameEnd, Optional: key.ModShortcut | key.ModShift},
		key.Filter{Focus: e, Name: key.NamePageDown, Optional: key.ModShift},
		key.Filter{Focus: e, Name: key.NamePageUp, Optional: key.ModShift},
		key.Filter{Focus: e, Name: key.NameTab},
		condFilter(!atBeginning, key.Filter{Focus: e, Name: key.NameLeftArrow, Optional: key.ModShortcutAlt | key.ModShift}),
		condFilter(!atBeginning, key.Filter{Focus: e, Name: key.NameUpArrow, Optional: key.ModShortcutAlt | key.ModShift}),
		condFilter(!atEnd, key.Filter{Focus: e, Name: key.NameRightArrow, Optional: key.ModShortcutAlt | key.ModShift}),
		condFilter(!atEnd, key.Filter{Focus: e, Name: key.NameDownArrow, Optional: key.ModShortcutAlt | key.ModShift}),
	}

	for {
		ke, ok := gtx.Event(filters...)
		if !ok {
			break
		}
		e.blinkStart = gtx.Now
		switch ke := ke.(type) {
		case key.FocusEvent:
			// Reset IME state.
			e.ime.imeState = imeState{}
			if ke.Focus && !e.readOnly {
				gtx.Execute(key.SoftKeyboardCmd{Show: true})
			}
		case key.Event:
			if !gtx.Focused(e) || ke.State != key.Press {
				break
			}
			e.scrollCaret = true
			e.scroller.Stop()
			ev, ok := e.command(gtx, ke)
			if ok {
				return ev, ok
			}
		case key.SnippetEvent:
			e.updateSnippet(gtx, ke.Start, ke.End)
		case key.EditEvent:
			if e.readOnly {
				break
			}
			e.scrollCaret = true
			e.scroller.Stop()
			e.replace(ke.Range.Start, ke.Range.End, ke.Text)
			// Reset caret xoff.
			e.text.MoveCaret(0, 0)
		// Complete a paste event, initiated by Shortcut-V in Editor.command().
		case transfer.DataEvent:
			e.scrollCaret = true
			e.scroller.Stop()
			content, err := io.ReadAll(ke.Open())
			if err == nil {
				if e.Insert(string(content)) != 0 {
					return ChangeEvent{}, true
				}
			}
		case key.SelectionEvent:
			e.scrollCaret = true
			e.scroller.Stop()
			e.text.SetCaret(ke.Start, ke.End)
		}
	}
	if e.text.Changed() {
		return ChangeEvent{}, true
	}
	return nil, false
}

func (e *Editor) command(gtx layout.Context, k key.Event) (EditorEvent, bool) {
	direction := 1
	if gtx.Locale.Direction.Progression() == system.TowardOrigin {
		direction = -1
	}
	moveByWord := k.Modifiers.Contain(key.ModShortcutAlt)
	selAct := selectionClear
	if k.Modifiers.Contain(key.ModShift) {
		selAct = selectionExtend
	}
	if k.Modifiers.Contain(key.ModShortcut) {
		switch k.Name {
		// Initiate a paste operation, by requesting the clipboard contents; other
		// half is in Editor.processKey() under clipboard.Event.
		case "V":
			if !e.readOnly {
				gtx.Execute(clipboard.ReadCmd{Tag: e})
			}
		// Copy or Cut selection -- ignored if nothing selected.
		case "C", "X":
			e.scratch = e.text.SelectedText(e.scratch)
			if text := string(e.scratch); text != "" {
				gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(strings.NewReader(text))})
				if k.Name == "X" && !e.readOnly {
					if e.Delete(1) != 0 {
						return ChangeEvent{}, true
					}
				}
			}
		// Select all
		case "A":
			e.text.SetCaret(0, e.text.Len())
		case "Z":
			if !e.readOnly {
				if k.Modifiers.Contain(key.ModShift) {
					if ev, ok := e.redo(); ok {
						return ev, ok
					}
				} else {
					if ev, ok := e.undo(); ok {
						return ev, ok
					}
				}
			}
		case key.NameHome:
			e.text.MoveTextStart(selAct)
		case key.NameEnd:
			e.text.MoveTextEnd(selAct)
		}
		return nil, false
	}
	switch k.Name {
	case key.NameReturn, key.NameEnter:
		if !e.readOnly {
			if e.Insert("\n") != 0 {
				return ChangeEvent{}, true
			}
		}
	case key.NameTab:
		if !e.readOnly {
			if e.Insert("\t") != 0 {
				return ChangeEvent{}, true
			}
		}
	case key.NameDeleteBackward:
		if !e.readOnly {
			if moveByWord {
				if e.deleteWord(-1) != 0 {
					return ChangeEvent{}, true
				}
			} else {
				if e.Delete(-1) != 0 {
					return ChangeEvent{}, true
				}
			}
		}
	case key.NameDeleteForward:
		if !e.readOnly {
			if moveByWord {
				if e.deleteWord(1) != 0 {
					return ChangeEvent{}, true
				}
			} else {
				if e.Delete(1) != 0 {
					return ChangeEvent{}, true
				}
			}
		}
	case key.NameUpArrow:
		e.text.MoveLines(-1, selAct)
	case key.NameDownArrow:
		e.text.MoveLines(+1, selAct)
	case key.NameLeftArrow:
		if moveByWord {
			e.text.MoveWords(-1*direction, selAct)
		} else {
			if selAct == selectionClear {
				e.text.ClearSelection()
			}
			e.text.MoveCaret(-1*direction, -1*direction*int(selAct))
		}
	case key.NameRightArrow:
		if moveByWord {
			e.text.MoveWords(1*direction, selAct)
		} else {
			if selAct == selectionClear {
				e.text.ClearSelection()
			}
			e.text.MoveCaret(1*direction, int(selAct)*direction)
		}
	case key.NamePageUp:
		e.text.MovePages(-1, selAct)
	case key.NamePageDown:
		e.text.MovePages(+1, selAct)
	case key.NameHome:
		e.text.MoveLineStart(selAct)
	case key.NameEnd:
		e.text.MoveLineEnd(selAct)
	}
	return nil, false
}

// updateSnippet queues a key.SnippetCmd if the snippet content or position
// have changed. off and len are in runes.
func (e *Editor) updateSnippet(gtx layout.Context, start, end int) {
	if start > end {
		start, end = end, start
	}
	length := e.text.Len()
	if start > length {
		start = length
	}
	if end > length {
		end = length
	}
	e.ime.start = start
	e.ime.end = end
	startOff := e.text.ByteOffset(start)
	endOff := e.text.ByteOffset(end)
	n := endOff - startOff
	if n > int64(len(e.ime.scratch)) {
		e.ime.scratch = make([]byte, n)
	}
	scratch := e.ime.scratch[:n]
	read, _ := e.buffer.ReadAt(scratch, startOff)

	if read != len(scratch) {
		panic("e.rr.Read truncated data")
	}
	newSnip := key.Snippet{
		Range: key.Range{
			Start: e.ime.start,
			End:   e.ime.end,
		},
		Text: e.ime.snippet.Text,
	}
	if string(scratch) != newSnip.Text {
		newSnip.Text = string(scratch)
	}
	if newSnip == e.ime.snippet {
		return
	}
	e.ime.snippet = newSnip
	gtx.Execute(key.SnippetCmd{Tag: e, Snippet: newSnip})
}
