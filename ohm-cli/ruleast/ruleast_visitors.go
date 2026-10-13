package ruleast

import (
	"fmt"
	"slices"
	"strings"

	"github.com/ohmjs/ohm-go/ohm"
	"github.com/samber/lo"
)

type ruleAstBuilder struct {
}

// VisitGrammars implements [VisitorGrammars].
func (r *ruleAstBuilder) VisitGrammars(node *Grammars[any, *GrammarsNode]) (result *GrammarsNode, err error) {
	gmrs := map[string]GrammarNode{}
	gmr_names := []string{}
	for _, n := range node.Grammar.Children() {
		kids := n.Children()
		resp, err := (&Grammar[any, *GrammarNode]{
			Ident:        kids[0].(ohm.RuleNode),
			SuperGrammar: kids[1].(ohm.OptNode),
			Term1:        kids[2].(ohm.TerminalNode),
			Rule:         kids[3].(ohm.ListNode),
			Term2:        kids[4].(ohm.TerminalNode),
		}).Accept(n, r, nil)
		if err != nil {
			return nil, err
		}
		gmr_names = append(gmr_names, resp.Name)
		gmrs[resp.Name] = *resp
	}
	return new(Make_GrammarsNode(gmr_names, gmrs)), nil
}

// VisitGrammar implements [VisitorGrammar].
func (r *ruleAstBuilder) VisitGrammar(node *Grammar[any, *GrammarNode]) (result *GrammarNode, err error) {
	rules := []string{}
	rmap := map[string]RuleNode{}
	for _, n := range node.Rule.Children() {
		rns, err := (&Rule[string, []RuleNode]{
			Node: n.(ohm.RuleNode),
		}).Accept(n, r, n.SourceString())
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

// VisitRuleDefine implements [VisitorRuleDefine].
func (r *ruleAstBuilder) VisitRuleDefine(node *RuleDefine[string, []RuleNode], payload string) (result []RuleNode, err error) {
	descr := ""
	if len(node.RuleDescr.Children()) > 0 {
		dn := node.RuleDescr.Children()[0]
		kids := dn.Children()
		descr, _ = (&LexRuleDescr[any, string]{
			Term1:         kids[0].(ohm.TerminalNode),
			RuleDescrText: kids[1].(ohm.RuleNode),
			Term2:         kids[2].(ohm.TerminalNode),
		}).Accept(dn, r, nil)
	}
	return r.BuildRuleAstRule(
		node.RuleBody,
		Make_RuleType_define(),
		node.Ident.SourceString(),
		payload,
		descr,
	)
}

// VisitRuleExtend implements [VisitorRuleExtend].
func (r *ruleAstBuilder) VisitRuleExtend(node *RuleExtend[string, []RuleNode], payload string) (result []RuleNode, err error) {
	return r.BuildRuleAstRule(
		node.RuleBody,
		Make_RuleType_extend(),
		node.Ident.SourceString(),
		payload,
		"",
	)
}

// VisitRuleOverride implements [VisitorPRERuleOverride].
func (r *ruleAstBuilder) VisitRuleOverride(node *RuleOverride[string, []RuleNode], payload string) (result []RuleNode, err error) {
	panic("unimplemented")
}

func (r *ruleAstBuilder) BuildRuleAstRule(
	ruleBody ohm.RuleNode,
	rule_type RuleType,
	name string,
	sourceString string,
	descr string,
) ([]RuleNode, error) {
	rb := ruleBody.Children()
	details, err := (&RuleBody[any, []RuleDetailNode]{
		Term:           rb[0].(ohm.OptNode),
		NonemptyListOf: rb[1].(ohm.BHorNode),
	}).Accept(ruleBody, r, nil)
	if err != nil {
		return nil, err
	}
	cases := lo.FlatMap[RuleDetailNode, InlineNode](details, func(item RuleDetailNode, index int) []InlineNode {
		if b, ok := item.Cast_inline(); ok {
			return []InlineNode{
				Make_InlineNode(
					b.Case_name,
					b.Source,
					UnifyBranches2Args(
						[]BareNode{
							Make_BareNode(b.Args),
						},
						len(b.Args),
					),
				),
			}
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
	// cases_args := lo.Map[InlineNode, InlineNode](cases, func(item InlineNode, index int) InlineNode {
	// 	return Make_InlineNode(
	// 		item.Case_name,
	// 		unifyBranches2Args(bare, size),
	// 	)
	// })
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

// VisitLexRuleDescr implements [VisitorLexRuleDescr].
func (r *ruleAstBuilder) VisitLexRuleDescr(node *LexRuleDescr[any, string]) (result string) {
	return node.RuleDescrText.SourceString()
}

// VisitRuleBody implements [VisitorRuleBody].
func (r *ruleAstBuilder) VisitRuleBody(node *RuleBody[any, []RuleDetailNode]) (result []RuleDetailNode, err error) {
	resp := []RuleDetailNode{}
	for _, el := range node.NonemptyListOf.Elems() {
		rdn, err := (&TopLevelTerm[any, *RuleDetailNode]{
			Node: el.(ohm.RuleNode),
		}).Accept(el, r, nil)
		if err != nil {
			return nil, err
		}
		resp = append(resp, *rdn)
	}
	return resp, nil
}

// VisitTopLevelTermInline implements [VisitorTopLevelTermInline].
func (r *ruleAstBuilder) VisitTopLevelTermInline(node *TopLevelTermInline[any, *RuleDetailNode]) (result *RuleDetailNode, err error) {
	name := node.CaseName.Children()[2].SourceString()
	bnode, err := (&Seq[any, *RuleDetailNode]{
		Iter: node.Seq.Children()[0].(ohm.ListNode),
	}).Accept(node.Seq, r, nil)
	if err != nil {
		return nil, err
	}
	src := node.Seq.SourceString()
	bare, _ := bnode.Cast_bare()
	return new(Make_RuleDetailNode_inline(
		Make_InlineNode(name, src, bare.Args),
	)), nil

	// return WithError[*InlineNode]{
	// 	Val: new(Make_InlineNode(name, src, bnode.Val.Args)),
	// }
}

// VisitSeq implements [VisitorSeq].
func (r *ruleAstBuilder) VisitSeq(node *Seq[any, *RuleDetailNode]) (result *RuleDetailNode, err error) {
	args := []NamedArgNode{}
	for _, n := range node.Iter.Children() {
		arg, err := (&Iter[any, *NamedArgNode]{
			Node: n.(ohm.RuleNode),
		}).Accept(n, r, nil)
		if err != nil {
			return nil, err
		}
		if arg == nil {
			continue
		}
		args = append(args, *arg)
	}
	return new(Make_RuleDetailNode_bare(
		Make_BareNode(args),
	)), nil
	// return WithError[*BareNode]{Val: new(Make_BareNode(args))}
}

// VisitIterStar implements [VisitorIterStar].
func (r *ruleAstBuilder) VisitIterStar(node *IterStar[any, *NamedArgNode]) (result *NamedArgNode, err error) {
	arg, err := (&Pred[any, *NamedArgNode]{
		Node: node.Pred.Children()[0].(ohm.RuleNode),
	}).Accept(node.Pred, r, nil)
	if arg == nil || err != nil {
		// return nil, err
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

// VisitIterPlus implements [VisitorIterPlus].
func (r *ruleAstBuilder) VisitIterPlus(node *IterPlus[any, *NamedArgNode]) (result *NamedArgNode, err error) {
	arg, err := (&Pred[any, *NamedArgNode]{
		Node: node.Pred.Children()[0].(ohm.RuleNode),
	}).Accept(node.Pred, r, nil)
	if arg == nil || err != nil {
		// return nil, err
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

// VisitIterOpt implements [VisitorIterOpt].
func (r *ruleAstBuilder) VisitIterOpt(node *IterOpt[any, *NamedArgNode]) (result *NamedArgNode, err error) {
	arg, err := (&Pred[any, *NamedArgNode]{
		Node: node.Pred.Children()[0].(ohm.RuleNode),
	}).Accept(node.Pred, r, nil)
	if arg == nil || err != nil {
		// return nil, err
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

// VisitPredNot implements [VisitorREPredNot].
func (r *ruleAstBuilder) VisitPredNot(node *PredNot[any, *NamedArgNode]) (result *NamedArgNode, err error) {
	return
}

// VisitPredLookahead implements [VisitorREPredLookahead].
func (r *ruleAstBuilder) VisitPredLookahead(node *PredLookahead[any, *NamedArgNode]) (result *NamedArgNode, err error) {
	return
}

// VisitLexLex implements [VisitorLexLex].
func (r *ruleAstBuilder) VisitLexLex(node *LexLex[any, *NamedArgNode]) (result *NamedArgNode, err error) {
	resp, err := (&Base[any, *NamedArgNode]{
		Node: node.Base,
	}).Accept(node.Base, r, nil)
	return resp, err
}

// VisitBaseApplication implements [VisitorBaseApplication].
func (r *ruleAstBuilder) VisitBaseApplication(node *BaseApplication[any, *NamedArgNode]) (result *NamedArgNode, err error) {
	if len(node.Params.Children()) > 0 {
		hor_name := node.Ident.SourceString()
		if slices.Contains(builtInHOR, hor_name) {
			params := node.Params.Children()[0]
			listof := params.Children()[1]
			nel := listof.Children()[0]
			seq := nel.Children()[0]
			elem, err := (&Seq[any, *RuleDetailNode]{
				Iter: seq.Children()[0].(ohm.ListNode),
			}).Accept(seq, r, nil)
			if err != nil {
				return nil, err
			}
			//
			list2 := nel.Children()[1]
			seq2 := list2.Children()[1]
			sep, err := (&Seq[any, *RuleDetailNode]{
				Iter: seq2.Children()[0].(ohm.ListNode),
			}).Accept(seq2, r, nil)
			if err != nil {
				return nil, err
			}
			ebare, _ := elem.Cast_bare()
			sbare, _ := sep.Cast_bare()
			if len(ebare.Args) != 1 || len(sbare.Args) != 1 {
				return nil, fmt.Errorf(
					"expected exactly one argument for elem and sep in builtin hor. got %d and %d",
					len(ebare.Args),
					len(sbare.Args),
				)
			}
			return new(NamedArgNode(Make_Named(
				hor_name,
				Make_ArgNode_bhor(
					Make_BuiltinHorArgNode(
						hor_name,
						ebare.Args[0],
						sbare.Args[0],
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

// VisitBaseRange implements [VisitorBaseRange].
func (r *ruleAstBuilder) VisitBaseRange(node *BaseRange[any, *NamedArgNode]) (result *NamedArgNode, err error) {
	resp := NamedArgNode(Make_Named(
		"rng",
		Make_ArgNode_term(
			Make_TermArgNode(),
		),
	))
	return &resp, nil
}

// VisitBaseTerminal implements [VisitorBaseTerminal].
func (r *ruleAstBuilder) VisitBaseTerminal(node *BaseTerminal[any, *NamedArgNode]) (result *NamedArgNode, err error) {
	resp := NamedArgNode(Make_Named(
		"term",
		Make_ArgNode_term(
			Make_TermArgNode(),
		),
	))
	return &resp, nil
}

// VisitBaseParen implements [VisitorBaseParen].
func (r *ruleAstBuilder) VisitBaseParen(node *BaseParen[any, *NamedArgNode]) (result *NamedArgNode, err error) {
	resp := NamedArgNode(Make_Named(
		"alt",
		Make_ArgNode_node(
			Make_NodeArgNode(),
		),
	))
	return &resp, nil
}

var (
	_ VisitorRE_Grammars[any, *GrammarsNode]             = (*ruleAstBuilder)(nil)
	_ VisitorRE_Grammar[any, *GrammarNode]               = (*ruleAstBuilder)(nil)
	_ VisitorR_LexRuleDescr[any, string]                 = (*ruleAstBuilder)(nil)
	_ VisitorPRE_RuleDefine[string, []RuleNode]          = (*ruleAstBuilder)(nil)
	_ VisitorPRE_RuleExtend[string, []RuleNode]          = (*ruleAstBuilder)(nil)
	_ VisitorPRE_RuleOverride[string, []RuleNode]        = (*ruleAstBuilder)(nil)
	_ VisitorRE_RuleBody[any, []RuleDetailNode]          = (*ruleAstBuilder)(nil)
	_ VisitorRE_TopLevelTermInline[any, *RuleDetailNode] = (*ruleAstBuilder)(nil)
	_ VisitorRE_Seq[any, *RuleDetailNode]                = (*ruleAstBuilder)(nil)
	_ VisitorRE_IterStar[any, *NamedArgNode]             = (*ruleAstBuilder)(nil)
	_ VisitorRE_IterPlus[any, *NamedArgNode]             = (*ruleAstBuilder)(nil)
	_ VisitorRE_IterOpt[any, *NamedArgNode]              = (*ruleAstBuilder)(nil)
	_ VisitorRE_PredNot[any, *NamedArgNode]              = (*ruleAstBuilder)(nil)
	_ VisitorRE_PredLookahead[any, *NamedArgNode]        = (*ruleAstBuilder)(nil)
	_ VisitorRE_LexLex[any, *NamedArgNode]               = (*ruleAstBuilder)(nil)
	_ VisitorRE_BaseApplication[any, *NamedArgNode]      = (*ruleAstBuilder)(nil)
	_ VisitorRE_BaseRange[any, *NamedArgNode]            = (*ruleAstBuilder)(nil)
	_ VisitorRE_BaseTerminal[any, *NamedArgNode]         = (*ruleAstBuilder)(nil)
	_ VisitorRE_BaseParen[any, *NamedArgNode]            = (*ruleAstBuilder)(nil)
)
