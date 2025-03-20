package gvcode

import (
	"image"
	"regexp"
	"strings"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

// SearchBar represents a search interface for the editor
type SearchBar struct {
	// Editor is used to search in
	Editor *Editor
	// Visible controls whether the search bar is shown
	Visible bool
	// SearchTerm is the current search term
	SearchEditor widget.Editor
	// CaseSensitive controls case sensitivity for the search
	CaseSensitive widget.Bool
	// RegexSearch controls whether the search uses regex
	RegexSearch widget.Bool
	// SearchResults stores the matches from the last search
	SearchResults []TextRange
	// CurrentMatch is the index of the currently highlighted match
	CurrentMatch int
	// NextButton navigates to the next match
	NextButton widget.Clickable
	// PrevButton navigates to the previous match
	PrevButton widget.Clickable
	// CloseButton closes the search bar
	CloseButton widget.Clickable

	// Height represents the height of the search bar
	Height unit.Dp
	// Material theme for rendering
	Theme *material.Theme
}

// NewSearchBar creates a new search bar
func NewSearchBar(editor *Editor, theme *material.Theme) *SearchBar {
	sb := &SearchBar{
		Editor: editor,
		Height: unit.Dp(40),
		Theme:  theme,
	}

	// Initialize the search editor
	sb.SearchEditor.SingleLine = true
	sb.SearchEditor.Submit = true

	return sb
}

// ToggleVisibility toggles the visibility of the search bar
func (sb *SearchBar) ToggleVisibility() {
	sb.Visible = !sb.Visible
}

// Close hides the search bar
func (sb *SearchBar) Close() {
	sb.Visible = false
}

// Search performs a search operation with the current settings
func (sb *SearchBar) Search() {
	searchTerm := sb.SearchEditor.Text()

	// Clear previous results
	sb.SearchResults = nil
	sb.CurrentMatch = 0

	if searchTerm == "" {
		// Remove highlights when search is empty
		sb.Editor.SetHighlights(nil)
		return
	}

	text := sb.Editor.Text()

	// Perform search based on settings
	if sb.RegexSearch.Value {
		sb.searchWithRegex(text, searchTerm)
	} else {
		sb.searchWithString(text, searchTerm)
	}

	// Update highlights in the editor
	sb.updateHighlights()
}

// searchWithString performs a simple string search
func (sb *SearchBar) searchWithString(text, searchTerm string) {
	if !sb.CaseSensitive.Value {
		text = strings.ToLower(text)
		searchTerm = strings.ToLower(searchTerm)
	}

	// Find all occurrences
	var pos int
	for {
		found := strings.Index(text[pos:], searchTerm)
		if found == -1 {
			break
		}

		matchPos := pos + found
		sb.SearchResults = append(sb.SearchResults, TextRange{
			Start: matchPos,
			End:   matchPos + len(searchTerm),
		})

		pos = matchPos + 1 // Move past this match to find the next one
	}
}

// searchWithRegex performs a regex-based search
func (sb *SearchBar) searchWithRegex(text, pattern string) {
	var regexFlags string
	if !sb.CaseSensitive.Value {
		regexFlags = "(?i)"
	}

	re, err := regexp.Compile(regexFlags + pattern)
	if err != nil {
		// Handle invalid regex
		return
	}

	// Find all matches
	matches := re.FindAllStringIndex(text, -1)
	for _, match := range matches {
		sb.SearchResults = append(sb.SearchResults, TextRange{
			Start: match[0],
			End:   match[1],
		})
	}
}

// updateHighlights updates the highlighted regions in the editor
func (sb *SearchBar) updateHighlights() {
	if len(sb.SearchResults) == 0 {
		sb.Editor.SetHighlights(nil)
		return
	}

	// Apply all matches as highlights
	sb.Editor.SetHighlights(sb.SearchResults)

	// Navigate to the first or current match
	if sb.CurrentMatch < len(sb.SearchResults) {
		match := sb.SearchResults[sb.CurrentMatch]
		sb.Editor.SetCaret(match.Start, match.End)
		sb.Editor.scrollCaret = true
	}
}

// NextMatch navigates to the next match
func (sb *SearchBar) NextMatch() {
	if len(sb.SearchResults) == 0 {
		return
	}

	sb.CurrentMatch = (sb.CurrentMatch + 1) % len(sb.SearchResults)
	match := sb.SearchResults[sb.CurrentMatch]
	sb.Editor.SetCaret(match.Start, match.End)
	sb.Editor.scrollCaret = true
}

// PrevMatch navigates to the previous match
func (sb *SearchBar) PrevMatch() {
	if len(sb.SearchResults) == 0 {
		return
	}

	sb.CurrentMatch = (sb.CurrentMatch - 1 + len(sb.SearchResults)) % len(sb.SearchResults)
	match := sb.SearchResults[sb.CurrentMatch]
	sb.Editor.SetCaret(match.Start, match.End)
	sb.Editor.scrollCaret = true
}

// Layout renders the search bar
func (sb *SearchBar) Layout(gtx layout.Context) layout.Dimensions {
	if !sb.Visible {
		return layout.Dimensions{}
	}

	// Process events
	for {
		event, ok := sb.SearchEditor.Update(gtx)
		if !ok {
			break
		}

		switch event.(type) {
		case widget.SubmitEvent:
			sb.Search()
		case widget.ChangeEvent:
			sb.Search()
		}
	}

	// Process button clicks
	if sb.NextButton.Clicked(gtx) {
		sb.NextMatch()
	}

	if sb.PrevButton.Clicked(gtx) {
		sb.PrevMatch()
	}

	if sb.CloseButton.Clicked(gtx) {
		sb.Close()
	}

	// Handle changes to search settings
	if sb.CaseSensitive.Update(gtx) || sb.RegexSearch.Update(gtx) {
		sb.Search()
	}

	borderColor := sb.Theme.Fg
	borderColor.A = 0xb6
	return widget.Border{
		Color: borderColor,
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		// Layout the search bar
		return layout.Background{}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				// Background
				rect := image.Rectangle{
					Max: image.Point{X: gtx.Constraints.Max.X, Y: gtx.Dp(sb.Height)},
				}
				paint.FillShape(gtx.Ops, sb.Theme.Bg, clip.Rect(rect).Op())
				return layout.Dimensions{Size: rect.Max}
			},
			func(gtx layout.Context) layout.Dimensions {
				// Content
				return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								editor := material.Editor(sb.Theme, &sb.SearchEditor, "Search...")
								return editor.Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								// Match count label
								count := material.Body2(sb.Theme,
									formatMatchCount(len(sb.SearchResults), sb.CurrentMatch))
								return count.Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								// Previous button
								btn := material.IconButton(sb.Theme, &sb.PrevButton, prevIcon, "Previous")
								btn.Size = unit.Dp(20)
								btn.Inset = layout.UniformInset(unit.Dp(4))
								return btn.Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								// Next button
								btn := material.IconButton(sb.Theme, &sb.NextButton, nextIcon, "Next")
								btn.Size = unit.Dp(20)
								btn.Inset = layout.UniformInset(unit.Dp(4))
								return btn.Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								// Case sensitive checkbox
								return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return material.CheckBox(sb.Theme, &sb.CaseSensitive, "Aa").Layout(gtx)
									}),
								)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								// Regex checkbox
								return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return material.CheckBox(sb.Theme, &sb.RegexSearch, ".*").Layout(gtx)
									}),
								)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								// Close button
								btn := material.IconButton(sb.Theme, &sb.CloseButton, closeIcon, "Close")
								btn.Size = unit.Dp(20)
								btn.Inset = layout.UniformInset(unit.Dp(4))
								return btn.Layout(gtx)
							}),
						)
					},
				)
			},
		)

	})
}

// formatMatchCount formats the match count display
func formatMatchCount(total, current int) string {
	if total == 0 {
		return "No matches"
	}
	return string([]byte{byte(current + 1 + '0'), '/', byte(total + '0')})
}

// Icons for the search bar
var (
	prevIcon *widget.Icon = func() *widget.Icon {
		icon, _ := widget.NewIcon(icons.NavigationChevronLeft)
		return icon
	}()

	nextIcon *widget.Icon = func() *widget.Icon {
		icon, _ := widget.NewIcon(icons.NavigationChevronRight)
		return icon
	}()

	closeIcon *widget.Icon = func() *widget.Icon {
		icon, _ := widget.NewIcon(icons.NavigationClose)
		return icon
	}()
)
