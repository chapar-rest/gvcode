package completion

import (
	"slices"

	"github.com/oligo/gvcode"
)

type triggerKind uint8

const (
	autoTrigger triggerKind = iota
	charTrigger
	keyTrigger
)

type triggerState struct {
	triggerKind triggerKind
	// the actived completor.
	completor *delegatedCompletor
}

// A session is started when some trigger is activated, and is destroyed when
// the completion is canceled or confirmed.
type session struct {
	ctx      gvcode.CompletionContext
	state    *triggerState
	canceled bool
	// buffered text while the user types.
	buf []rune
	// input range of the cursor since when the session started and when completion
	// confirmed.
	prefixRange gvcode.EditRange
}

func newSession(completor *delegatedCompletor, kind triggerKind) *session {
	return &session{
		state: &triggerState{
			triggerKind: kind,
			completor:   completor,
		},
	}
}

func isSymbolChar(ch rune) bool {
	if (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '_' {
		return true
	}

	return false
}

func (s *session) Update(ctx gvcode.CompletionContext) {
	if s.canceled {
		return
	}

	if ctx.Input != "" {
		tr := s.state.completor.Trigger()
		if !slices.Contains(tr.Characters, ctx.Input) && !isSymbolChar([]rune(ctx.Input)[0]) {
			s.makeInvalid()
			return
		}
	}

	s.ctx = ctx
	s.buf = append(s.buf, []rune(ctx.Input)...)
	if s.prefixRange == (gvcode.EditRange{}) {
		start := ctx.Position
		start.Column = max(0, start.Column-len([]rune(ctx.Input)))
		start.Runes = 0
		s.prefixRange.Start = start
	}
	s.prefixRange.End = ctx.Position
	s.prefixRange.End.Runes = 0
}

func (s *session) makeInvalid() {
	s.canceled = true
}

func (s *session) IsValid() bool {
	return s != nil && s.state != nil && !s.canceled
}

// bufferedText returns text buffered since the session is triggered.
func (s *session) BufferedText() string {
	return string(s.buf)
}

func (s *session) PrefixRange() gvcode.EditRange {
	return s.prefixRange
}

func (s *session) Completor() *delegatedCompletor {
	return s.state.completor
}
