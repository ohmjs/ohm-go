// Package arith evaluates and pretty-prints arithmetic expressions. It is
// the worked example for the --generic-methods flag of `ohm-cli generate go`.
//
// The arith_visitor package was generated with
//
//	ohm-cli generate go --generic-methods -P arith_visitor --output-dir arith_visitor @arith.ohm
//
// so its node structs carry no type parameters. Accept, DefaultAccept and
// the Accept<Arg> helpers are generic methods (Go 1.27) that are instantiated
// at each call. A node value is therefore not tied to one payload and result
// type, which this package uses in two ways:
//
//   - [EvaluateAndPrint] builds one *arith_visitor.Exp and hands it to the
//     evaluator (P=Env, R=int) and then to the printer (P=any, R=string).
//   - evaluator.VisitMulExpDivide calls a child Accept with the printer's
//     type arguments from inside the evaluator, to show the offending divisor
//     in a division by zero error. With struct generics that needs an
//     unsafe.Pointer cast from *MulExpDivide[Env, int] to
//     *MulExpDivide[any, string].
package arith

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/ohmjs/ohm-go/ohm"
	"github.com/ohmjs/ohmgo-examples/generic-methods/arith_visitor"
)

//go:embed arith.wasm
var wasmBytes []byte

// Env maps identifiers to their values.
type Env map[string]int

// NewGrammar loads the compiled Arithmetic grammar. Close it when done.
func NewGrammar(ctx context.Context) (*ohm.Grammar, error) {
	return ohm.NewGrammar(ctx, wasmBytes)
}

// Evaluate parses src and evaluates it, looking identifiers up in env.
func Evaluate(gmr *ohm.Grammar, src string, env Env) (value int, err error) {
	err = withRoot(gmr, src, func(root ohm.Node) (err error) {
		value, err = arith_visitor.MakeNodeFromRoot[Env, int](root)(root, &evaluator{}, env)
		return
	})
	return
}

// Print parses src and returns it fully parenthesised, so the precedence
// the grammar assigned is visible.
func Print(gmr *ohm.Grammar, src string) (text string, err error) {
	err = withRoot(gmr, src, func(root ohm.Node) (err error) {
		text, err = arith_visitor.MakeNodeFromRoot[any, string](root)(root, &printer{}, nil)
		return
	})
	return
}

// EvaluateAndPrint parses src once and runs both visitors over the same
// node. The two Accept calls instantiate the generic method with different
// type arguments; with struct generics the node would have to be built
// twice, once as *Exp[Env, int] and once as *Exp[any, string].
func EvaluateAndPrint(gmr *ohm.Grammar, src string, env Env) (value int, text string, err error) {
	err = withRoot(gmr, src, func(root ohm.Node) (err error) {
		var exp = &arith_visitor.Exp{
			AddExp: root.Children()[0].(ohm.RuleNode),
		}
		if value, err = exp.Accept[Env, int](root, &evaluator{}, env); err != nil {
			return err
		}
		text, err = exp.Accept[any, string](root, &printer{}, nil)
		return err
	})
	return
}

// withRoot matches src and calls fn with the CST root. The match result
// owns the CST, so it is closed only after fn returns.
func withRoot(gmr *ohm.Grammar, src string, fn func(root ohm.Node) error) error {
	var (
		mr   *ohm.MatchResult
		root ohm.Node
		err  error
	)
	if mr, err = gmr.Match(src); err != nil {
		return fmt.Errorf("matching %q: %w", src, err)
	}
	defer mr.Close()
	if !mr.Succeeded() {
		return fmt.Errorf("%q is not an arithmetic expression", src)
	}
	if root, err = mr.GetCstRoot(); err != nil {
		return fmt.Errorf("reading cst for %q: %w", src, err)
	}
	return fn(root)
}
