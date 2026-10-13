package ruleast

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/ohmjs/ohm-go/ohm"
	"github.com/ohmjs/ohm-go/ohm-cli/utils"
)

func BenchmarkRuleAstCmd_Process(b *testing.B) {
	barr, err := os.ReadFile("../../ohm-repo/packages/ohm-js/src/ohm-grammar.ohm")
	if err != nil {
		b.Fatalf("Error reading grammar file. %v", err)
	}
	vc := NewRuleAstCmd()
	vc.Grammar = string(barr)
	// vc.GrammarName = "ES5"
	// vc.HandCodedWalk = true

	vc.sbldr = &strings.Builder{}
	ctx := context.Background()
	var (
		gmr  *ohm.Grammar
		mr   *ohm.MatchResult
		root ohm.Node
		gAst *GrammarsNode
	)
	if gmr, err = ohm.NewGrammar(ctx, utils.OhmGrammarWasmBytes()); err != nil {
		b.Fatalf("creating grammar: %v", err)
	}
	defer gmr.Close()
	if mr, err = gmr.Match(vc.Grammar); err != nil {
		b.Fatalf("matching: %v", err)
	}
	defer mr.Close()
	if !mr.Succeeded() {
		b.Fatalf("match failed")
	}
	if root, err = mr.GetCstRoot(); err != nil {
		b.Fatalf("Error getting cst root. %v", err)
	}
	for b.Loop() {
		if vc.HandCodedWalk {
			gAst, err = BuildGrammarsHandCodedWalk(root).BuildRuleAst(root)
		} else {
			gmrs := MakeNodeFromRoot[any, *GrammarsNode](root)
			v := &ruleAstBuilder{}
			gAst, err = gmrs.Accept(root, v, nil)
		}
		_ = gAst
		if err != nil {
			b.Fatalf("%v", err)
		}
	}
}
