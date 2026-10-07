package arith

import "github.com/ohmjs/ohmgo-examples/generic-methods/arith_visitor"

// printer renders an expression fully parenthesised. It takes no payload
// (P=any, always nil) and returns a string.
type printer struct{}

var (
	_ arith_visitor.VisitorRE_AddExpPlus[any, string]   = (*printer)(nil)
	_ arith_visitor.VisitorRE_AddExpMinus[any, string]  = (*printer)(nil)
	_ arith_visitor.VisitorRE_MulExpTimes[any, string]  = (*printer)(nil)
	_ arith_visitor.VisitorRE_MulExpDivide[any, string] = (*printer)(nil)
	_ arith_visitor.VisitorRE_PriExpParen[any, string]  = (*printer)(nil)
	_ arith_visitor.VisitorRE_LexIdent[any, string]     = (*printer)(nil)
	_ arith_visitor.VisitorRE_LexNumber[any, string]    = (*printer)(nil)
)

// acceptFn is the shape of an instantiated Accept<Arg>[any, string] method
// value, which is what the printer's binary helper takes.
type acceptFn = func(visitor any, payload any) (string, error)

// VisitAddExpPlus implements [arith_visitor.VisitorRE_AddExpPlus].
func (p *printer) VisitAddExpPlus(node *arith_visitor.AddExpPlus) (string, error) {
	return p.binary(node.AcceptAddExp[any, string], "+", node.AcceptMulExp[any, string])
}

// VisitAddExpMinus implements [arith_visitor.VisitorRE_AddExpMinus].
func (p *printer) VisitAddExpMinus(node *arith_visitor.AddExpMinus) (string, error) {
	return p.binary(node.AcceptAddExp[any, string], "-", node.AcceptMulExp[any, string])
}

// VisitMulExpTimes implements [arith_visitor.VisitorRE_MulExpTimes].
func (p *printer) VisitMulExpTimes(node *arith_visitor.MulExpTimes) (string, error) {
	return p.binary(node.AcceptMulExp[any, string], "*", node.AcceptPriExp[any, string])
}

// VisitMulExpDivide implements [arith_visitor.VisitorRE_MulExpDivide].
func (p *printer) VisitMulExpDivide(node *arith_visitor.MulExpDivide) (string, error) {
	return p.binary(node.AcceptMulExp[any, string], "/", node.AcceptPriExp[any, string])
}

// VisitPriExpParen implements [arith_visitor.VisitorRE_PriExpParen]. The
// source parentheses are dropped; binary adds its own.
func (p *printer) VisitPriExpParen(node *arith_visitor.PriExpParen) (string, error) {
	return node.AcceptExp[any, string](p, nil)
}

// VisitLexIdent implements [arith_visitor.VisitorRE_LexIdent].
func (p *printer) VisitLexIdent(node *arith_visitor.LexIdent) (string, error) {
	return node.Letter.SourceString() + node.Alnum.SourceString(), nil
}

// VisitLexNumber implements [arith_visitor.VisitorRE_LexNumber].
func (p *printer) VisitLexNumber(node *arith_visitor.LexNumber) (string, error) {
	return node.Digit.SourceString(), nil
}

// binary visits both operands with the printer and joins them with op.
func (p *printer) binary(left acceptFn, op string, right acceptFn) (string, error) {
	var (
		l, r string
		err  error
	)
	if l, err = left(p, nil); err != nil {
		return "", err
	}
	if r, err = right(p, nil); err != nil {
		return "", err
	}
	return "(" + l + " " + op + " " + r + ")", nil
}
