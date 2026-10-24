package utils

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ohmjs/ohm-go/ohm"
)

// NoInputName names an input that was given as text, not as a path.
const NoInputName = "<input>"

// MatchError is the error of a failed match, named for its input. Its text
// is `name:line:col: expected …`, then the excerpt of the input with a
// caret on the next lines. See error-reporting prd section 9.
type MatchError struct {
	Name string // the input path, or `archive:member` for a txtar input
	Err  *ohm.MatchError
}

// NewMatchError answers the error of the failed match mr, named name. An
// empty name reads as NoInputName. It answers nil when mr succeeded. Call
// it before the next Match on the grammar, and before mr.Close, so the
// error holds the next valid tokens.
func NewMatchError(name string, mr *ohm.MatchResult) error {
	var merr *ohm.MatchError
	if !errors.As(mr.Err(), &merr) {
		return nil
	}
	if name == "" {
		name = NoInputName
	}
	return &MatchError{Name: name, Err: merr}
}

// Error answers `name:line:col: expected …`, then the excerpt. The
// position is in the prefix only: the `Line L, col C` text of
// ShortMessage and Message is cut, and so is the `Expected …` line that
// ends Message.
func (e *MatchError) Error() string {
	pos := e.Err.Pos
	lineCol := fmt.Sprintf("Line %d, col %d:", pos.Line, pos.Column)
	detail := strings.TrimPrefix(e.Err.ShortMessage(), lineCol+" ")
	excerpt := strings.TrimPrefix(e.Err.Message(), lineCol+"\n")
	if text, ok := strings.CutPrefix(detail, "expected "); ok && e.Err.Expected != nil {
		excerpt = strings.TrimSuffix(excerpt, "Expected "+text)
	}
	return fmt.Sprintf("%s:%d:%d: %s\n%s", e.Name, pos.Line, pos.Column, detail,
		strings.TrimSuffix(excerpt, "\n"))
}

// Unwrap answers the *ohm.MatchError.
func (e *MatchError) Unwrap() error { return e.Err }
