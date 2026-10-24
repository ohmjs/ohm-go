// Package matcherr prints the error of a failed match, or its next valid
// tokens. It backs the `.error` and `.expected` txtar kinds.
// See error-reporting prd section 10.5.
package matcherr

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ohmjs/ohm-go/ohm"
	"github.com/ohmjs/ohm-go/ohm-cli/candp"
	"github.com/ohmjs/ohm-go/ohm-cli/utils"
)

type MatchErrorCmd struct {
	Grammar  string `help:"Path to the .ohm grammar file, compiled with docker. Defaults to the embedded Ohm grammar."`
	Start    string `opts:"short=r" help:"The start rule. Defaults to the grammar's default start rule."`
	Short    bool   `opts:"short=s" help:"Print the one-line short message."`
	Expected bool   `opts:"short=e" help:"Print the next valid tokens, one '<kind> <text>' line each, not the message."`
	Fluffy   bool   `opts:"short=f" help:"With --expected, also print the fluffy failures, each marked ' (fluffy)'."`
	Source   string `opts:"mode=arg" help:"Input text. If it starts with an @ it is a path to the input file."`
}

func NewMatchErrorCmd() *MatchErrorCmd {
	return &MatchErrorCmd{}
}

func (cmd *MatchErrorCmd) Run() error {
	if strings.HasPrefix(cmd.Source, "@") {
		barr, err := os.ReadFile(cmd.Source[1:])
		if err != nil {
			return fmt.Errorf("reading source file: %v", err)
		}
		cmd.Source = string(barr)
	}
	out, err := cmd.Process()
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", out)
	return nil
}

// Process matches Source and answers the error message, or
// `match succeeded` when the input matches. Under Expected it answers the
// next valid tokens instead; see expectedLines.
func (cmd *MatchErrorCmd) Process() (string, error) {
	wasmBytes := utils.OhmGrammarWasmBytes()
	if cmd.Grammar != "" {
		var err error
		if wasmBytes, err = candp.Compile(cmd.Grammar, false); err != nil {
			return "", err
		}
	}
	gmr, err := ohm.NewGrammar(context.Background(), wasmBytes)
	if err != nil {
		return "", fmt.Errorf("creating grammar: %v", err)
	}
	defer gmr.Close()
	mr, err := gmr.Match(cmd.Source, cmd.Start)
	if err != nil {
		return "", fmt.Errorf("matching: %v", err)
	}
	defer mr.Close()
	if mr.Succeeded() {
		return "match succeeded", nil
	}
	if cmd.Expected {
		return cmd.expectedLines(mr)
	}
	var merr *ohm.MatchError
	if !errors.As(mr.Err(), &merr) {
		return "", fmt.Errorf("matching: no MatchError for a failed match")
	}
	if cmd.Short {
		return merr.ShortMessage(), nil
	}
	return merr.Message(), nil
}

// expectedLines answers one `<kind> <text>` line per next valid token, in
// the order of first recording. Under Fluffy it adds the fluffy failures,
// in the same order, each line ending in ` (fluffy)`.
// See error-reporting prd section 10.2.
func (cmd *MatchErrorCmd) expectedLines(mr *ohm.MatchResult) (string, error) {
	failures, err := mr.RightmostFailures()
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, f := range failures {
		if f.Fluffy && !cmd.Fluffy {
			continue
		}
		fmt.Fprintf(&sb, "%s %s", f.Kind, f.Text)
		if f.Fluffy {
			sb.WriteString(" (fluffy)")
		}
		sb.WriteString("\n")
	}
	return strings.TrimSuffix(sb.String(), "\n"), nil
}
