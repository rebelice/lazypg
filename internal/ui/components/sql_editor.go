package components

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/rebeliceyang/lazypg/internal/ui/theme"
)

// SQLEditorHeightPreset defines the height presets for the editor
type SQLEditorHeightPreset int

const (
	SQLEditorSmall  SQLEditorHeightPreset = iota // 20% of available height
	SQLEditorMedium                              // 35% of available height
	SQLEditorLarge                               // 50% of available height
)

// SQLEditor is a multiline SQL editor component
type SQLEditor struct {
	// Content
	lines      []string // Lines of text
	cursorRow  int      // Current cursor row (0-indexed)
	cursorCol  int      // Current cursor column (0-indexed)

	// Dimensions
	Width  int
	Height int

	// State
	expanded     bool
	heightPreset SQLEditorHeightPreset

	// Theme
	Theme theme.Theme

	// History
	history    []string
	historyIdx int
}

// NewSQLEditor creates a new SQL editor
func NewSQLEditor(th theme.Theme) *SQLEditor {
	return &SQLEditor{
		lines:        []string{""},
		cursorRow:    0,
		cursorCol:    0,
		expanded:     false,
		heightPreset: SQLEditorMedium,
		Theme:        th,
		history:      []string{},
		historyIdx:   -1,
	}
}

// IsExpanded returns whether the editor is expanded
func (e *SQLEditor) IsExpanded() bool {
	return e.expanded
}

// Toggle expands or collapses the editor
func (e *SQLEditor) Toggle() {
	e.expanded = !e.expanded
}

// Expand expands the editor
func (e *SQLEditor) Expand() {
	e.expanded = true
}

// Collapse collapses the editor
func (e *SQLEditor) Collapse() {
	e.expanded = false
}

// GetHeightPreset returns the current height preset
func (e *SQLEditor) GetHeightPreset() SQLEditorHeightPreset {
	return e.heightPreset
}

// IncreaseHeight increases the height preset
func (e *SQLEditor) IncreaseHeight() {
	if e.heightPreset < SQLEditorLarge {
		e.heightPreset++
	}
}

// DecreaseHeight decreases the height preset
func (e *SQLEditor) DecreaseHeight() {
	if e.heightPreset > SQLEditorSmall {
		e.heightPreset--
	}
}

// GetHeightRatio returns the height ratio for the current preset
func (e *SQLEditor) GetHeightRatio() float64 {
	switch e.heightPreset {
	case SQLEditorSmall:
		return 0.20
	case SQLEditorMedium:
		return 0.35
	case SQLEditorLarge:
		return 0.50
	default:
		return 0.35
	}
}

// GetContent returns the full content as a single string
func (e *SQLEditor) GetContent() string {
	return strings.Join(e.lines, "\n")
}

// SetContent sets the editor content
func (e *SQLEditor) SetContent(content string) {
	if content == "" {
		e.lines = []string{""}
	} else {
		e.lines = strings.Split(content, "\n")
	}
	e.cursorRow = len(e.lines) - 1
	e.cursorCol = len(e.lines[e.cursorRow])
}

// Clear clears the editor content
func (e *SQLEditor) Clear() {
	e.lines = []string{""}
	e.cursorRow = 0
	e.cursorCol = 0
}

// GetCollapsedHeight returns the height when collapsed (2 lines + border)
func (e *SQLEditor) GetCollapsedHeight() int {
	return 4 // 2 content lines + 2 border lines
}

// MoveCursorLeft moves cursor left
func (e *SQLEditor) MoveCursorLeft() {
	if e.cursorCol > 0 {
		e.cursorCol--
	} else if e.cursorRow > 0 {
		// Move to end of previous line
		e.cursorRow--
		e.cursorCol = len(e.lines[e.cursorRow])
	}
}

// MoveCursorRight moves cursor right
func (e *SQLEditor) MoveCursorRight() {
	if e.cursorCol < len(e.lines[e.cursorRow]) {
		e.cursorCol++
	} else if e.cursorRow < len(e.lines)-1 {
		// Move to start of next line
		e.cursorRow++
		e.cursorCol = 0
	}
}

// MoveCursorUp moves cursor up
func (e *SQLEditor) MoveCursorUp() {
	if e.cursorRow > 0 {
		e.cursorRow--
		// Clamp column to line length
		if e.cursorCol > len(e.lines[e.cursorRow]) {
			e.cursorCol = len(e.lines[e.cursorRow])
		}
	}
}

// MoveCursorDown moves cursor down
func (e *SQLEditor) MoveCursorDown() {
	if e.cursorRow < len(e.lines)-1 {
		e.cursorRow++
		// Clamp column to line length
		if e.cursorCol > len(e.lines[e.cursorRow]) {
			e.cursorCol = len(e.lines[e.cursorRow])
		}
	}
}

// MoveCursorToLineStart moves cursor to start of line
func (e *SQLEditor) MoveCursorToLineStart() {
	e.cursorCol = 0
}

// MoveCursorToLineEnd moves cursor to end of line
func (e *SQLEditor) MoveCursorToLineEnd() {
	e.cursorCol = len(e.lines[e.cursorRow])
}

// MoveCursorToDocStart moves cursor to start of document
func (e *SQLEditor) MoveCursorToDocStart() {
	e.cursorRow = 0
	e.cursorCol = 0
}

// MoveCursorToDocEnd moves cursor to end of document
func (e *SQLEditor) MoveCursorToDocEnd() {
	e.cursorRow = len(e.lines) - 1
	e.cursorCol = len(e.lines[e.cursorRow])
}

// InsertChar inserts a character at cursor position
func (e *SQLEditor) InsertChar(ch rune) {
	line := e.lines[e.cursorRow]
	// Insert character at cursor position
	newLine := line[:e.cursorCol] + string(ch) + line[e.cursorCol:]
	e.lines[e.cursorRow] = newLine
	e.cursorCol++
}

// InsertNewline inserts a new line at cursor position
func (e *SQLEditor) InsertNewline() {
	line := e.lines[e.cursorRow]
	// Split line at cursor
	before := line[:e.cursorCol]
	after := line[e.cursorCol:]

	e.lines[e.cursorRow] = before
	// Insert new line after current
	newLines := make([]string, len(e.lines)+1)
	copy(newLines[:e.cursorRow+1], e.lines[:e.cursorRow+1])
	newLines[e.cursorRow+1] = after
	copy(newLines[e.cursorRow+2:], e.lines[e.cursorRow+1:])
	e.lines = newLines

	e.cursorRow++
	e.cursorCol = 0
}

// DeleteCharBefore deletes character before cursor (backspace)
func (e *SQLEditor) DeleteCharBefore() {
	if e.cursorCol > 0 {
		// Delete character before cursor
		line := e.lines[e.cursorRow]
		e.lines[e.cursorRow] = line[:e.cursorCol-1] + line[e.cursorCol:]
		e.cursorCol--
	} else if e.cursorRow > 0 {
		// Merge with previous line
		prevLine := e.lines[e.cursorRow-1]
		currLine := e.lines[e.cursorRow]
		e.cursorCol = len(prevLine)
		e.lines[e.cursorRow-1] = prevLine + currLine
		// Remove current line
		e.lines = append(e.lines[:e.cursorRow], e.lines[e.cursorRow+1:]...)
		e.cursorRow--
	}
}

// DeleteCharAfter deletes character after cursor (delete key)
func (e *SQLEditor) DeleteCharAfter() {
	line := e.lines[e.cursorRow]
	if e.cursorCol < len(line) {
		// Delete character at cursor
		e.lines[e.cursorRow] = line[:e.cursorCol] + line[e.cursorCol+1:]
	} else if e.cursorRow < len(e.lines)-1 {
		// Merge with next line
		nextLine := e.lines[e.cursorRow+1]
		e.lines[e.cursorRow] = line + nextLine
		// Remove next line
		e.lines = append(e.lines[:e.cursorRow+1], e.lines[e.cursorRow+2:]...)
	}
}

// SQL keywords for syntax highlighting
var sqlKeywords = map[string]bool{
	"SELECT": true, "FROM": true, "WHERE": true, "AND": true, "OR": true,
	"INSERT": true, "INTO": true, "VALUES": true, "UPDATE": true, "SET": true,
	"DELETE": true, "CREATE": true, "TABLE": true, "DROP": true, "ALTER": true,
	"INDEX": true, "VIEW": true, "JOIN": true, "LEFT": true, "RIGHT": true,
	"INNER": true, "OUTER": true, "FULL": true, "ON": true, "AS": true,
	"ORDER": true, "BY": true, "GROUP": true, "HAVING": true, "LIMIT": true,
	"OFFSET": true, "UNION": true, "ALL": true, "DISTINCT": true, "CASE": true,
	"WHEN": true, "THEN": true, "ELSE": true, "END": true, "NULL": true,
	"NOT": true, "IN": true, "EXISTS": true, "BETWEEN": true, "LIKE": true,
	"IS": true, "TRUE": true, "FALSE": true, "ASC": true, "DESC": true,
	"PRIMARY": true, "KEY": true, "FOREIGN": true, "REFERENCES": true,
	"CONSTRAINT": true, "UNIQUE": true, "CHECK": true, "DEFAULT": true,
	"CASCADE": true, "NULLS": true, "FIRST": true, "LAST": true,
	"BEGIN": true, "COMMIT": true, "ROLLBACK": true, "TRANSACTION": true,
	"WITH": true, "RECURSIVE": true, "RETURNING": true, "COALESCE": true,
	"CAST": true, "COUNT": true, "SUM": true, "AVG": true, "MIN": true, "MAX": true,
}

// TokenType represents the type of a syntax token
type TokenType int

const (
	TokenText TokenType = iota
	TokenKeyword
	TokenString
	TokenNumber
	TokenComment
	TokenOperator
)

// Token represents a syntax-highlighted token
type Token struct {
	Type  TokenType
	Value string
}

// tokenizeLine tokenizes a single line for syntax highlighting
func (e *SQLEditor) tokenizeLine(line string) []Token {
	var tokens []Token
	i := 0

	for i < len(line) {
		// Skip whitespace
		if unicode.IsSpace(rune(line[i])) {
			start := i
			for i < len(line) && unicode.IsSpace(rune(line[i])) {
				i++
			}
			tokens = append(tokens, Token{Type: TokenText, Value: line[start:i]})
			continue
		}

		// Comment (-- to end of line)
		if i+1 < len(line) && line[i:i+2] == "--" {
			tokens = append(tokens, Token{Type: TokenComment, Value: line[i:]})
			break
		}

		// String literal (single quotes)
		if line[i] == '\'' {
			start := i
			i++
			for i < len(line) {
				if line[i] == '\'' {
					if i+1 < len(line) && line[i+1] == '\'' {
						// Escaped quote
						i += 2
					} else {
						i++
						break
					}
				} else {
					i++
				}
			}
			tokens = append(tokens, Token{Type: TokenString, Value: line[start:i]})
			continue
		}

		// Number
		if unicode.IsDigit(rune(line[i])) || (line[i] == '.' && i+1 < len(line) && unicode.IsDigit(rune(line[i+1]))) {
			start := i
			for i < len(line) && (unicode.IsDigit(rune(line[i])) || line[i] == '.') {
				i++
			}
			tokens = append(tokens, Token{Type: TokenNumber, Value: line[start:i]})
			continue
		}

		// Identifier or keyword
		if unicode.IsLetter(rune(line[i])) || line[i] == '_' {
			start := i
			for i < len(line) && (unicode.IsLetter(rune(line[i])) || unicode.IsDigit(rune(line[i])) || line[i] == '_') {
				i++
			}
			word := line[start:i]
			if sqlKeywords[strings.ToUpper(word)] {
				tokens = append(tokens, Token{Type: TokenKeyword, Value: word})
			} else {
				tokens = append(tokens, Token{Type: TokenText, Value: word})
			}
			continue
		}

		// Operators
		if strings.ContainsRune("=<>!+-*/%&|^~", rune(line[i])) {
			start := i
			for i < len(line) && strings.ContainsRune("=<>!+-*/%&|^~", rune(line[i])) {
				i++
			}
			tokens = append(tokens, Token{Type: TokenOperator, Value: line[start:i]})
			continue
		}

		// Other single characters (parens, commas, etc.)
		tokens = append(tokens, Token{Type: TokenText, Value: string(line[i])})
		i++
	}

	return tokens
}

// renderTokens renders tokens with syntax highlighting
func (e *SQLEditor) renderTokens(tokens []Token) string {
	var result strings.Builder

	for _, token := range tokens {
		var style lipgloss.Style
		switch token.Type {
		case TokenKeyword:
			style = lipgloss.NewStyle().Foreground(e.Theme.Keyword).Bold(true)
		case TokenString:
			style = lipgloss.NewStyle().Foreground(e.Theme.String)
		case TokenNumber:
			style = lipgloss.NewStyle().Foreground(e.Theme.Number)
		case TokenComment:
			style = lipgloss.NewStyle().Foreground(e.Theme.Comment).Italic(true)
		case TokenOperator:
			style = lipgloss.NewStyle().Foreground(e.Theme.Operator)
		default:
			style = lipgloss.NewStyle().Foreground(e.Theme.Foreground)
		}
		result.WriteString(style.Render(token.Value))
	}

	return result.String()
}

// View renders the SQL editor
func (e *SQLEditor) View() string {
	// Calculate visible lines based on height
	contentHeight := e.Height - 2 // Account for borders
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Determine which lines to show
	var visibleLines []string
	var startLine int

	if e.expanded {
		// Show all lines that fit, scroll if needed
		if e.cursorRow >= contentHeight {
			startLine = e.cursorRow - contentHeight + 1
		}
		endLine := startLine + contentHeight
		if endLine > len(e.lines) {
			endLine = len(e.lines)
		}

		for i := startLine; i < endLine; i++ {
			visibleLines = append(visibleLines, e.renderLine(i, i == e.cursorRow))
		}

		// Pad with empty lines if needed
		for len(visibleLines) < contentHeight {
			visibleLines = append(visibleLines, e.renderEmptyLine(len(e.lines)+len(visibleLines)-len(e.lines)+startLine))
		}
	} else {
		// Collapsed: show first 2 lines
		for i := 0; i < 2 && i < len(e.lines); i++ {
			visibleLines = append(visibleLines, e.renderLine(i, false))
		}
		// Pad if less than 2 lines
		for len(visibleLines) < 2 {
			visibleLines = append(visibleLines, e.renderEmptyLine(len(visibleLines)))
		}
	}

	content := strings.Join(visibleLines, "\n")

	// Container style
	borderColor := e.Theme.Border
	if e.expanded {
		borderColor = e.Theme.BorderFocused
	}

	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(e.Width - 2). // Account for border
		Height(contentHeight)

	return containerStyle.Render(content)
}

// renderLine renders a single line with line number and syntax highlighting
func (e *SQLEditor) renderLine(lineNum int, hasCursor bool) string {
	// Line number
	lineNumWidth := e.getLineNumberWidth()
	lineNumStr := fmt.Sprintf("%*d", lineNumWidth-3, lineNum+1)

	lineNumStyle := lipgloss.NewStyle().Foreground(e.Theme.Metadata)
	sepStyle := lipgloss.NewStyle().Foreground(e.Theme.Border)

	lineNumPart := lineNumStyle.Render(lineNumStr) + sepStyle.Render(" │ ")

	// Line content with syntax highlighting
	line := e.lines[lineNum]
	tokens := e.tokenizeLine(line)
	contentPart := e.renderTokens(tokens)

	// Insert cursor if this line has it
	if hasCursor && e.expanded {
		contentPart = e.insertCursor(line, tokens)
	}

	return lineNumPart + contentPart
}

// renderEmptyLine renders an empty line placeholder
func (e *SQLEditor) renderEmptyLine(lineNum int) string {
	lineNumWidth := e.getLineNumberWidth()
	lineNumStr := fmt.Sprintf("%*s", lineNumWidth-3, "~")

	lineNumStyle := lipgloss.NewStyle().Foreground(e.Theme.Metadata)
	sepStyle := lipgloss.NewStyle().Foreground(e.Theme.Border)

	return lineNumStyle.Render(lineNumStr) + sepStyle.Render(" │ ")
}

// getLineNumberWidth returns the width needed for line numbers
func (e *SQLEditor) getLineNumberWidth() int {
	maxLine := len(e.lines)
	if maxLine < 10 {
		maxLine = 10
	}
	digits := len(fmt.Sprintf("%d", maxLine))
	if digits < 2 {
		digits = 2
	}
	return digits + 3 // digits + space + separator
}

// insertCursor inserts the cursor character into the rendered line
func (e *SQLEditor) insertCursor(line string, tokens []Token) string {
	// Rebuild line with cursor
	var result strings.Builder
	charIdx := 0

	cursorStyle := lipgloss.NewStyle().
		Foreground(e.Theme.Background).
		Background(e.Theme.Cursor)

	for _, token := range tokens {
		var style lipgloss.Style
		switch token.Type {
		case TokenKeyword:
			style = lipgloss.NewStyle().Foreground(e.Theme.Keyword).Bold(true)
		case TokenString:
			style = lipgloss.NewStyle().Foreground(e.Theme.String)
		case TokenNumber:
			style = lipgloss.NewStyle().Foreground(e.Theme.Number)
		case TokenComment:
			style = lipgloss.NewStyle().Foreground(e.Theme.Comment).Italic(true)
		case TokenOperator:
			style = lipgloss.NewStyle().Foreground(e.Theme.Operator)
		default:
			style = lipgloss.NewStyle().Foreground(e.Theme.Foreground)
		}

		for _, ch := range token.Value {
			if charIdx == e.cursorCol {
				result.WriteString(cursorStyle.Render(string(ch)))
			} else {
				result.WriteString(style.Render(string(ch)))
			}
			charIdx++
		}
	}

	// Cursor at end of line
	if e.cursorCol >= charIdx {
		result.WriteString(cursorStyle.Render(" "))
	}

	return result.String()
}
