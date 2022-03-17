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

package misc

import (
	ui "github.com/gizak/termui/v3"
	vz "github.com/vyattalee/grofer/pkg/utils/visualization"
)

// HelpMenu is a wrapper widget around a List meant
// to display the help menu for a command. HelpMenu
// implements the ui.Drawable interface.
type HelpMenu struct {
	*vz.Table
	keybindings [][]string
}

// NewHelpMenu is a constructor for the HelpMenu type.
func NewHelpMenu() *HelpMenu {
	t := vz.NewTable()
	t.IsHelp = true
	return &HelpMenu{
		Table:       t,
		keybindings: getDefaultHelpKeybinding(),
	}
}

// ForCommand sets the keybindings to be displayed as part of the help
// for a specific command and returns the modified HelpMenu.
func (help *HelpMenu) ForCommand(command HelpKeybindingType) *HelpMenu {
	help.keybindings = getHelpKeybindingsForCommand(command)
	return help
}

// Resize resizes the widget based on specified width
// and height
func (help *HelpMenu) Resize(termWidth, termHeight int) {
	textWidth := 50
	for _, line := range help.keybindings {
		if textWidth < len(line[0]) {
			textWidth = len(line[0]) + 2
		}
	}
	textHeight := len(help.keybindings) + 3
	x := (termWidth - textWidth) / 2
	y := (termHeight - textHeight) / 2
	if x < 0 {
		x = 0
		textWidth = termWidth
	}
	if y < 0 {
		y = 0
		textHeight = termHeight
	}

	help.Table.SetRect(x, y, textWidth+x, textHeight+y)
}

// Draw puts the required text into the widget.
func (help *HelpMenu) Draw(buf *ui.Buffer) {
	help.Table.Title = " Keybindings "
	help.Table.Rows = help.keybindings
	help.Table.BorderStyle.Fg = ui.ColorCyan
	help.Table.BorderStyle.Bg = ui.ColorClear
	help.Table.ColResizer = func() {
		x := help.Table.Inner.Dx()
		help.Table.ColWidths = []int{x}
	}
	help.Table.Draw(buf)
}

// ensure interface compliance.
var _ ui.Drawable = (*HelpMenu)(nil)
