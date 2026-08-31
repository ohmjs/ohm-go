package ruleast

import (
	"fmt"
	"strings"

	"github.com/samber/lo"
)

func upper1st(s string) string {
	return strings.ToUpper(s[:1]) + s[1:]
}

func r2name(rname string) string {
	if rname[0] >= 'A' && rname[0] <= 'Z' {
		return rname
	}
	return "Lex" + upper1st(rname)
}

func (NodeArgNode) IsNode() bool       { return true }
func (RuleArgNode) IsNode() bool       { return false }
func (TermArgNode) IsNode() bool       { return false }
func (n ListArgNode) IsNode() bool     { return false }
func (OptArgNode) IsNode() bool        { return false }
func (BuiltinHorArgNode) IsNode() bool { return false }
func (AltUnaryRules) IsNode() bool     { return false }

func (NodeArgNode) GoType(rtpkg string) string { return rtpkg + ".Node" }
func (RuleArgNode) GoType(rtpkg string) string { return rtpkg + ".RuleNode" }
func (TermArgNode) GoType(rtpkg string) string { return rtpkg + ".TerminalNode" }
func (n ListArgNode) GoType(rtpkg string) string {
	return rtpkg + ".ListNode"
	// return n.Elem.GetBranch().GoType(rtpkg)
}
func (OptArgNode) GoType(rtpkg string) string        { return rtpkg + ".OptNode" }
func (BuiltinHorArgNode) GoType(rtpkg string) string { return rtpkg + ".BHorNode" }
func (AltUnaryRules) GoType(rtpkg string) string     { return rtpkg + ".RuleNode" }

type GoTypedRule interface {
	Descr(pre, suf string) string
	// TypeName() string
	RuleName() string
	// Source() []string
	// GetArgs() []NamedArgNode
	// GenGoTypes(vc *genTypesCmd)
	// GenGoAccepts(vc *genAcceptsCmd, gmr_name string)
	// GenGoLeafInstAccepts(vc *genAcceptsCmd, tabs int)
}

func (rule RuleNode) GetBranch() GoTypedRule {
	return Handle_RuleNode[GoTypedRule](
		rule,
		func(r BareRuleNode) GoTypedRule {
			return BareRuleNode{r._BareRuleNode}
		},
		func(r VirtRuleNode) GoTypedRule {
			return VirtRuleNode(Named[BareRuleNode]{r._Named})
		},
		func(r CasesRuleNode) GoTypedRule {
			return CasesRuleNode{r._CasesRuleNode}
		},
		nil,
	)
}

func (r BareRuleNode) Descr(pre, suf string) string {
	if r._BareRuleNode.Descr == "" {
		return ""
	}
	return pre + r._BareRuleNode.Descr + suf
}
func (r CasesRuleNode) Descr(pre, suf string) string {
	if r._CasesRuleNode.Descr == "" {
		return ""
	}
	return pre + r._CasesRuleNode.Descr + suf
}
func (r VirtRuleNode) Descr(pre, suf string) string {
	return r.Node.Descr(pre, suf)
}

func (r BareRuleNode) TypeName() string  { return r2name(r.Name) }
func (r CasesRuleNode) TypeName() string { return r2name(r.Name) }
func (r VirtRuleNode) TypeName() string {
	return r2name(r.Name) + upper1st(r.Node.Name)
}

func (r BareRuleNode) RuleName() string  { return r._BareRuleNode.Name }
func (r CasesRuleNode) RuleName() string { return r._CasesRuleNode.Name }
func (r VirtRuleNode) RuleName() string {
	return r._Named.Name + "_" + r._Named.Node._BareRuleNode.Name
}

//	func (r BareRuleNode) Source() []string {
//		return strings.Split(r._BareRuleNode.Source, "\n")
//	}
//
//	func (r CasesRuleNode) Source() []string {
//		return strings.Split(r._CasesRuleNode.Source, "\n")
//	}
func (r VirtRuleNode) Source() string {
	return r.Node.Source
}

// func (r BareRuleNode) GetArgs() []NamedArgNode  { return r.Args }
// func (r CasesRuleNode) GetArgs() []NamedArgNode { return r.Args }
// func (r VirtRuleNode) GetArgs() []NamedArgNode  { return r._Named.Node.Args }

type nodemeta struct {
	type_ string
	name  string
}

func UnifyBranches2Args(bare []BareNode, size int) (args []NamedArgNode) {
	nodetypes := make([]string, size)
	nodenames := make([]string, size)
	nameMap := map[string]int{}
	all_rules := true
	for _, rulebody := range bare {
		for i := range size {
			key := Key4ArgNode(rulebody.Args[i].Node)
			name := rulebody.Args[i].Name
			if key != "rule" {
				all_rules = false
			}
			if nodetypes[i] == "" {
				nodetypes[i] = key
				nodenames[i] = name
				nameMap[rulebody.Args[i].Name]++
				continue
			}
			if nodetypes[i] != key || nodenames[i] != name {
				nodetypes[i] = "node"
				nodenames[i] = "arg"
				nameMap["arg"]++
			} else {
				nodenames[i] = name
			}
		}
	}
	// fmt.Printf("UnifyBranches2Args: size: %d, bare_count %d bare=%v nodetypes=%+v, nodenames=%+v, nameMap=%+v\n", size, len(bare), bare, nodetypes, nodenames, nameMap)
	if size == 1 && len(bare) > 1 && all_rules {
		// fmt.Println("!UnifyBranches2Args: special case for single rule arg with multiple branches")
		x := Make_Named("rule", Make_ArgNode_alt_rules(
			// Make_ArgNode_alt_unary_rules(),
			Make_AltUnaryRules(
				lo.Map[BareNode, RuleArgNode](bare, func(item BareNode, index int) RuleArgNode {
					rule, _ := item.Args[0].Node.Cast_rule()
					return Make_RuleArgNode(rule.Rule)
				}),
			),
		))
		args = []NamedArgNode{NamedArgNode(x)}
		return
	}
	for k, v := range nameMap {
		if v == 1 {
			continue
		}
		c := 1
		for i := range nodenames {
			if nodenames[i] == k {
				nodenames[i] = fmt.Sprintf("%s%d", k, c)
				c++
			}
		}
	}
	for i := range nodetypes {
		var argNode ArgNode = bare[0].Args[i].Node
		if nodetypes[i] == "node" {
			argNode = Make_ArgNode_node(
				Make_NodeArgNode(),
			)
		}
		args = append(args,
			NamedArgNode(
				Make_Named(
					nodenames[i],
					argNode,
				),
			),
		)
	}
	return
}

func Key4ArgNode(arg ArgNode) string {
	return Handle_ArgNode[string](
		arg,
		func(nobj NodeArgNode) string {
			return "node"
		},
		func(rule RuleArgNode) string {
			return "rule"
		},
		func(term TermArgNode) string {
			return "term"
		},
		func(list ListArgNode) string {
			return "list"
		},
		func(opt OptArgNode) string {
			return "opt"
		},
		func(bhor BuiltinHorArgNode) string {
			return "hor"
		},
		func(alt_rules AltUnaryRules) string {
			return "alt_unary_rules"
		},
		nil,
	)
}
