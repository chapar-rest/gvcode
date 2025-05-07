package gvcode

import (
	"image"
	"strings"
	"unicode/utf8"

	"gioui.org/io/key"
	"gioui.org/layout"
)

// Position is a position in the eidtor. Line/column and Runes may not
// be set at the same time depending on the use cases.
type Position struct {
	// Line number of the caret where the typing is happening.
	Line int
	// Column is the rune offset from the start of the line.
	Column int
	// Runes is the rune offset in the editor text of the input.
	Runes int
}

type EditRange struct {
	Start Position
	End   Position
}

// Completion is the main auto-completion interface for the editor. A Completion object
// schedules flow between the editor, the visual popup widget and completion algorithms(the Completor).
type Completion interface {
	// AddCompletors adds Completors to Completion. Completors should run independently and return
	// candicates to Completion. All candicates are then re-ranked and presented to the user.
	AddCompletor(completor Completor, popup CompletionPopup, trigger Trigger) error

	// OnText update the completion context. If there is no ongoing session, it should start one.
	OnText(ctx CompletionContext)
	// OnConfirm set a callback which is called when the user selected the candidates.
	OnConfirm(idx int)
	// Cancel cancels the current completion session.
	Cancel()
	// IsActive reports if the completion popup is visible.
	IsActive() bool

	// Offset returns the offset used to locate the popup when painting.
	Offset() image.Point
	// Layout layouts the completion selection box as popup near the caret.
	Layout(gtx layout.Context) layout.Dimensions
}

type CompletionPopup interface {
	Layout(gtx layout.Context, items []CompletionCandidate) layout.Dimensions
}

type CompletionContext struct {
	// Prefix is the text before the caret.
	Prefix string
	// Suffix is the text after the caret.
	Suffix string
	// Coordinates of the caret. Scroll off will change after we update the position,
	// so we use doc view position instead of viewport position.
	Coords image.Point
	// The position of the caret in line/column and selection range.
	Position Position
}

// CompletionCandidate are results returned from Completor, to be presented
// to the user to select from.
type CompletionCandidate struct {
	// Label is a short text shown to user to indicate
	// what the candicate looks like.
	Label string
	// TextEdit is the real text with range info to be
	// inserted into the editor.
	TextEdit TextEdit
	// A short description of the candicate.
	Description string
	// Kind of the candicate, for example, function,
	// class, keywords etc.
	Kind string
}

// TextEdit is the text with range info to be
// inserted into the editor, used in auto-completion.
type TextEdit struct {
	NewText   string
	EditRange EditRange
}

// Completor defines a interface that each of the delegated completor must implement.
type Completor interface {
	Suggest(ctx CompletionContext) []CompletionCandidate
}

// Trigger
type Trigger struct {
	// The minimum length in runes of the prefix to trigger completion.
	//
	// This is mutually exclusive with Prefix.
	MinSize int

	// Prefix that must be present to trigger the completion.
	// If it is empty, any character will trigger the completion. Prefix should
	// be removed when doing the completion, and should not be inserted when the
	// completion is confirmed.
	//
	// This is mutually exclusive with MinSize.
	Prefix string

	// Special key binding triggers the completion.
	KeyBinding struct {
		Name      key.Name
		Modifiers key.Modifiers
	}
}

func (tr Trigger) ActivateOnKey(evt key.Event) bool {
	return tr.KeyBinding.Name == evt.Name &&
		evt.Modifiers.Contain(tr.KeyBinding.Modifiers)
}

func (tr Trigger) ActivateOnPrefix(prefix string) bool {
	return tr.Prefix != "" && strings.HasPrefix(prefix, tr.Prefix)
}

func (tr Trigger) ActivateOnPrefixLen(prefix string) bool {
	return prefix != "" && utf8.RuneCountInString(prefix) >= tr.MinSize
}
