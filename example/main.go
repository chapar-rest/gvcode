package main

import (
	"image/color"
	"log"
	"os"
	"regexp"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/oligo/gvcode"
	"github.com/oligo/gvcode/buffer"
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

func (ed *EditorApp) run() error {

	var ops op.Ops
	for {
		e := ed.window.Event()

		switch e := e.(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx C) D {
				return ed.layout(gtx, ed.th)
			})
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
			styles := HightlightTextByPattern(ed.state.Text(), "sample|demostrating")
			ed.state.UpdateTextStyles(styles)
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
			return widget.Border{
				Color: borderColor, Width: unit.Dp(1),
			}.Layout(gtx, func(gtx C) D {
				return layout.Inset{
					Top:    unit.Dp(6),
					Bottom: unit.Dp(6),
					Left:   unit.Dp(24),
					Right:  unit.Dp(24),
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					es := wg.NewEditor(th, ed.state)
					es.Font.Typeface = "monospace"
					es.TextSize = unit.Sp(14)
					es.LineHeightScale = 1.5
					es.TextHighlightColor = color.NRGBA{R: 120, G: 120, B: 120, A: 200}
					es.TextWeight = font.Normal

					return es.Layout(gtx)
				})
			})
		}),
	)

}

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	th := material.NewTheme()

	editorApp := EditorApp{
		window: &app.Window{},
		th:     th,
	}
	editorApp.window.Option(app.Title("Basic Example"))

	buffer.SetDebug(false)
	editorApp.state = &gvcode.Editor{
		// Have to set it to true as horizontal scrolling does not work yet.
		WrapLine: true,
	}
	editorApp.state.SetText("This is a sample text for demostrating purpose!")
	// editorApp.state.SetHighlights([]editor.TextRange{{Start: 0, End: 5}})
	styles := HightlightTextByPattern(editorApp.state.Text(), "sample|demostrating")
	editorApp.state.UpdateTextStyles(styles)

	go func() {
		err := editorApp.run()
		if err != nil {
			os.Exit(1)
		}

		os.Exit(0)
	}()

	app.Main()

}

func HightlightTextByPattern(text string, pattern string) []*gvcode.TextStyle {
	var styles []*gvcode.TextStyle

	re := regexp.MustCompile(pattern)
	matches := re.FindAllIndex([]byte(text), -1)
	for _, match := range matches {
		styles = append(styles, &gvcode.TextStyle{
			TextRange: gvcode.TextRange{
				Start: match[0],
				End:   match[1],
			},
			Color:      rgbaToOp(color.NRGBA{R: 255, A: 255}),
			Background: rgbaToOp(color.NRGBA{R: 215, G: 215, B: 215, A: 250}),
		})
	}

	return styles
}

func rgbaToOp(textColor color.NRGBA) op.CallOp {
	ops := new(op.Ops)

	m := op.Record(ops)
	paint.ColorOp{Color: textColor}.Add(ops)
	return m.Stop()
}
