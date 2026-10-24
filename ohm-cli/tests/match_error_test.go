package tests

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	goohm "github.com/ohmjs/ohm-go/ohm"
	"github.com/ohmjs/ohm-go/ohm-cli/sexpr"
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

// TestMatchError_FailuresNotMostRecent checks that RightmostFailures on a
// result that is no longer the most recent returns the most-recent error,
// as miniohm _assertMostRecent does. Err then gives the position only.
func TestMatchError_FailuresNotMostRecent(t *testing.T) {
	gmr := newOhmGrammar(t)
	first, err := gmr.Match("G {\n  S = *x\n}")
	if err != nil {
		t.Fatalf("matching: %v", err)
	}
	second, err := gmr.Match("G { S = \"a\" }\nH")
	if err != nil {
		t.Fatalf("matching: %v", err)
	}
	_, err = first.RightmostFailures()
	if err == nil || !strings.Contains(err.Error(), "not the most recent match") {
		t.Errorf("RightmostFailures after a second Match: got error %v, want the most-recent error", err)
	}
	if got, want := first.Err().Error(), "Line 2, col 7: match failed"; got != want {
		t.Errorf("Err after a second Match: got %q, want %q", got, want)
	}
	// The second result is the most recent, so it still gets its failures.
	if got, err := second.ExpectedText(); err != nil || got != `"{"` {
		t.Errorf("ExpectedText of the second result: got %q, %v, want %q", got, err, `"{"`)
	}
	second.Close()
	first.Close()
}

// TestMatchError_ExpectedAfterClose checks that Expected after Close
// returns the cache, and returns the closed error when there is no cache.
func TestMatchError_ExpectedAfterClose(t *testing.T) {
	gmr := newOhmGrammar(t)
	mr, err := gmr.Match("G {\n  S = *x\n}")
	if err != nil {
		t.Fatalf("matching: %v", err)
	}
	before, err := mr.Expected()
	if err != nil {
		t.Fatalf("Expected before Close: %v", err)
	}
	mr.Close()
	after, err := mr.Expected()
	if err != nil {
		t.Fatalf("Expected after Close: %v", err)
	}
	want := []goohm.Expected{{Kind: goohm.ExpectedLiteral, Text: `"}"`, Literal: "}"}}
	if !slices.Equal(before, want) || !slices.Equal(after, want) {
		t.Errorf("Expected: got %+v before Close and %+v after, want %+v", before, after, want)
	}
	// With no call before Close there is no cache, so Expected fails.
	uncached, err := gmr.Match("G {\n  S = *x\n}")
	if err != nil {
		t.Fatalf("matching: %v", err)
	}
	uncached.Close()
	if _, err := uncached.Expected(); err == nil || !strings.Contains(err.Error(), "closed") {
		t.Errorf("Expected after Close with no cache: got error %v, want the closed error", err)
	}
}

// TestMatchError_CloseRestoresHeap checks that Close after
// RightmostFailures resets the Wasm bump pointer to the watermark from
// before the match, so it frees both parses.
func TestMatchError_CloseRestoresHeap(t *testing.T) {
	gmr := newOhmGrammar(t)
	offset := func() uint64 {
		return gmr.GetModule().ExportedGlobal("__offset").Get()
	}
	watermark := offset()
	mr, err := gmr.Match("G {\n  S = *x\n}")
	if err != nil {
		t.Fatalf("matching: %v", err)
	}
	afterMatch := offset()
	if _, err := mr.RightmostFailures(); err != nil {
		t.Fatalf("RightmostFailures: %v", err)
	}
	afterRecord := offset()
	// The second parse allocates above the first, so Close must free both.
	if afterRecord <= afterMatch {
		t.Errorf("__offset after RightmostFailures: got %d, want above %d, the value after Match", afterRecord, afterMatch)
	}
	mr.Close()
	if got := offset(); got != watermark {
		t.Errorf("__offset after Close: got %d, want the watermark %d", got, watermark)
	}
}

// TestMatchError_CommandFormat checks the error of a command on input that
// does not parse: `name:line:col: expected …`, then the excerpt, with the
// position and the expected text each given once (error-reporting prd
// section 9). An input with no name reads as utils.NoInputName.
func TestMatchError_CommandFormat(t *testing.T) {
	for _, tc := range []struct {
		inputName, want string
	}{
		{"g.ohm", "g.ohm:2:7: expected \"}\"\n" +
			"  1 | G {\n" +
			"> 2 |   S = *x\n" +
			"            ^\n" +
			"  3 | }"},
		{"", "<input>:2:7: expected \"}\"\n" +
			"  1 | G {\n" +
			"> 2 |   S = *x\n" +
			"            ^\n" +
			"  3 | }"},
	} {
		cmd := &sexpr.ToSexprCmd{Grammar: "G {\n  S = *x\n}", InputName: tc.inputName}
		_, err := cmd.Process()
		if err == nil {
			t.Fatalf("Process: got no error, want a match error")
		}
		if got := err.Error(); got != tc.want {
			t.Errorf("Process error:\ngot\n%s\nwant\n%s", got, tc.want)
		}
		if merr := (*goohm.MatchError)(nil); !errors.As(err, &merr) {
			t.Errorf("Process error: got %T, want one that wraps *ohm.MatchError", err)
		}
	}
}

// TestMatchError_CommandFormatNoExpected checks the error when the next
// valid tokens are not known: `name:line:col: match failed`, then the
// excerpt with no `Expected` line.
func TestMatchError_CommandFormatNoExpected(t *testing.T) {
	gmr := newOhmGrammar(t)
	first, err := gmr.Match("G {\n  S = *x\n}")
	if err != nil {
		t.Fatalf("matching: %v", err)
	}
	defer first.Close()
	second, err := gmr.Match("G { S = \"a\" }\nH")
	if err != nil {
		t.Fatalf("matching: %v", err)
	}
	defer second.Close()
	want := "g.ohm:2:7: match failed\n" +
		"  1 | G {\n" +
		"> 2 |   S = *x\n" +
		"            ^\n" +
		"  3 | }"
	if got := utils.NewMatchError("g.ohm", first).Error(); got != want {
		t.Errorf("NewMatchError after a second Match:\ngot\n%s\nwant\n%s", got, want)
	}
}
