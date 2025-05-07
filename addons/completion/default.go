package completion

import (
	"errors"
	"image"
	"slices"
	"strings"
	"unicode/utf8"

	"gioui.org/io/key"
	"gioui.org/layout"
	"github.com/oligo/gvcode"
)

type triggerKind uint8

const (
	prefixLenTrigger triggerKind = iota
	prefixTrigger
	keyTrigger
)

var _ gvcode.Completion = (*DefaultCompletion)(nil)

type DefaultCompletion struct {
	Editor     *gvcode.Editor
	completors []delegatedCompletor
	triggers   map[*gvcode.Trigger]int
	candicates []gvcode.CompletionCandidate
	session    *session
}

type delegatedCompletor struct {
	trigger gvcode.Trigger
	popup   gvcode.CompletionPopup
	gvcode.Completor
}

// A session is started when some trigger is activated, and is destroyed when
// the completion is canceled or confirmed.
type session struct {
	ctx         gvcode.CompletionContext
	triggerKind triggerKind
	// the actived trigger.
	trigger *gvcode.Trigger
	invalid bool
}

func newSession() *session {
	return &session{}
}

func (s *session) update(ctx gvcode.CompletionContext) {
	if s.invalid {
		return
	}

	s.ctx = ctx
}

// A session triggered by a specific key binding cannot be
// re-triggered by other triggers.
func (s *session) triggerBy(tr *gvcode.Trigger, kind triggerKind) {
	if s.invalid {
		return
	}

	if s.trigger != nil && s.triggerKind == keyTrigger {
		return
	}

	s.triggerKind = kind
	s.trigger = tr
}

func (s *session) makeInvalid() {
	s.invalid = true
}

func (s *session) isValid() bool {
	return s != nil && !s.invalid && s.trigger != nil
}

func (dc *DefaultCompletion) AddCompletor(completor gvcode.Completor, popup gvcode.CompletionPopup, trigger gvcode.Trigger) error {
	c := delegatedCompletor{
		Completor: completor,
		popup:     popup,
		trigger:   trigger,
	}

	if trigger.MinSize > 0 && trigger.Prefix != "" {
		return errors.New("invalid trigger: be sure to set MinSize or Prefix, not both")
	}

	duplicatedKey := slices.ContainsFunc(dc.completors, func(cm delegatedCompletor) bool {
		return cm.trigger.KeyBinding.Name == trigger.KeyBinding.Name &&
			cm.trigger.KeyBinding.Modifiers == trigger.KeyBinding.Modifiers
	})
	if duplicatedKey {
		return errors.New("duplicated key binding")
	}

	if c.trigger.KeyBinding.Name != "" && c.trigger.KeyBinding.Modifiers != 0 {
		dc.Editor.RegisterCommand(dc,
			key.Filter{Name: c.trigger.KeyBinding.Name, Required: c.trigger.KeyBinding.Modifiers},
			func(gtx layout.Context, evt key.Event) gvcode.EditorEvent {
				dc.onKey(evt)
				return nil
			})
	}

	idx := len(dc.completors)
	if dc.triggers == nil {
		dc.triggers = make(map[*gvcode.Trigger]int)
	}
	dc.triggers[&c.trigger] = idx

	dc.completors = append(dc.completors, c)
	return nil
}

func (dc *DefaultCompletion) activeCompletor() *delegatedCompletor {
	if dc.session == nil || dc.session.trigger == nil {
		return nil
	}

	idx := dc.triggers[dc.session.trigger]
	if idx < 0 || idx >= len(dc.completors) {
		return nil
	}

	return &dc.completors[idx]
}

// onKey activates the completor when the registered key binding are pressed.
// If there is a valid prefix to help to complete, the activated completor is run once.
// The execution of the activated is repeated as the user type ahead, which is run by
// the OnText method.
func (dc *DefaultCompletion) onKey(evt key.Event) {
	// cancel existing completion.
	dc.Cancel()

	var trigger *gvcode.Trigger
	for tr := range dc.triggers {
		if tr.ActivateOnKey(evt) {
			trigger = tr
			break
		}
	}

	if trigger == nil {
		return
	}

	ctx := dc.Editor.GetCompletionContext()
	dc.session = newSession()
	dc.session.update(ctx)
	dc.session.triggerBy(trigger, keyTrigger)

	dc.runCompletor(ctx, dc.activeCompletor())
}

func (dc *DefaultCompletion) OnText(ctx gvcode.CompletionContext) {
	var trigger *gvcode.Trigger
	var kind triggerKind
	for tr := range dc.triggers {
		if tr.ActivateOnPrefix(ctx.Prefix) {
			trigger = tr
			kind = prefixTrigger
			break
		}

		if tr.ActivateOnPrefixLen(ctx.Prefix) {
			trigger = tr
			kind = prefixLenTrigger
			break
		}
	}

	if trigger != nil {
		if dc.session == nil {
			dc.session = newSession()
		}

		dc.session.update(ctx)
		dc.session.triggerBy(trigger, kind)
	} else {
		if dc.session != nil && dc.session.triggerKind != keyTrigger {
			// no activated prefix trigger, and no key trigger.
			dc.Cancel()
			return
		}
	}

	if !dc.session.isValid() {
		return
	}

	dc.runCompletor(ctx, dc.activeCompletor())
}

func (dc *DefaultCompletion) runCompletor(ctx gvcode.CompletionContext, completor *delegatedCompletor) {
	dc.candicates = dc.candicates[:0]
	if completor == nil {
		return
	}

	if dc.session.triggerKind == prefixTrigger {
		ctx.Prefix = strings.TrimPrefix(ctx.Prefix, dc.session.trigger.Prefix)
	}

	items := completor.Suggest(ctx)
	dc.candicates = append(dc.candicates, items...)

	if len(dc.candicates) == 0 {
		dc.Cancel()
		return
	}
}

func (dc *DefaultCompletion) IsActive() bool {
	return dc.session != nil
}

func (dc *DefaultCompletion) Offset() image.Point {
	if dc.session == nil {
		return image.Point{}
	}

	return dc.session.ctx.Coords
}

func (dc *DefaultCompletion) Layout(gtx layout.Context) layout.Dimensions {
	completor := dc.activeCompletor()
	if completor == nil {
		return layout.Dimensions{}
	}

	if !dc.session.isValid() {
		dc.session = nil
	}

	return completor.popup.Layout(gtx, dc.candicates)
}

func (dc *DefaultCompletion) Cancel() {
	if dc.session != nil {
		dc.session.makeInvalid()
	}
	dc.candicates = dc.candicates[:0]
}

func (dc *DefaultCompletion) OnConfirm(idx int) {
	if dc.Editor == nil {
		return
	}
	if idx < 0 || idx >= len(dc.candicates) {
		return
	}

	candidate := dc.candicates[idx]
	editRange := candidate.TextEdit.EditRange
	if editRange == (gvcode.EditRange{}) {
		// No range is set, replace the prefix with the candicate text.
		dc.Editor.SetCaret(dc.session.ctx.Position.Runes-utf8.RuneCountInString(dc.session.ctx.Prefix), dc.session.ctx.Position.Runes)
	} else {
		caretStart, caretEnd := editRange.Start.Runes, editRange.End.Runes
		// Line/column is set, convert the line/column position to rune offsets.
		if (editRange.Start != gvcode.Position{}) && editRange.End != (gvcode.Position{}) {
			caretStart = dc.Editor.ConvertPos(editRange.Start.Line, editRange.Start.Column)
			caretEnd = dc.Editor.ConvertPos(editRange.End.Line, editRange.End.Column)
		}
		// set the selection using range provided by the completor.
		dc.Editor.SetCaret(caretStart, caretEnd)
	}

	dc.Editor.Insert(candidate.TextEdit.NewText)
	dc.Cancel()
}
