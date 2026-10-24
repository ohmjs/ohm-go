// Package matcherr prints the error of a failed match.
// It backs the `.error` txtar kind. See error-reporting prd section 10.5.
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
	Grammar string `help:"Path to the .ohm grammar file, compiled with docker. Defaults to the embedded Ohm grammar."`
	Start   string `opts:"short=r" help:"The start rule. Defaults to the grammar's default start rule."`
	Short   bool   `opts:"short=s" help:"Print the one-line short message."`
	Source  string `opts:"mode=arg" help:"Input text. If it starts with an @ it is a path to the input file."`
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
// `match succeeded` when the input matches.
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
	var merr *ohm.MatchError
	if !errors.As(mr.Err(), &merr) {
		return "match succeeded", nil
	}
	if cmd.Short {
		return merr.ShortMessage(), nil
	}
	return merr.Message(), nil
}
