package tests

import (
	"context"
	"testing"

	goohm "github.com/ohmjs/ohm-go/ohm"
	"github.com/ohmjs/ohm-go/ohm-cli/utils"
)

// These tests state the rules of error-reporting prd section 10.6 that a
// txtar fixture cannot. They match against the embedded Ohm grammar, so
// they need no docker.

func newOhmGrammar(t *testing.T) *goohm.Grammar {
	t.Helper()
	gmr, err := goohm.NewGrammar(context.Background(), utils.OhmGrammarWasmBytes())
	if err != nil {
		t.Fatalf("creating grammar: %v", err)
	}
	t.Cleanup(func() { gmr.Close() })
	return gmr
}

// failurePosition matches input and answers the failure position. It
// fails the test when the input matches.
func failurePosition(t *testing.T, gmr *goohm.Grammar, input string, startRule ...string) goohm.Position {
	t.Helper()
	mr, err := gmr.Match(input, startRule...)
	if err != nil {
		t.Fatalf("matching %q: %v", input, err)
	}
	defer mr.Close()
	if mr.Succeeded() {
		t.Fatalf("matching %q: succeeded, want a failure", input)
	}
	return mr.FailurePosition()
}

// TestMatchError_CRLF checks that `\r\n` and `\n` input give the same line
// and column. A txtar member can lose its `\r` to git settings, so this
// case is Go code.
func TestMatchError_CRLF(t *testing.T) {
	gmr := newOhmGrammar(t)
	lf := failurePosition(t, gmr, "G {\n  S = *x\n}")
	crlf := failurePosition(t, gmr, "G {\r\n  S = *x\r\n}")
	if lf.Line != 2 || lf.Column != 7 {
		t.Errorf("\\n input: got line %d, col %d, want line 2, col 7", lf.Line, lf.Column)
	}
	if crlf.Line != lf.Line || crlf.Column != lf.Column {
		t.Errorf("\\r\\n input: got line %d, col %d, want line %d, col %d",
			crlf.Line, crlf.Column, lf.Line, lf.Column)
	}
}

// TestMatchError_EmptyInput checks that empty input gives line 1, column 1.
// The start rule is Grammar, since empty input matches Grammars.
func TestMatchError_EmptyInput(t *testing.T) {
	gmr := newOhmGrammar(t)
	pos := failurePosition(t, gmr, "", "Grammar")
	want := goohm.Position{Offset: 0, ByteOffset: 0, Line: 1, Column: 1}
	if pos != want {
		t.Errorf("got %+v, want %+v", pos, want)
	}
}

// TestMatchError_PositionAfterClose checks that FailurePosition and Err
// read only Go memory, so they give the same value after Close.
func TestMatchError_PositionAfterClose(t *testing.T) {
	gmr := newOhmGrammar(t)
	mr, err := gmr.Match("G {\n  S = *x\n}")
	if err != nil {
		t.Fatalf("matching: %v", err)
	}
	before := mr.FailurePosition()
	beforeErr := mr.Err()
	mr.Close()
	if after := mr.FailurePosition(); after != before {
		t.Errorf("FailurePosition after Close: got %+v, want %+v", after, before)
	}
	afterErr := mr.Err()
	if beforeErr == nil || afterErr == nil || afterErr.Error() != beforeErr.Error() {
		t.Errorf("Err after Close: got %v, want %v", afterErr, beforeErr)
	}
	// A second match overwrites the Wasm global; the closed result keeps
	// its own copy.
	other, err := gmr.Match("G { S = \"a\" }\nH")
	if err != nil {
		t.Fatalf("matching: %v", err)
	}
	defer other.Close()
	if after := mr.FailurePosition(); after != before {
		t.Errorf("FailurePosition after a second Match: got %+v, want %+v", after, before)
	}
}
