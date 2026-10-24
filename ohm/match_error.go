package ohm

import (
	"fmt"
	"strconv"
	"strings"
)

// MatchError describes a failed match.
// Its text follows message and shortMessage in miniohm.
// See error-reporting prd section 6.
type MatchError struct {
	Input    string
	Pos      Position
	Expected []Expected // nil when the next valid tokens are not known
}

// Error returns the short message.
func (e *MatchError) Error() string {
	return e.ShortMessage()
}

// ShortMessage returns one line, such as
// `Line 1, col 5: expected "(", an identifier, or a number`.
// It reads `Line 1, col 5: match failed` when Expected is nil.
func (e *MatchError) ShortMessage() string {
	prefix := fmt.Sprintf("Line %d, col %d: ", e.Pos.Line, e.Pos.Column)
	if e.Expected == nil {
		return prefix + "match failed"
	}
	return prefix + "expected " + expectedText(e.Expected)
}

// Message returns an excerpt of the input with a caret under the failure,
// then `Expected ...` when Expected is not nil. The excerpt shows the line
// before and the line after, when they exist, as getLineAndColumnMessage
// in ohm-js does.
func (e *MatchError) Message() string {
	excerpt := lineAndColumnMessage(e.Input, e.Pos)
	if e.Expected == nil {
		return strings.TrimSuffix(excerpt, "\n")
	}
	return excerpt + "Expected " + expectedText(e.Expected)
}

// expectedText joins the texts as miniohm getExpectedText does:
// "a", "a or b", "a, b, or c".
func expectedText(expected []Expected) string {
	var sb strings.Builder
	for i, ex := range expected {
		if i > 0 {
			if i == len(expected)-1 {
				if len(expected) > 2 {
					sb.WriteString(", or ")
				} else {
					sb.WriteString(" or ")
				}
			} else {
				sb.WriteString(", ")
			}
		}
		sb.WriteString(ex.Text)
	}
	return sb.String()
}

// lineAndColumnMessage mirrors getLineAndColumnMessage in ohm-js util.js,
// without ranges. Each line of the result ends in "\n".
func lineAndColumnMessage(input string, pos Position) string {
	// Guard against a Position that positionAt did not make.
	byteOffset := min(max(pos.ByteOffset, 0), len(input))
	column := max(pos.Column, 1)
	// Find the failure line, and the lines around it.
	lineStart := strings.LastIndexByte(input[:byteOffset], '\n') + 1
	line, rest, hasNext := strings.Cut(input[lineStart:], "\n")
	var prevLine, nextLine string
	hasPrev := lineStart > 0
	if hasPrev {
		before := input[:lineStart-1]
		prevLine = before[strings.LastIndexByte(before, '\n')+1:]
	}
	if hasNext {
		nextLine, _, _ = strings.Cut(rest, "\n")
	}
	// Right-align the line numbers to the widest number shown.
	width := len(strconv.Itoa(pos.Line))
	if hasNext {
		width = len(strconv.Itoa(pos.Line + 1))
	}
	var sb strings.Builder
	appendLine := func(prefix string, num int, content string) {
		fmt.Fprintf(&sb, "%s%*d | %s\n", prefix, width, num, strings.TrimSuffix(content, "\r"))
	}
	fmt.Fprintf(&sb, "Line %d, col %d:\n", pos.Line, pos.Column)
	if hasPrev {
		appendLine("  ", pos.Line-1, prevLine)
	}
	appendLine("> ", pos.Line, line)
	// The gutter is "> ", the line number and " | ".
	sb.WriteString(strings.Repeat(" ", 2+width+3+column-1))
	sb.WriteString("^\n")
	if hasNext {
		appendLine("  ", pos.Line+1, nextLine)
	}
	return sb.String()
}
