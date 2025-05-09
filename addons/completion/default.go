package completion

import (
	"errors"
	"image"
	"slices"

	"gioui.org/io/key"
	"gioui.org/layout"
	"github.com/oligo/gvcode"
)

var _ gvcode.Completion = (*DefaultCompletion)(nil)

type DefaultCompletion struct {
	Editor     *gvcode.Editor
	completors []*delegatedCompletor
	candicates []gvcode.CompletionCandidate
	session    *session
}

type delegatedCompletor struct {
	popup gvcode.CompletionPopup
	gvcode.Completor
}

func (dc *DefaultCompletion) AddCompletor(completor gvcode.Completor, popup gvcode.CompletionPopup) error {
	c := &delegatedCompletor{
		Completor: completor,
		popup:     popup,
	}

	trigger := completor.Trigger()

	duplicatedKey := slices.ContainsFunc(dc.completors, func(cm *delegatedCompletor) bool {
		tr := cm.Completor.Trigger()
		return tr.KeyBinding.Name == trigger.KeyBinding.Name && tr.KeyBinding.Modifiers == trigger.KeyBinding.Modifiers
	})
	if duplicatedKey {
		return errors.New("duplicated key binding")
	}

	if trigger.KeyBinding.Name != "" && trigger.KeyBinding.Modifiers != 0 {
		dc.Editor.RegisterCommand(dc,
			key.Filter{Name: trigger.KeyBinding.Name, Required: trigger.KeyBinding.Modifiers},
			func(gtx layout.Context, evt key.Event) gvcode.EditorEvent {
				dc.onKey(evt)
				return nil
			})
	}

	dc.completors = append(dc.completors, c)
	return nil
}

func canTrigger(tr gvcode.Trigger, input string) (bool, triggerKind) {
	// check explicit trigger characters.
	if slices.Contains(tr.Characters, input) {
		return true, charTrigger
	}

	// else check other allowed characters
	char := []rune(input)[0]
	if isSymbolChar(char) {
		return true, autoTrigger
	}

	return false, 0
}

func (dc *DefaultCompletion) triggerOnInput(ctx gvcode.CompletionContext) {
	if dc.session != nil && dc.session.IsValid() {
		dc.session.Update(ctx)
		return
	}

	var completor *delegatedCompletor
	var kind triggerKind

	for _, cmp := range dc.completors {
		yes, k := canTrigger(cmp.Trigger(), ctx.Input)
		if yes {
			completor = cmp
			kind = k
			break
		}
	}

	if completor != nil {
		if dc.session == nil {
			dc.session = newSession(completor, kind)
		}

		dc.session.Update(ctx)
	}
}

// onKey activates the completor when the registered key binding are pressed.
// The execution of the activated completor is repeated as the user type ahead,
// which is run by the OnText method.
func (dc *DefaultCompletion) onKey(evt key.Event) {
	// cancel existing completion.
	dc.Cancel()

	var cmp *delegatedCompletor
	for _, c := range dc.completors {
		if c.Trigger().ActivateOnKey(evt) {
			cmp = c
			break
		}
	}

	if cmp == nil {
		return
	}

	ctx := dc.Editor.GetCompletionContext()
	dc.session = newSession(cmp, keyTrigger)
	dc.session.Update(ctx)

	dc.runCompletor(ctx, cmp)
}

func (dc *DefaultCompletion) OnText(ctx gvcode.CompletionContext) {
	if ctx.Input == "" {
		dc.Cancel()
		return
	}

	dc.triggerOnInput(ctx)
	if !dc.session.IsValid() {
		return
	}

	dc.runCompletor(ctx, dc.session.Completor())
}

func (dc *DefaultCompletion) runCompletor(ctx gvcode.CompletionContext, completor *delegatedCompletor) {
	dc.candicates = dc.candicates[:0]
	if completor == nil {
		return
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
	if dc.session == nil {
		return layout.Dimensions{}
	}

	completor := dc.session.Completor()
	// when a session is marked as invalid, we'll have to still layout once to
	// reset the popup to unregister the event handler.
	if !dc.session.IsValid() {
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
		// No range is set, invalid candidate.
		logger.Error("No range is set, invalid candidate")
		return
	} else {
		caretStart, caretEnd := editRange.Start.Runes, editRange.End.Runes

		// Assume line/column is set, convert the line/column position to rune offsets.
		if caretStart <= 0 && caretEnd <= 0 {
			caretStart = dc.Editor.ConvertPos(editRange.Start.Line, editRange.Start.Column)
			caretEnd = dc.Editor.ConvertPos(editRange.End.Line, editRange.End.Column)
		}
		// set the selection using range provided by the completor.
		dc.Editor.SetCaret(caretStart, caretEnd)
	}

	dc.Editor.Insert(candidate.TextEdit.NewText)
	dc.Cancel()
}
