package matcherr

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf16"

	"github.com/ohmjs/ohm-go/ohm"
)

// NextTokensCmd tests the next valid tokens of a failed match. For each
// token it matches the input up to the failure, with the token after it,
// and reports whether the grammar took the token. A token the runtime
// records but the grammar then rejects is a bug in the failure recording,
// or a token that only a longer input can take.
// See error-reporting prd section 10.9.
type NextTokensCmd struct {
	Grammar string `help:"Path to the .ohm grammar file, compiled with docker. Defaults to the embedded Ohm grammar."`
	Start   string `opts:"short=r" help:"The start rule. Defaults to the grammar's default start rule."`
	Fluffy  bool   `opts:"short=f" help:"Also test the fluffy failures, each marked ' (fluffy)'."`
	Source  string `opts:"mode=arg" help:"Input text. If it starts with an @ it is a path to the input file."`
}

func NewNextTokensCmd() *NextTokensCmd {
	return &NextTokensCmd{}
}

func (cmd *NextTokensCmd) Run() error {
	var err error
	if cmd.Source, err = readSource(cmd.Source); err != nil {
		return err
	}
	out, err := cmd.Process()
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", out)
	return nil
}

// Process matches Source and answers one line per next valid token, as
// `<kind> <text>: <verdict>`, in the order of first recording, or the one
// line `match succeeded` when the input matches. The verdict is one of:
//
//   - `accepted`: the prefix with the token after it fails past the token;
//   - `match succeeded`: the prefix with the token after it matches;
//   - `rejected at L:C`: the prefix with the token after it fails inside
//     the token, or, for end of input, the prefix alone fails;
//   - `untested`: a description, which names no text to try.
//
// A range is tested with its first character. Under Fluffy the fluffy
// failures are tested too, each line ending in ` (fluffy)`.
func (cmd *NextTokensCmd) Process() (string, error) {
	gmr, err := openGrammar(cmd.Grammar)
	if err != nil {
		return "", err
	}
	defer gmr.Close()
	mr, err := gmr.Match(cmd.Source, cmd.Start)
	if err != nil {
		return "", fmt.Errorf("matching: %v", err)
	}
	if mr.Succeeded() {
		mr.Close()
		return "match succeeded", nil
	}
	failures, err := mr.RightmostFailures()
	if err != nil {
		mr.Close()
		return "", err
	}
	pos := mr.FailurePosition()
	// The failures are cached on mr, so the probes can match next.
	mr.Close()
	prefix := cmd.Source[:pos.ByteOffset]
	var sb strings.Builder
	for _, f := range failures {
		if f.Fluffy && !cmd.Fluffy {
			continue
		}
		verdict, err := cmd.probe(gmr, prefix, pos, f.Expected)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&sb, "%s %s: %s", f.Kind, f.Text, verdict)
		if f.Fluffy {
			sb.WriteString(" (fluffy)")
		}
		sb.WriteString("\n")
	}
	return strings.TrimSuffix(sb.String(), "\n"), nil
}

// probe matches prefix with the text of ex after it, and answers the
// verdict. pos is the failure position of the original match, the end
// of prefix.
func (cmd *NextTokensCmd) probe(gmr *ohm.Grammar, prefix string, pos ohm.Position, ex ohm.Expected) (string, error) {
	text, ok := probeText(ex)
	if !ok {
		return "untested", nil
	}
	mr, err := gmr.Match(prefix+text, cmd.Start)
	if err != nil {
		return "", fmt.Errorf("matching %q: %v", prefix+text, err)
	}
	defer mr.Close()
	if mr.Succeeded() {
		return "match succeeded", nil
	}
	// The token is taken when no expression failed inside it, so the
	// rightmost failure is at or past its end. End of input has no text,
	// so a failure of the prefix alone always rejects it.
	tokenEnd := pos.Offset + len(utf16.Encode([]rune(text)))
	if ex.Kind != ohm.ExpectedEnd && mr.RightmostFailurePosition() >= tokenEnd {
		return "accepted", nil
	}
	at := mr.FailurePosition()
	return fmt.Sprintf("rejected at %d:%d", at.Line, at.Column), nil
}

// probeText answers the text to try for ex: the literal, the first
// character of a range, or nothing for end of input. A description names
// no text, so ok is false.
func probeText(ex ohm.Expected) (text string, ok bool) {
	switch ex.Kind {
	case ohm.ExpectedLiteral:
		return ex.Literal, true
	case ohm.ExpectedEnd:
		return "", true
	case ohm.ExpectedRange:
		// Text is `"lo".."hi"`; both sides are JSON strings.
		lo, _, found := strings.Cut(ex.Text, "..")
		if !found {
			return "", false
		}
		var s string
		if err := json.Unmarshal([]byte(lo), &s); err != nil {
			return "", false
		}
		return s, true
	}
	return "", false
}
