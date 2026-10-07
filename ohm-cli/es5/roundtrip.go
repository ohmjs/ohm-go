package es5

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/ohmjs/ohm-go/ohm"
)

type es5roundtrip struct {
	Source string `opts:"mode=arg"`
	sb     *strings.Builder
}

//go:generate docker run --rm -v $PWD/../../ohm-repo/examples/ecmascript/src:/local -v $PWD:/dst ohmjs/ohm:latest compile es5.ohm

var (
	//go:embed es5.wasm
	es5GrammarWasmBytes []byte
)

func NewEs5Roundtrip() *es5roundtrip {
	return &es5roundtrip{
		sb: &strings.Builder{},
	}
}

func (cm *es5roundtrip) Run() error {
	ctx := context.Background()
	var (
		gmr  *ohm.Grammar
		mr   *ohm.MatchResult
		err  error
		root ohm.Node
	)
	if len(cm.Source) > 1 && cm.Source[:1] == "@" {
		barr, err := os.ReadFile(cm.Source[1:])
		if err != nil {
			return fmt.Errorf("Error reading grammar file. %[1]v", err)
		}
		cm.Source = string(barr)
	}
	if gmr, err = ohm.NewGrammar(ctx, es5GrammarWasmBytes); err != nil {
		return fmt.Errorf("creating grammar: %[1]v", err)
	}
	defer gmr.Close()
	if mr, err = gmr.Match(cm.Source); err != nil {
		return fmt.Errorf("matching: %[1]v", err)
	}
	defer mr.Close()
	if !mr.Succeeded() {
		return fmt.Errorf("match failed")
	}
	if root, err = mr.GetCstRoot(); err != nil {
		return fmt.Errorf("Error getting cst root. %[1]v", err)
	}
	program := MakeNodeFromRoot[any, any](root)
	// program := Program[any, any]{
	// 	SourceElement: root.Children()[0].(goohm.ListNode),
	// }
	program.Accept(root, cm, nil)
	fmt.Printf("%s", cm.sb.String())
	return nil
}

var (
	_ ohm.TerminalVisitor    = &es5roundtrip{}
	_ ohm.NodeVisitor        = &es5roundtrip{}
	_ ohm.SpaceBeforeVisitor = &es5roundtrip{}
)

// SpaceBefore implements [goohm.SpaceBeforeVisitor].
func (cm *es5roundtrip) SpaceBefore(spaces string) {
	fmt.Printf("'%s'\n", spaces)
	cm.sb.WriteString(spaces)
}

// Terminal implements [goohm.TerminalVisitor].
func (cm *es5roundtrip) Terminal(node ohm.TerminalNode) {
	cm.sb.WriteString(node.SourceString())
}

// NodeVisit implements [goohm.NodeVisitor].
func (cm *es5roundtrip) NodeVisit(node ohm.Node) {
	cm.sb.WriteString(node.SourceString())
	// cm.sb.WriteString(node.CtorName())
	// cm.sb.WriteString("\n")
}
