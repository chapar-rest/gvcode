package main

import (
	"image/color"
	"os"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/oligo/gvcode"
	"github.com/oligo/gvcode/editor"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

type EditorApp struct {
	window *app.Window
	th     *material.Theme
	state  *editor.Editor
	conf   *gvcode.EditorConf
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
			ed.layout(gtx)
			e.Frame(gtx.Ops)
		}
	}
}

func (ed *EditorApp) layout(gtx C) D {
	return gvcode.NewEditor(ed.state, ed.conf, "").Layout(gtx)
}

func main() {
	th := material.NewTheme()

	editorApp := EditorApp{
		window: &app.Window{},
		th:     th,
	}

	editorApp.conf = &gvcode.EditorConf{
		Shaper:             th.Shaper,
		TextColor:          th.Fg,
		Bg:                 th.Bg,
		SelectionColor:     th.ContrastBg,
		TypeFace:           font.Typeface("monospace"),
		TextSize:           unit.Sp(14),
		Weight:             font.Normal,
		LineHeightScale:    1.5,
		ShowLineNum:        true,
		LineNumPadding:     unit.Dp(24),
		LineHighlightColor: th.ContrastBg,
		TextMatchColor:     color.NRGBA{R: 255, G: 100, B: 100, A: 0x96},
	}

	editorApp.state = &editor.Editor{}

	go func() {
		err := editorApp.run()
		if err != nil {
			os.Exit(1)
		}

		os.Exit(0)
	}()

	app.Main()

}
