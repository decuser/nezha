package position

import (
	"fmt"
	"strings"
)

// RenderDiagnostic renders an error location with source line and caret.
//
// Output:
//
//	file.nez:4:10: message
//	    let x = 1 / 0
//	             ^
func RenderDiagnostic(pos Position, source string, message string) string {
	var sb strings.Builder

	if pos.File != "" {
		sb.WriteString(fmt.Sprintf("%s:%d:%d: %s", pos.File, pos.Line, pos.Col, message))
	} else {
		sb.WriteString(fmt.Sprintf("%d:%d: %s", pos.Line, pos.Col, message))
	}

	if source != "" {
		sb.WriteString("\n    ")
		sb.WriteString(strings.TrimRight(source, "\r\n"))
		sb.WriteString("\n    ")
		sb.WriteString(CaretLine(pos.Col))
	}

	return sb.String()
}

// SourceLine extracts a specific line from source text.
// Line is 1-indexed. Returns empty string if out of range.
func SourceLine(source string, line int) string {
	if line < 1 {
		return ""
	}
	lines := strings.Split(source, "\n")
	if line > len(lines) {
		return ""
	}
	return lines[line-1]
}

// CaretLine returns a caret string positioned at the given column.
func CaretLine(col int) string {
	if col <= 0 {
		return "^"
	}
	return strings.Repeat(" ", col-1) + "^"
}
