/*
Copyright © 2020 The PES Open Source Team pesos@pes.edu

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// This particularar widget is inspired and borrowed from the implementation of https://github.com/cjbassi/gotop

package utils

import (
	"fmt"
	"image"
	"log"
	"strings"

	ui "github.com/gizak/termui/v3"
)

const (
	// UpArrow is a symbol used when sorting tables
	UpArrow = "▲"
	// DownArrow is a symbol used when sorting tables
	DownArrow = "▼"
)

// Table is a custom table widget
type Table struct {
	*ui.Block

	Header []string
	Rows   [][]string

	// Different Styles for Header and Rows
	HeaderStyle ui.Style
	RowStyle    ui.Style

	ColWidths []int
	ColGap    int
	PadLeft   int

	ShowCursor  bool
	CursorColor ui.Color

	ShowLocation bool

	UniqueCol    int    // the column used to uniquely identify each table row
	SelectedItem string // used to keep the cursor on the correct item if the data changes
	SelectedRow  int
	TopRow       int // used to indicate where in the table we are scrolled at
	// Map that stores custom column colors
	ColColor   map[int]ui.Color
	ColResizer func()

	IsHelp             bool     // indicates if table is a help widget
	DefaultBorderColor ui.Color // indicates default border color
	ActiveBorderColor  ui.Color // indicates active border color
}

// NewTable returns a new Table instance
func NewTable() *Table {
	return &Table{
		Block:              ui.NewBlock(),
		HeaderStyle:        ui.NewStyle(ui.ColorClear, ui.ColorClear, ui.ModifierBold),
		RowStyle:           ui.NewStyle(ui.Theme.Default.Fg),
		SelectedRow:        0,
		TopRow:             0,
		UniqueCol:          0,
		ColResizer:         func() {},
		ColColor:           make(map[int]ui.Color),
		CursorColor:        ui.ColorCyan,
		DefaultBorderColor: ui.ColorCyan,
		ActiveBorderColor:  ui.ColorWhite,
	}
}

// Draw helps draw the Table widget onto the UI buffer
func (t *Table) Draw(buf *ui.Buffer) {
	t.Block.Draw(buf)

	if t.ShowLocation {
		t.drawLocation(buf)
	}

	t.ColResizer()

	// finds exact column starting position
	colXPos := []int{}
	cur := 1 + t.PadLeft
	for _, w := range t.ColWidths {
		colXPos = append(colXPos, cur)
		cur += w
		cur += t.ColGap
	}

	// prints header
	for i, h := range t.Header {
		width := t.ColWidths[i]
		if width == 0 {
			continue
		}
		// don't render column if it doesn't fit in widget
		if width > (t.Inner.Dx()-colXPos[i])+1 {
			continue
		}
		buf.SetString(
			h,
			t.HeaderStyle,
			image.Pt(t.Inner.Min.X+colXPos[i]-1, t.Inner.Min.Y),
		)
	}

	if t.TopRow < 0 {
		log.Printf("table widget TopRow value less than 0. TopRow: %v", t.TopRow)
		return
	}
	// prints each row
	for rowNum := t.TopRow; rowNum < t.TopRow+t.Inner.Dy()-1 && rowNum < len(t.Rows); rowNum++ {
		row := t.Rows[rowNum]
		y := (rowNum + 2) - t.TopRow
		// prints cursor
		style := t.RowStyle
		if t.IsHelp {
			if len(t.Rows[rowNum][0]) > 0 && string(t.Rows[rowNum][0][0]) != " " {
				style = t.HeaderStyle
			}
		}
		if t.ShowCursor {
			if (t.SelectedItem == "" && rowNum == t.SelectedRow) || (t.SelectedItem != "" && t.SelectedItem == row[t.UniqueCol]) {
				style.Fg = t.CursorColor
				style.Modifier = ui.ModifierReverse
				for _, width := range t.ColWidths {
					if width == 0 {
						continue
					}
					buf.SetString(
						strings.Repeat(" ", t.Inner.Dx()),
						style,
						image.Pt(t.Inner.Min.X, t.Inner.Min.Y+y-1),
					)
				}
				t.SelectedItem = row[t.UniqueCol]
				t.SelectedRow = rowNum
			}
		}
		// prints each col of the row
		tempFgColor := style.Fg
		tempBgColor := style.Bg
		for i, width := range t.ColWidths {
			style.Fg = tempFgColor
			style.Bg = tempBgColor
			// Change Foreground color if the column number is in the ColColor list
			if val, ok := t.ColColor[i]; ok {
				if rowNum == t.SelectedRow && t.ShowCursor {
					style.Fg = t.CursorColor
				} else {
					style.Fg = val
				}
			}
			if width == 0 {
				continue
			}
			// don't render column if width is greater than distance to end of widget
			if width > (t.Inner.Dx()-colXPos[i])+1 {
				continue
			}
			r := ui.TrimString(row[i], width)
			buf.SetString(
				r,
				style,
				image.Pt(t.Inner.Min.X+colXPos[i]-1, t.Inner.Min.Y+y-1),
			)
		}
	}
}

func (t *Table) drawLocation(buf *ui.Buffer) {
	total := len(t.Rows)
	topRow := t.TopRow + 1
	bottomRow := t.TopRow + t.Inner.Dy() - 1
	if bottomRow > total {
		bottomRow = total
	}

	loc := fmt.Sprintf(" %d - %d of %d ", topRow, bottomRow, total)

	width := len(loc)
	buf.SetString(loc, t.TitleStyle, image.Pt(t.Max.X-width-2, t.Min.Y))
}

// Scrolling ///////////////////////////////////////////////////////////////////

// calcPos is used to calculate the cursor position and the current view into the table.
func (t *Table) calcPos() {
	t.SelectedItem = ""

	if t.SelectedRow < 0 {
		t.SelectedRow = 0
	}
	if t.SelectedRow < t.TopRow {
		t.TopRow = t.SelectedRow
	}

	if t.SelectedRow > len(t.Rows)-1 {
		t.SelectedRow = len(t.Rows) - 1
	}
	if t.SelectedRow > t.TopRow+(t.Inner.Dy()-2) {
		t.TopRow = t.SelectedRow - (t.Inner.Dy() - 2)
	}
}

// ScrollUp moves the cursor one position upwards
func (t *Table) ScrollUp() {
	t.SelectedRow--
	t.calcPos()
}

// ScrollDown moves the cursor one position downwards
func (t *Table) ScrollDown() {
	t.SelectedRow++
	t.calcPos()
}

// ScrollTop moves the cursor to the top
func (t *Table) ScrollTop() {
	t.SelectedRow = 0
	t.calcPos()
}

// ScrollBottom moves the cursor to the bottom
func (t *Table) ScrollBottom() {
	t.SelectedRow = len(t.Rows) - 1
	t.calcPos()
}

// ScrollHalfPageUp moves the cursor half a page up
func (t *Table) ScrollHalfPageUp() {
	t.SelectedRow = t.SelectedRow - (t.Inner.Dy()-2)/2
	t.calcPos()
}

// ScrollHalfPageDown moves the cursor half a page down
func (t *Table) ScrollHalfPageDown() {
	t.SelectedRow = t.SelectedRow + (t.Inner.Dy()-2)/2
	t.calcPos()
}

// ScrollPageUp moves the cursor a page up
func (t *Table) ScrollPageUp() {
	t.SelectedRow -= (t.Inner.Dy() - 2)
	t.calcPos()
}

// ScrollPageDown moves the cursor a page down
func (t *Table) ScrollPageDown() {
	t.SelectedRow += (t.Inner.Dy() - 2)
	t.calcPos()
}

// ScrollToIndex moves the cursor to a specified index
func (t *Table) ScrollToIndex(idx int) {
	if idx < 0 || idx >= len(t.Rows) {
		return
	}
	t.SelectedRow = idx
	t.calcPos()
}

// DisableCursor turns off the cursor and un highlights the table
func (t *Table) DisableCursor() {
	t.ShowCursor = false
	t.BorderStyle.Fg = t.DefaultBorderColor
}

// EnableCursor turns on the cursor and highlights the table
func (t *Table) EnableCursor() {
	t.ShowCursor = true
	t.BorderStyle.Fg = t.ActiveBorderColor
}

// ensure interface compliance.
var _ ui.Drawable = (*Table)(nil)
