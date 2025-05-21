package main

import (
	"fmt"
	"image/color"
	"log"
	_ "net/http/pprof" // This line registers the pprof handlers
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/oligo/gvcode"
	"github.com/oligo/gvcode/addons/completion"
	gvcolor "github.com/oligo/gvcode/color"
	"github.com/oligo/gvcode/textstyle/decoration"
	"github.com/oligo/gvcode/textstyle/syntax"
	wg "github.com/oligo/gvcode/widget"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

type EditorApp struct {
	window *app.Window
	th     *material.Theme
	state  *gvcode.Editor
}

const (
	syntaxPattern = "package|import|type|func|struct|for|var|switch|case|if|else"
)

func (ed *EditorApp) run() error {

	var ops op.Ops
	for {
		e := ed.window.Event()

		switch e := e.(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			ed.layout(gtx, ed.th)
			e.Frame(gtx.Ops)
		}
	}
}

func (ed *EditorApp) layout(gtx C, th *material.Theme) D {
	for {
		evt, ok := ed.state.Update(gtx)
		if !ok {
			break
		}

		switch evt.(type) {
		case gvcode.ChangeEvent:
			tokens := HightlightTextByPattern(ed.state.Text(), syntaxPattern)
			ed.state.SetSyntaxTokens(tokens...)
		}
	}

	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			lb := material.Label(th, th.TextSize, "gvcode editor")
			lb.Alignment = text.Middle
			return lb.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
		layout.Flexed(1, func(gtx C) D {
			borderColor := th.Fg
			borderColor.A = 0xb6
			return layout.Inset{
				Top:    unit.Dp(6),
				Bottom: unit.Dp(6),
				Left:   unit.Dp(6),
				Right:  unit.Dp(6),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				es := wg.NewEditor(th, ed.state)
				es.Font.Typeface = "monospace"
				es.Font.Weight = font.SemiBold
				es.TextSize = unit.Sp(14)
				es.LineHeightScale = 1.5

				return es.Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx C) D {
			line, col := ed.state.CaretPos()
			lb := material.Label(th, th.TextSize*0.8, fmt.Sprintf("Line:%d, Col:%d", line+1, col+1))
			lb.Alignment = text.End
			return lb.Layout(gtx)
		}),
	)

}

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	th := material.NewTheme()
	th.ContrastBg = color.NRGBA{R: 40, G: 204, B: 187, A: 255}
	//th.Bg = color.NRGBA{R: 32, G: 26, B: 16, A: 255}
	//th.Fg = color.NRGBA{R: 255, G: 255, B: 250, A: 255}

	editorApp := EditorApp{
		window: &app.Window{},
		th:     th,
	}
	editorApp.window.Option(app.Title("gvcode feature demo"))

	gvcode.SetDebug(false)
	editorApp.state = &gvcode.Editor{}

	thisFile, _ := os.ReadFile("./main.go")
	editorApp.state.SetText(string(thisFile))

	// Setting up auto-completion.
	cm := &completion.DefaultCompletion{Editor: editorApp.state}

	// set popup widget to let user navigate the candidates.
	popup := completion.NewCompletionPopup(editorApp.state, cm)
	popup.Theme = th
	popup.TextSize = unit.Sp(12)

	cm.AddCompletor(&goCompletor{editor: editorApp.state}, popup)

	// color scheme
	colorScheme := syntax.ColorScheme{}
	//colorScheme.Background = gvcolor.MakeColor(color.NRGBA{R: 32, G: 26, B: 16, A: 255})
	//colorScheme.Foreground = gvcolor.MakeColor(color.NRGBA{R: 255, G: 255, B: 250, A: 255})
	keywordColor, _ := gvcolor.Hex2Color("#AF00DB")
	colorScheme.AddTokenType("keyword", syntax.Underline, keywordColor, gvcolor.Color{})

	editorApp.state.WithOptions(
		gvcode.WrapLine(false),
		gvcode.WithAutoCompletion(cm),
		gvcode.WithColorScheme(colorScheme),
	)

	tokens := HightlightTextByPattern(editorApp.state.Text(), syntaxPattern)
	editorApp.state.SetSyntaxTokens(tokens...)

	highlightColor, _ := gvcolor.Hex2Color("#e74c3c50")
	highlightColor2, _ := gvcolor.Hex2Color("#f1c40f50")
	highlightColor3, _ := gvcolor.Hex2Color("#e74c3c")

	editorApp.state.AddDecorations(
		decoration.Decoration{Source: "test", Start: 5, End: 150, Background: &decoration.Background{Color: highlightColor}},
		decoration.Decoration{Source: "test", Start: 100, End: 200, Background: &decoration.Background{Color: highlightColor2}},
		decoration.Decoration{Source: "test", Start: 100, End: 200, Squiggle: &decoration.Squiggle{Color: highlightColor3}},
		decoration.Decoration{Source: "test", Start: 250, End: 400, Strikethrough: &decoration.Strikethrough{Color: highlightColor3}},
	)

	go func() {
		err := editorApp.run()
		if err != nil {
			os.Exit(1)
		}

		os.Exit(0)
	}()

	app.Main()

}

func HightlightTextByPattern(text string, pattern string) []syntax.Token {
	var tokens []syntax.Token

	re := regexp.MustCompile(pattern)
	matches := re.FindAllIndex([]byte(text), -1)
	for _, match := range matches {
		tokens = append(tokens, syntax.Token{
			Start:     match[0],
			End:       match[1],
			TokenType: "keyword",
		})
	}

	return tokens
}

var golangKeywords = []string{
	"break",
	"default",
	"func",
	"interface",
	"select",
	"case",
	"defer", "go", "map", "struct",
	"chan", "else", "goto", "package", "switch",
	"const", "fallthrough", "if", "range", "type",
	"continue", "for", "import", "return", "var",
}

type goCompletor struct {
	editor *gvcode.Editor
}

func isSymbolSeperator(ch rune) bool {
	if (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '_' {
		return false
	}

	return true
}

func (c *goCompletor) Trigger() gvcode.Trigger {
	return gvcode.Trigger{
		Characters: []string{"."},
		KeyBinding: struct {
			Name      key.Name
			Modifiers key.Modifiers
		}{
			Name: "P", Modifiers: key.ModShortcut,
		},
	}
}

func (c *goCompletor) Suggest(ctx gvcode.CompletionContext) []gvcode.CompletionCandidate {
	prefix := c.editor.ReadUntil(-1, isSymbolSeperator)
	candicates := make([]gvcode.CompletionCandidate, 0)
	for _, kw := range golangKeywords {
		if strings.Contains(kw, prefix) {
			candicates = append(candicates, gvcode.CompletionCandidate{
				Label: kw,
				TextEdit: gvcode.TextEdit{
					NewText: kw,
					EditRange: gvcode.EditRange{
						Start: gvcode.Position{Runes: ctx.Position.Runes - utf8.RuneCountInString(prefix)},
						End:   gvcode.Position{Runes: ctx.Position.Runes},
					},
				},
				Description: kw,
				Kind:        "text",
			})
		}
	}

	return candicates
}
