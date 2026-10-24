package ohm

import "unicode/utf8"

// Position is a place in the input of a MatchResult.
// See error-reporting prd sections 4 and 5.
type Position struct {
	Offset     int // UTF-16 code units from the start, as the Wasm reports
	ByteOffset int // bytes from the start of the Go string
	Line       int // 1-based
	Column     int // 1-based, in code points; a `\r` is not counted
}

// positionAt converts a UTF-16 offset into input to a Position.
// It mirrors getLineAndColumn in ohm-js extras.ts, but counts columns in
// code points, not UTF-16 code units:
//   - lines are split at `\n`; a `\r` is not counted as a column;
//   - a character outside the BMP is one column, and a tab is one column;
//   - an offset at the end of input points one column past the last character;
//   - an offset inside a surrogate pair rounds down to the start of the pair.
//
// An offset below 0 is treated as 0; one past the end is treated as the end.
// Offset in the result is the (clamped, rounded) offset that was used.
func positionAt(input string, utf16Offset int) Position {
	p := Position{Line: 1, Column: 1}
	units := 0
	for i, r := range input {
		// Matches utf16.Encode([]rune(input)): an invalid byte is U+FFFD, one unit.
		width := 1
		if r >= 0x10000 && r <= utf8.MaxRune {
			width = 2
		}
		if units+width > utf16Offset {
			p.Offset = units
			p.ByteOffset = i
			return p
		}
		units += width
		switch r {
		case '\n':
			p.Line++
			p.Column = 1
		case '\r':
		default:
			p.Column++
		}
	}
	p.Offset = units
	p.ByteOffset = len(input)
	return p
}
