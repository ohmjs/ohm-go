package arith

import (
	"fmt"
	"strconv"

	"github.com/ohmjs/ohmgo-examples/generic-methods/arith_visitor"
)

// evaluator computes the value of an expression. The payload is the Env
// identifiers are looked up in and the result is the int value. Rules with
// no Visit method here (Exp, AddExp, MulExp, PriExp) fall through to the
// generated DefaultAccept, which dispatches to the case that matched.
type evaluator struct{}

var (
	_ arith_visitor.VisitorPRE_AddExpPlus[Env, int]   = (*evaluator)(nil)
	_ arith_visitor.VisitorPRE_AddExpMinus[Env, int]  = (*evaluator)(nil)
	_ arith_visitor.VisitorPRE_MulExpTimes[Env, int]  = (*evaluator)(nil)
	_ arith_visitor.VisitorPRE_MulExpDivide[Env, int] = (*evaluator)(nil)
	_ arith_visitor.VisitorPRE_PriExpParen[Env, int]  = (*evaluator)(nil)
	_ arith_visitor.VisitorPRE_LexIdent[Env, int]     = (*evaluator)(nil)
	_ arith_visitor.VisitorRE_LexNumber[Env, int]     = (*evaluator)(nil)
)

// VisitAddExpPlus implements [arith_visitor.VisitorPRE_AddExpPlus].
//
//	AddExp "+" MulExp  -- plus
func (e *evaluator) VisitAddExpPlus(node *arith_visitor.AddExpPlus, env Env) (int, error) {
	var (
		l, r int
		err  error
	)
	if l, err = node.AcceptAddExp[Env, int](e, env); err != nil {
		return 0, err
	}
	if r, err = node.AcceptMulExp[Env, int](e, env); err != nil {
		return 0, err
	}
	return l + r, nil
}

// VisitAddExpMinus implements [arith_visitor.VisitorPRE_AddExpMinus].
//
//	AddExp "-" MulExp  -- minus
func (e *evaluator) VisitAddExpMinus(node *arith_visitor.AddExpMinus, env Env) (int, error) {
	var (
		l, r int
		err  error
	)
	if l, err = node.AcceptAddExp[Env, int](e, env); err != nil {
		return 0, err
	}
	if r, err = node.AcceptMulExp[Env, int](e, env); err != nil {
		return 0, err
	}
	return l - r, nil
}

// VisitMulExpTimes implements [arith_visitor.VisitorPRE_MulExpTimes].
//
//	MulExp "*" PriExp  -- times
func (e *evaluator) VisitMulExpTimes(node *arith_visitor.MulExpTimes, env Env) (int, error) {
	var (
		l, r int
		err  error
	)
	if l, err = node.AcceptMulExp[Env, int](e, env); err != nil {
		return 0, err
	}
	if r, err = node.AcceptPriExp[Env, int](e, env); err != nil {
		return 0, err
	}
	return l * r, nil
}

// VisitMulExpDivide implements [arith_visitor.VisitorPRE_MulExpDivide].
//
//	MulExp "/" PriExp  -- divide
func (e *evaluator) VisitMulExpDivide(node *arith_visitor.MulExpDivide, env Env) (int, error) {
	var (
		l, r int
		text string
		err  error
	)
	if l, err = node.AcceptMulExp[Env, int](e, env); err != nil {
		return 0, err
	}
	if r, err = node.AcceptPriExp[Env, int](e, env); err != nil {
		return 0, err
	}
	if r == 0 {
		// The same node, visited again with the printer's type arguments.
		// Nothing is cast or rebuilt; only the instantiation differs.
		if text, err = node.AcceptPriExp[any, string](&printer{}, nil); err != nil {
			return 0, err
		}
		return 0, fmt.Errorf("division by zero: %s evaluates to 0", text)
	}
	return l / r, nil
}

// VisitPriExpParen implements [arith_visitor.VisitorPRE_PriExpParen].
//
//	"(" Exp ")"  -- paren
func (e *evaluator) VisitPriExpParen(node *arith_visitor.PriExpParen, env Env) (int, error) {
	return node.AcceptExp[Env, int](e, env)
}

// VisitLexIdent implements [arith_visitor.VisitorPRE_LexIdent].
//
//	ident = letter alnum*
func (e *evaluator) VisitLexIdent(node *arith_visitor.LexIdent, env Env) (int, error) {
	var (
		name  = node.Letter.SourceString() + node.Alnum.SourceString()
		v, ok = env[name]
	)
	if !ok {
		return 0, fmt.Errorf("undefined identifier %q", name)
	}
	return v, nil
}

// VisitLexNumber implements [arith_visitor.VisitorRE_LexNumber]. It needs
// no payload, so it uses the RE flavour of the interface.
//
//	number = digit+
func (e *evaluator) VisitLexNumber(node *arith_visitor.LexNumber) (int, error) {
	return strconv.Atoi(node.Digit.SourceString())
}
