package main

import "strconv"

func (v *View) DisplayView() {
	tabsize := int(v.Buf.Settings["tabsize"].(float64))
	if v.Type == vtLog {
		// Log views should always follow the cursor...
		v.Relocate()
	}

	// We need to know the string length of the largest line number
	// so we can pad appropriately when displaying line numbers
	maxLineNumLength := len(strconv.Itoa(v.Buf.NumLines))

	if v.Buf.Settings["ruler"] == true {
		// + 1 for the little space after the line number
		v.lineNumOffset = maxLineNumLength + 1
	} else {
		v.lineNumOffset = 0
	}

	// We need to add to the line offset if there are gutter messages
	var hasGutterMessages bool
	for _, v := range v.messages {
		if len(v) > 0 {
			hasGutterMessages = true
		}
	}
	if hasGutterMessages {
		v.lineNumOffset += 2
	}

	if v.x != 0 {
		// One space for the extra split divider
		v.lineNumOffset++
	}

	xOffset := v.x + v.lineNumOffset
	yOffset := v.y

	height := v.Height
	width := v.Width
	left := v.leftCol
	top := v.Topline

	v.cellview.Draw(v.Buf, top, height, left, width-v.lineNumOffset)

	screenX := v.x
	realLineN := top - 1
	visualLineN := 0
	var line []*Char
	for visualLineN, line = range v.cellview.lines {
		var firstChar *Char
		if len(line) > 0 {
			firstChar = line[0]
		}

		var softwrapped bool
		if firstChar != nil {
			if firstChar.realLoc.Y == realLineN {
				softwrapped = true
			}
			realLineN = firstChar.realLoc.Y
		} else {
			realLineN++
		}

		screenX = v.x

		if v.x != 0 {
			// Draw the split divider
			screen.SetContent(screenX, yOffset+visualLineN, '|', nil, defStyle.Reverse(true))
			screenX++
		}

		lineStr := v.Buf.Line(realLineN)

		// If there are gutter messages we need to display the '>>' symbol here
		if hasGutterMessages {
			// msgOnLine stores whether or not there is a gutter message on this line in particular
			msgOnLine := false
			for k := range v.messages {
				for _, msg := range v.messages[k] {
					if msg.lineNum == realLineN {
						msgOnLine = true
						gutterStyle := defStyle
						switch msg.kind {
						case GutterInfo:
							if style, ok := colorscheme["gutter-info"]; ok {
								gutterStyle = style
							}
						case GutterWarning:
							if style, ok := colorscheme["gutter-warning"]; ok {
								gutterStyle = style
							}
						case GutterError:
							if style, ok := colorscheme["gutter-error"]; ok {
								gutterStyle = style
							}
						}
						v.drawCell(screenX, yOffset+visualLineN, '>', nil, gutterStyle)
						screenX++
						v.drawCell(screenX, yOffset+visualLineN, '>', nil, gutterStyle)
						screenX++
						if v.Cursor.Y == realLineN && !messenger.hasPrompt {
							messenger.Message(msg.msg)
							messenger.gutterMessage = true
						}
					}
				}
			}
			// If there is no message on this line we just display an empty offset
			if !msgOnLine {
				v.drawCell(screenX, yOffset+visualLineN, ' ', nil, defStyle)
				screenX++
				v.drawCell(screenX, yOffset+visualLineN, ' ', nil, defStyle)
				screenX++
				if v.Cursor.Y == realLineN && messenger.gutterMessage {
					messenger.Reset()
					messenger.gutterMessage = false
				}
			}
		}

		lineNumStyle := defStyle
		if v.Buf.Settings["ruler"] == true {
			// Write the line number
			if style, ok := colorscheme["line-number"]; ok {
				lineNumStyle = style
			}
			if style, ok := colorscheme["current-line-number"]; ok {
				if realLineN == v.Cursor.Y && tabs[curTab].CurView == v.Num && !v.Cursor.HasSelection() {
					lineNumStyle = style
				}
			}

			lineNum := strconv.Itoa(realLineN + 1)

			// Write the spaces before the line number if necessary
			for i := 0; i < maxLineNumLength-len(lineNum); i++ {
				screen.SetContent(screenX, yOffset+visualLineN, ' ', nil, lineNumStyle)
				screenX++
			}
			if softwrapped && visualLineN != 0 {
				// Pad without the line number because it was written on the visual line before
				for range lineNum {
					screen.SetContent(screenX, yOffset+visualLineN, ' ', nil, lineNumStyle)
					screenX++
				}
			} else {
				// Write the actual line number
				for _, ch := range lineNum {
					screen.SetContent(screenX, yOffset+visualLineN, ch, nil, lineNumStyle)
					screenX++
				}
			}

			// Write the extra space
			screen.SetContent(screenX, yOffset+visualLineN, ' ', nil, lineNumStyle)
			screenX++
		}

		var lastChar *Char
		for _, char := range line {
			if char != nil {
				lineStyle := char.style

				charLoc := char.realLoc
				if v.Cursor.HasSelection() &&
					(charLoc.GreaterEqual(v.Cursor.CurSelection[0]) && charLoc.LessThan(v.Cursor.CurSelection[1]) ||
						charLoc.LessThan(v.Cursor.CurSelection[0]) && charLoc.GreaterEqual(v.Cursor.CurSelection[1])) {
					// The current character is selected
					lineStyle = defStyle.Reverse(true)

					if style, ok := colorscheme["selection"]; ok {
						lineStyle = style
					}

					width := StringWidth(string(char.char), tabsize)
					for i := 1; i < width; i++ {
						screen.SetContent(xOffset+char.visualLoc.X+i, yOffset+char.visualLoc.Y, ' ', nil, lineStyle)
					}
				}

				if tabs[curTab].CurView == v.Num && !v.Cursor.HasSelection() &&
					v.Cursor.Y == char.realLoc.Y && v.Cursor.X == char.realLoc.X {
					screen.ShowCursor(xOffset+char.visualLoc.X, yOffset+char.visualLoc.Y)
				}

				if v.Buf.Settings["cursorline"].(bool) && tabs[curTab].CurView == v.Num &&
					!v.Cursor.HasSelection() && v.Cursor.Y == realLineN {
					style := GetColor("cursor-line")
					fg, _, _ := style.Decompose()
					lineStyle = lineStyle.Background(fg)

					width := StringWidth(string(char.char), tabsize)
					for i := 1; i < width; i++ {
						screen.SetContent(xOffset+char.visualLoc.X+i, yOffset+char.visualLoc.Y, ' ', nil, lineStyle)
					}
				}

				screen.SetContent(xOffset+char.visualLoc.X, yOffset+char.visualLoc.Y, char.drawChar, nil, lineStyle)

				lastChar = char
			}
		}

		lastX := 0
		var realLoc Loc
		var visualLoc Loc
		if lastChar != nil {
			if tabs[curTab].CurView == v.Num && !v.Cursor.HasSelection() &&
				v.Cursor.Y == lastChar.realLoc.Y && v.Cursor.X == lastChar.realLoc.X+1 {
				screen.ShowCursor(xOffset+StringWidth(string(lineStr), tabsize), yOffset+lastChar.visualLoc.Y)
			}
			lastX = xOffset + StringWidth(string(lineStr), tabsize)
			realLoc = Loc{lastChar.realLoc.X, realLineN}
			visualLoc = Loc{lastX - xOffset, lastChar.visualLoc.Y}
		} else if len(line) == 0 {
			if tabs[curTab].CurView == v.Num && !v.Cursor.HasSelection() &&
				v.Cursor.Y == realLineN {
				screen.ShowCursor(xOffset, yOffset+visualLineN)
			}
			lastX = xOffset
			realLoc = Loc{0, realLineN}
			visualLoc = Loc{0, visualLineN}
		}

		if v.Cursor.HasSelection() &&
			(realLoc.GreaterEqual(v.Cursor.CurSelection[0]) && realLoc.LessThan(v.Cursor.CurSelection[1]) ||
				realLoc.LessThan(v.Cursor.CurSelection[0]) && realLoc.GreaterEqual(v.Cursor.CurSelection[1])) {
			// The current character is selected
			selectStyle := defStyle.Reverse(true)

			if style, ok := colorscheme["selection"]; ok {
				selectStyle = style
			}
			screen.SetContent(xOffset+visualLoc.X, yOffset+visualLoc.Y, ' ', nil, selectStyle)
		}

		if v.Buf.Settings["cursorline"].(bool) && tabs[curTab].CurView == v.Num &&
			!v.Cursor.HasSelection() && v.Cursor.Y == realLineN {
			for i := lastX; i < xOffset+v.Width; i++ {
				style := GetColor("cursor-line")
				fg, _, _ := style.Decompose()
				style = style.Background(fg)
				screen.SetContent(i, yOffset+visualLineN, ' ', nil, style)
			}
		}
	}

	if v.x != 0 && visualLineN < v.Height {
		for i := visualLineN + 1; i < v.Height; i++ {
			screen.SetContent(v.x, yOffset+i, '|', nil, defStyle.Reverse(true))
		}
	}
}