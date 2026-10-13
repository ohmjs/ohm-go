package ruleast

import (
	"fmt"
	"slices"
	"strings"

	"github.com/ohmjs/ohm-go/ohm"
	"github.com/samber/lo"
)

func AssertName(this ohm.Node, name string) {
	if this.CtorName() != name {
		panic(fmt.Errorf(`name didn't name.
expected '%s'
received '%s'
`, name, this.CtorName()))
	}
}

type None struct{}

type M_Grammars Grammars[None, None]
type M_Grammar Grammar[None, None]
type M_SuperGrammar SuperGrammar[None, None]
type M_Rule Rule[None, None]
type M_RuleDefine RuleDefine[None, None]
type M_RuleOverride RuleOverride[None, None]
type M_RuleExtend RuleExtend[None, None]
type M_RuleBody RuleBody[None, None]
type M_TopLevelTerm TopLevelTerm[None, None]
type M_TopLevelTermInline TopLevelTermInline[None, None]
type M_OverrideRuleBody OverrideRuleBody[None, None]
type M_OverrideTopLevelTerm OverrideTopLevelTerm[None, None]
type M_OverrideTopLevelTermSuperSplice OverrideTopLevelTermSuperSplice[None, None]
type M_Formals Formals[None, None]
type M_Params Params[None, None]
type M_Alt Alt[None, None]
type M_Seq Seq[None, None]
type M_Iter Iter[None, None]
type M_IterStar IterStar[None, None]
type M_IterPlus IterPlus[None, None]
type M_IterOpt IterOpt[None, None]
type M_Pred Pred[None, None]
type M_PredNot PredNot[None, None]
type M_PredLookahead PredLookahead[None, None]
type M_Lex Lex[None, None]
type M_LexLex LexLex[None, None]
type M_Base Base[None, None]
type M_BaseApplication BaseApplication[None, None]
type M_BaseRange BaseRange[None, None]
type M_BaseTerminal BaseTerminal[None, None]
type M_BaseParen BaseParen[None, None]
type M_LexRuleDescr LexRuleDescr[None, None]
type M_LexRuleDescrText LexRuleDescrText[None, None]
type M_LexCaseName LexCaseName[None, None]
type M_LexName LexName[None, None]
type M_LexNameFirst LexNameFirst[None, None]
type M_LexNameRest LexNameRest[None, None]
type M_LexIdent LexIdent[None, None]
type M_LexTerminal LexTerminal[None, None]
type M_LexOneCharTerminal LexOneCharTerminal[None, None]
type M_LexTerminalChar LexTerminalChar[None, None]
type M_LexEscapeChar LexEscapeChar[None, None]
type M_LexEscapeCharBackslash LexEscapeCharBackslash[None, None]
type M_LexEscapeCharDoubleQuote LexEscapeCharDoubleQuote[None, None]
type M_LexEscapeCharSingleQuote LexEscapeCharSingleQuote[None, None]
type M_LexEscapeCharBackspace LexEscapeCharBackspace[None, None]
type M_LexEscapeCharLineFeed LexEscapeCharLineFeed[None, None]
type M_LexEscapeCharCarriageReturn LexEscapeCharCarriageReturn[None, None]
type M_LexEscapeCharTab LexEscapeCharTab[None, None]
type M_LexEscapeCharUnicodeCodePoint LexEscapeCharUnicodeCodePoint[None, None]
type M_LexEscapeCharUnicodeEscape LexEscapeCharUnicodeEscape[None, None]
type M_LexEscapeCharHexEscape LexEscapeCharHexEscape[None, None]
type M_LexSpace LexSpace[None, None]
type M_LexComment LexComment[None, None]
type M_LexCommentSingleLine LexCommentSingleLine[None, None]
type M_LexCommentMultiLine LexCommentMultiLine[None, None]
type M_LexTokens LexTokens[None, None]
type M_LexToken LexToken[None, None]
type M_LexOperator LexOperator[None, None]
type M_LexPunctuation LexPunctuation[None, None]

func BuildGrammarsHandCodedWalk(root ohm.Node) *M_Grammars {
	return &M_Grammars{
		Grammar: root.Children()[0].(ohm.ListNode),
	}
}

func (node M_Grammars) BuildRuleAst(this ohm.Node) (*GrammarsNode, error) {
	AssertName(this, "Grammars")
	gmrs := map[string]GrammarNode{}
	gmr_names := []string{}
	for _, n := range node.Grammar.Children() {
		kids := n.Children()
		result, err := (&M_Grammar{
			Ident:        kids[0].(ohm.RuleNode),
			SuperGrammar: kids[1].(ohm.OptNode),
			Term1:        kids[2].(ohm.TerminalNode),
			Rule:         kids[3].(ohm.ListNode),
			Term2:        kids[4].(ohm.TerminalNode),
		}).BuildRuleAst(n)
		if err != nil {
			return nil, err
		}
		gmr_names = append(gmr_names, result.Name)
		gmrs[result.Name] = *result
	}
	return new(Make_GrammarsNode(gmr_names, gmrs)), nil
}

func (node M_Grammar) BuildRuleAst(this ohm.Node) (*GrammarNode, error) {
	AssertName(this, "Grammar")
	rules := []string{}
	rmap := map[string]RuleNode{}
	for _, n := range node.Rule.Children() {
		rns, err := (&M_Rule{
			Node: n.(ohm.RuleNode),
		}).BuildRuleAst(n)
		if err != nil {
			return nil, err
		}
		for _, rn := range rns {
			name := rn.GetBranch().RuleName()
			rules = append(rules, name)
			rmap[name] = rn
		}
	}
	return new(Make_GrammarNode(
		node.Ident.SourceString(),
		rules,
		rmap,
	)), nil
}

func (node M_Rule) BuildRuleAst(this ohm.Node) (result []RuleNode, err error) {
	AssertName(this, "Rule")
	switch node.Node.CtorName() {
	case "Rule":
		return (M_Rule{
			Node: node.Node.Children()[0].(ohm.RuleNode),
		}).BuildRuleAst(this)
	case "Rule_define":
		kids := node.Node.Children()
		return (M_RuleDefine{
			Ident:     kids[0].(ohm.RuleNode),
			Formals:   kids[1].(ohm.OptNode),
			RuleDescr: kids[2].(ohm.OptNode),
			Term:      kids[3].(ohm.TerminalNode),
			RuleBody:  kids[4].(ohm.RuleNode),
		}).BuildRuleAst(node.Node)
	case "Rule_override":
		kids := node.Node.Children()
		return (M_RuleOverride{
			Ident:            kids[0].(ohm.RuleNode),
			Formals:          kids[1].(ohm.OptNode),
			Term:             kids[2].(ohm.TerminalNode),
			OverrideRuleBody: kids[3].(ohm.RuleNode),
		}).BuildRuleAst(node.Node)
	case "Rule_extend":
		kids := node.Node.Children()
		return (M_RuleExtend{
			Ident:    kids[0].(ohm.RuleNode),
			Formals:  kids[1].(ohm.OptNode),
			Term:     kids[2].(ohm.TerminalNode),
			RuleBody: kids[3].(ohm.RuleNode),
		}).BuildRuleAst(node.Node)
	default:
		panic("unexpected " + node.Node.CtorName())
	}
}

func (node M_RuleDefine) BuildRuleAst(this ohm.Node) ([]RuleNode, error) {
	AssertName(this, "Rule_define")
	descr := ""
	if len(node.RuleDescr.Children()) > 0 {
		descr = (&M_LexRuleDescr{
			RuleDescrText: node.RuleDescr.Children()[0].Children()[1].(ohm.RuleNode),
		}).BuildRuleAst(node.RuleDescr.Children()[0])
	}
	rb := node.RuleBody.Children()
	details, err := (&M_RuleBody{
		Term:           rb[0].(ohm.OptNode),
		NonemptyListOf: rb[1].(ohm.BHorNode),
	}).BuildRuleAst(node.RuleBody)
	if err != nil {
		return nil, err
	}
	return BuildRuleAstRule(
		details,
		Make_RuleType_define(),
		node.Ident.SourceString(),
		this.SourceString(),
		descr,
	)
}

func (node M_LexRuleDescr) BuildRuleAst(this ohm.Node) string {
	AssertName(this, "ruleDescr")
	return node.RuleDescrText.SourceString()
}

func (node M_RuleExtend) BuildRuleAst(this ohm.Node) ([]RuleNode, error) {
	AssertName(this, "Rule_extend")
	rb := node.RuleBody.Children()
	details, err := (&M_RuleBody{
		Term:           rb[0].(ohm.OptNode),
		NonemptyListOf: rb[1].(ohm.BHorNode),
	}).BuildRuleAst(node.RuleBody)
	if err != nil {
		return nil, err
	}
	return BuildRuleAstRule(
		details,
		Make_RuleType_extend(),
		node.Ident.SourceString(),
		this.SourceString(),
		"",
	)
}

func (node M_RuleOverride) BuildRuleAst(this ohm.Node) ([]RuleNode, error) {
	AssertName(this, "Rule_override")
	rb := node.OverrideRuleBody.Children()
	details, err := (&M_OverrideRuleBody{
		NonemptyListOf: rb[1].(ohm.BHorNode),
	}).BuildRuleAst(node.OverrideRuleBody)
	if err != nil {
		return nil, err
	}
	return BuildRuleAstRule(
		details,
		Make_RuleType_override(),
		node.Ident.SourceString(),
		this.SourceString(),
		"",
	)
}

func (node M_OverrideRuleBody) BuildRuleAst(this ohm.Node) (results []RuleDetailNode, err error) {
	AssertName(this, "OverrideRuleBody")
	for _, el := range node.NonemptyListOf.Elems() {
		rdns, err := (&M_OverrideTopLevelTerm{
			Node: el,
		}).BuildRuleAst(el)
		if err != nil {
			return nil, err
		}
		results = append(results, rdns...)
	}
	return results, nil
}

func (node M_OverrideTopLevelTerm) BuildRuleAst(this ohm.Node) ([]RuleDetailNode, error) {
	AssertName(this, "OverrideTopLevelTerm")
	switch node.Node.CtorName() {
	case "OverrideTopLevelTerm":
		return (M_OverrideTopLevelTerm{
			Node: node.Node.Children()[0].(ohm.RuleNode),
		}).BuildRuleAst(this)
	case "OverrideTopLevelTerm_superSplice":
		kids := node.Node.Children()
		return (M_OverrideTopLevelTermSuperSplice{
			Term: kids[0].(ohm.TerminalNode),
		}).BuildRuleAst(node.Node)
	case "TopLevelTerm":
		resp, err := (&M_TopLevelTerm{
			Node: node.Node.Children()[0].(ohm.RuleNode),
		}).BuildRuleAst(node.Node)
		if err != nil {
			return nil, err
		}
		return []RuleDetailNode{*resp}, nil
	default:
		panic("unexpected " + node.Node.CtorName())
	}
}

func (node M_OverrideTopLevelTermSuperSplice) BuildRuleAst(this ohm.Node) ([]RuleDetailNode, error) {
	panic("not implemented")
}

func BuildRuleAstRule(
	details []RuleDetailNode,
	rule_type RuleType,
	name string,
	sourceString string,
	descr string,
) ([]RuleNode, error) {
	cases := lo.FlatMap[RuleDetailNode, InlineNode](details, func(item RuleDetailNode, index int) []InlineNode {
		if b, ok := item.Cast_inline(); ok {
			return []InlineNode{Make_InlineNode(
				b.Case_name,
				b.Source,
				UnifyBranches2Args([]BareNode{Make_BareNode(b.Args)}, len(b.Args)),
			)}
		}
		return []InlineNode{}
	})
	size := 0
	bare := lo.FlatMap[RuleDetailNode, BareNode](details, func(item RuleDetailNode, index int) []BareNode {
		if b, ok := item.Cast_bare(); ok {
			if size == 0 {
				size = len(b.Args)
			} else if size != len(b.Args) {
				args := lo.Map[NamedArgNode, string](b.Args, func(item NamedArgNode, i int) string {
					return fmt.Sprintf("%d %s %v", i, item.Name, item.Node)
				})
				panic(fmt.Errorf("all branches must have the same number of args size %d curr %d. \n\t%s",
					size,
					len(b.Args),
					strings.Join(args, "\n\t"),
				))
			}
			return []BareNode{b}
		}
		return []BareNode{}
	})
	args := UnifyBranches2Args(bare, size)
	if len(cases) > 0 {
		result := []RuleNode{Make_RuleNode_case_rule(
			Make_CasesRuleNode(
				name,
				Make_RuleType_define(),
				descr,
				sourceString,
				args,
				cases,
			),
		)}
		for _, c := range cases {
			virt_rule := Make_RuleNode_virt_rule(
				VirtRuleNode(
					Make_Named(
						name,
						Make_BareRuleNode(
							c.Case_name,
							Make_RuleType_define(),
							"",
							c.Source,
							c.Args,
						),
					),
				),
			)
			result = append(result, virt_rule)
		}
		return result, nil
	}
	return []RuleNode{Make_RuleNode_bare_rule(
		Make_BareRuleNode(
			name,
			rule_type,
			descr,
			sourceString,
			args,
		),
	)}, nil
}

func (node M_RuleBody) BuildRuleAst(this ohm.Node) (results []RuleDetailNode, err error) {
	AssertName(this, "RuleBody")
	for _, el := range node.NonemptyListOf.Elems() {
		rdn, err := (&M_TopLevelTerm{
			Node: el.(ohm.RuleNode),
		}).BuildRuleAst(el)
		if err != nil {
			return nil, err
		}
		results = append(results, *rdn)
	}
	return results, nil
}

func (node M_TopLevelTerm) BuildRuleAst(this ohm.Node) (*RuleDetailNode, error) {
	AssertName(this, "TopLevelTerm")
	switch node.Node.CtorName() {
	case "TopLevelTerm":
		return (M_TopLevelTerm{
			Node: node.Node.Children()[0].(ohm.RuleNode),
		}).BuildRuleAst(this)
	case "TopLevelTerm_inline":
		kids := node.Node.Children()
		inlineNode, err := (&M_TopLevelTermInline{
			Seq:      kids[0].(ohm.RuleNode),
			CaseName: kids[1].(ohm.RuleNode),
		}).BuildRuleAst(node.Node)
		if err != nil {
			return nil, err
		}
		return new(Make_RuleDetailNode_inline(*inlineNode)), nil
		// Make_RuleDetailNode()
	case "Seq":
		kids := node.Node.Children()
		bnode, err := (&M_Seq{
			Iter: kids[0].(ohm.ListNode),
		}).BuildRuleAst(node.Node)
		if err != nil {
			return nil, err
		}
		return new(Make_RuleDetailNode_bare(*bnode)), nil
	default:
		panic("unexpected " + node.Node.CtorName())
	}
}

func (node M_TopLevelTermInline) BuildRuleAst(this ohm.Node) (*InlineNode, error) {
	AssertName(this, "TopLevelTerm_inline")
	name := node.CaseName.Children()[2].SourceString()
	bnode, err := (&M_Seq{
		Iter: node.Seq.Children()[0].(ohm.ListNode),
	}).BuildRuleAst(node.Seq)
	if err != nil {
		return nil, err
	}
	src := node.Seq.SourceString()
	return new(Make_InlineNode(name, src, bnode.Args)), nil
}

func (node M_Seq) BuildRuleAst(this ohm.Node) (*BareNode, error) {
	AssertName(this, "Seq")
	args := []NamedArgNode{}
	for _, n := range node.Iter.Children() {
		arg, err := (&M_Iter{
			Node: n.(ohm.RuleNode),
		}).BuildRuleAst(n)
		if err != nil {
			return nil, err
		}
		if arg == nil {
			continue
		}
		args = append(args, *arg)
	}
	return new(Make_BareNode(args)), nil
}

func (node M_Iter) BuildRuleAst(this ohm.Node) (*NamedArgNode, error) {
	AssertName(this, "Iter")
	switch node.Node.CtorName() {
	case "Iter":
		return (M_Iter{
			Node: node.Node.Children()[0].(ohm.RuleNode),
		}).BuildRuleAst(this)
	case "Iter_star":
		return (M_IterStar{
			Pred: node.Node.Children()[0].(ohm.RuleNode),
		}).BuildRuleAst(node.Node)
	case "Iter_plus":
		return (M_IterPlus{
			Pred: node.Node.Children()[0].(ohm.RuleNode),
		}).BuildRuleAst(node.Node)
	case "Iter_opt":
		return (M_IterOpt{
			Pred: node.Node.Children()[0].(ohm.RuleNode),
		}).BuildRuleAst(node.Node)
	case "Pred":
		return (M_Pred{
			Node: node.Node.Children()[0].(ohm.RuleNode),
		}).BuildRuleAst(node.Node)
	default:
		panic("unexpected " + node.Node.CtorName())
	}
}

func (node M_IterStar) BuildRuleAst(this ohm.Node) (*NamedArgNode, error) {
	AssertName(this, "Iter_star")
	arg, err := (&M_Pred{
		Node: node.Pred.Children()[0].(ohm.RuleNode),
	}).BuildRuleAst(node.Pred)
	if arg == nil || err != nil {
		return nil, err
	}
	return new(
		NamedArgNode(
			Make_Named(
				arg.Name,
				Make_ArgNode_list(
					Make_ListArgNode(
						arg.Node,
					),
				),
			),
		),
	), nil
}

func (node M_IterPlus) BuildRuleAst(this ohm.Node) (*NamedArgNode, error) {
	AssertName(this, "Iter_plus")
	arg, err := (&M_Pred{
		Node: node.Pred.Children()[0].(ohm.RuleNode),
	}).BuildRuleAst(node.Pred)
	if arg == nil || err != nil {
		return nil, err
	}
	return new(NamedArgNode(Make_Named(
		arg.Name,
		Make_ArgNode_list(
			Make_ListArgNode(
				arg.Node,
			),
		),
	))), nil
}

func (node M_IterOpt) BuildRuleAst(this ohm.Node) (*NamedArgNode, error) {
	AssertName(this, "Iter_opt")
	arg, err := (&M_Pred{
		Node: node.Pred.Children()[0].(ohm.RuleNode),
	}).BuildRuleAst(node.Pred)
	if arg == nil || err != nil {
		return nil, err
	}
	return new(NamedArgNode(Make_Named(
		arg.Name,
		Make_ArgNode_opt(
			Make_OptArgNode(
				arg.Node,
			),
		),
	))), nil
}

func (node M_Pred) BuildRuleAst(this ohm.Node) (*NamedArgNode, error) {
	AssertName(this, "Pred")
	switch node.Node.CtorName() {
	case "Pred":
		return (M_Pred{
			Node: node.Node.Children()[0].(ohm.RuleNode),
		}).BuildRuleAst(this)
	case "Pred_not":
		// none thing consumed
		return nil, nil
	case "Pred_lookahead":
		// none thing consumed
		return nil, nil
	case "Lex":
		return (M_Lex{
			Node: node.Node.Children()[0].(ohm.RuleNode),
		}).BuildRuleAst(node.Node)
	default:
		panic("unexpected " + node.Node.CtorName())
	}
}

func (node M_Lex) BuildRuleAst(this ohm.Node) (*NamedArgNode, error) {
	AssertName(this, "Lex")
	switch node.Node.CtorName() {
	case "Lex":
		return (M_Lex{
			Node: node.Node.Children()[0].(ohm.RuleNode),
		}).BuildRuleAst(this)
	case "Lex_lex":
		return (M_LexLex{
			Term: node.Node.Children()[0].(ohm.TerminalNode),
			Base: node.Node.Children()[1].(ohm.RuleNode),
		}).BuildRuleAst(node.Node)
	case "Base":
		return (M_Base{
			Node: node.Node.Children()[0].(ohm.RuleNode),
		}).BuildRuleAst(node.Node)
	default:
		panic("unexpected " + node.Node.CtorName())
	}
}

func (node M_LexLex) BuildRuleAst(this ohm.Node) (*NamedArgNode, error) {
	return (M_Base{
		Node: node.Base,
	}).BuildRuleAst(node.Base)
}

func (node M_Base) BuildRuleAst(this ohm.Node) (*NamedArgNode, error) {
	AssertName(this, "Base")
	switch node.Node.CtorName() {
	case "Base":
		return (M_Base{
			Node: node.Node.Children()[0].(ohm.RuleNode),
		}).BuildRuleAst(this)
	case "Base_application":
		kids := node.Node.Children()
		return (M_BaseApplication{
			Ident:  kids[0].(ohm.RuleNode),
			Params: kids[1].(ohm.OptNode),
		}).BuildRuleAst(node.Node)
	case "Base_range":
		kids := node.Node.Children()
		return new((&M_BaseRange{
			OneCharTerminal1: kids[0].(ohm.RuleNode),
			Term:             kids[1].(ohm.TerminalNode),
			OneCharTerminal2: kids[2].(ohm.RuleNode),
		}).BuildRuleAst(node.Node)), nil
	case "Base_terminal":
		kids := node.Node.Children()
		return new((&M_BaseTerminal{
			Terminal: kids[0].(ohm.RuleNode),
		}).BuildRuleAst(node.Node)), nil
	case "Base_paren":
		kids := node.Node.Children()
		return new((&M_BaseParen{
			Term1: kids[0].(ohm.TerminalNode),
			Alt:   kids[1].(ohm.RuleNode),
			Term2: kids[2].(ohm.TerminalNode),
		}).BuildRuleAst(node.Node)), nil
	default:
		panic("unexpected " + node.Node.CtorName())
	}
}

var builtInHOR = []string{
	"ListOf", "listOf", "NonemptyListOf", "nonemptyListOf", "EmptyListOf", "emptyListOf",
}

func (node M_BaseApplication) BuildRuleAst(this ohm.Node) (*NamedArgNode, error) {
	AssertName(this, "Base_application")
	if len(node.Params.Children()) > 0 {
		hor_name := node.Ident.SourceString()
		if slices.Contains(builtInHOR, hor_name) {
			params := node.Params.Children()[0]
			listof := params.Children()[1]
			nel := listof.Children()[0]
			seq := nel.Children()[0]
			elem, err := (&M_Seq{
				Iter: seq.Children()[0].(ohm.ListNode),
			}).BuildRuleAst(seq)
			if err != nil {
				return nil, err
			}
			//
			list2 := nel.Children()[1]
			seq2 := list2.Children()[1]
			sep, err := (&M_Seq{
				Iter: seq2.Children()[0].(ohm.ListNode),
			}).BuildRuleAst(seq2)
			if err != nil {
				return nil, err
			}
			if len(elem.Args) != 1 || len(sep.Args) != 1 {
				return nil, fmt.Errorf("expected exactly one argument for elem and sep in builtin hor. got %d and %d", len(elem.Args), len(sep.Args))
			}
			return new(NamedArgNode(Make_Named(
				hor_name,
				Make_ArgNode_bhor(
					Make_BuiltinHorArgNode(
						hor_name,
						elem.Args[0],
						sep.Args[0],
					),
				),
			))), nil
		}
		return new(NamedArgNode(Make_Named(
			node.Ident.SourceString(),
			Make_ArgNode_node(
				Make_NodeArgNode(),
			),
		))), nil
	}
	return new(NamedArgNode(Make_Named(
		node.Ident.SourceString(),
		Make_ArgNode_rule(
			Make_RuleArgNode(
				node.Ident.SourceString(),
			),
		),
	))), nil
}

func (node M_BaseRange) BuildRuleAst(this ohm.Node) NamedArgNode {
	AssertName(this, "Base_range")
	return NamedArgNode(Make_Named(
		"rng",
		Make_ArgNode_term(
			Make_TermArgNode(),
		),
	))
}

func (node M_BaseTerminal) BuildRuleAst(this ohm.Node) NamedArgNode {
	AssertName(this, "Base_terminal")
	return NamedArgNode(Make_Named(
		"term",
		Make_ArgNode_term(
			Make_TermArgNode(),
		),
	))
}

func (node M_BaseParen) BuildRuleAst(this ohm.Node) NamedArgNode {
	AssertName(this, "Base_paren")
	return NamedArgNode(Make_Named(
		"alt",
		Make_ArgNode_node(
			Make_NodeArgNode(),
		),
	))
}
