package completion

import (
	"errors"
	"image"
	"slices"
	"strings"
	"time"

	"gioui.org/io/key"
	"gioui.org/layout"
	"github.com/oligo/gvcode"
)

var _ gvcode.Completion = (*DefaultCompletion)(nil)

type DefaultCompletion struct {
	Editor     *gvcode.Editor
	runner     *deferredRunner[gvcode.CompletionCandidate]
	completors []*delegatedCompletor
	candidates []gvcode.CompletionCandidate
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
	if dc.runner == nil {
		dc.runner = newRunner[gvcode.CompletionCandidate](0)
	}
	return nil
}

// SetDelay set a delay duration of run completion after the user
// stopped typing. This makes the UI more responsive when the user
// types fast as it reduces the unnecessary completion computation.
func (dc *DefaultCompletion) SetDelay(delay time.Duration) {
	if dc.runner == nil {
		dc.runner = newRunner[gvcode.CompletionCandidate](delay)
	} else {
		dc.runner.SetDelay(delay)
	}
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
	// Run completor without delay.
	dc.runner.Async(func() []gvcode.CompletionCandidate {
		return dc.runCompletor(ctx)
	})
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

	dc.runner.Run(func() []gvcode.CompletionCandidate {
		return dc.runCompletor(ctx)
	})
}

func (dc *DefaultCompletion) runCompletor(ctx gvcode.CompletionContext) []gvcode.CompletionCandidate {
	if !dc.session.IsValid() {
		return nil
	}

	completor := dc.session.Completor()
	if completor == nil {
		return nil
	}

	return completor.Suggest(ctx)

}

func (dc *DefaultCompletion) updateCandidates() {
	select {
	case items := <-dc.runner.ResultChan():
		dc.candidates = dc.candidates[:0]
		dc.candidates = append(dc.candidates, items...)
		if len(dc.candidates) == 0 {
			dc.Cancel()
		}
	default:
		// no update
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
	dc.updateCandidates()

	if dc.session == nil {
		return layout.Dimensions{}
	}

	completor := dc.session.Completor()
	// when a session is marked as invalid, we'll have to still layout once to
	// reset the popup to unregister the event handler.
	if !dc.session.IsValid() {
		dc.session = nil
	}

	return completor.popup.Layout(gtx, dc.candidates)
}

func (dc *DefaultCompletion) Cancel() {
	if dc.session != nil {
		dc.session.makeInvalid()
	}
	dc.candidates = dc.candidates[:0]
}

func (dc *DefaultCompletion) OnConfirm(idx int) {
	if dc.Editor == nil {
		return
	}
	if idx < 0 || idx >= len(dc.candidates) {
		return
	}

	candidate := dc.candidates[idx]
	editRange := candidate.TextEdit.EditRange
	if editRange == (gvcode.EditRange{}) {
		editRange = dc.session.PrefixRange()
	}

	caretStart, caretEnd := editRange.Start.Runes, editRange.End.Runes
	// Assume line/column is set, convert the line/column position to rune offsets.
	if caretStart <= 0 && caretEnd <= 0 {
		caretStart = dc.Editor.ConvertPos(editRange.Start.Line, editRange.Start.Column)
		caretEnd = dc.Editor.ConvertPos(editRange.End.Line, editRange.End.Column)
	}
	// set the selection using range provided by the completor.
	dc.Editor.SetCaret(caretStart, caretEnd)

	if strings.ToLower(candidate.TextFormat) == "snippet" {
		_, err := dc.Editor.InsertSnippet(candidate.TextEdit.NewText)
		if err != nil {
			logger.Error("insert snippet failed", "error", err)
		}
	} else {
		dc.Editor.Insert(candidate.TextEdit.NewText)
	}
	dc.Cancel()
}
