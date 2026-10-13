package ruleast

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/ohmjs/ohm-go/ohm"
	"github.com/ohmjs/ohm-go/ohm-cli/utils"
	"github.com/samber/lo"
)

type RuleAstCmd struct {
	SkipSource        bool   `opts:"short=s"`
	SuffixOutfLineNos bool   `opts:"short=l"`
	Grammar           string `opts:"mode=arg" help:"Grammar source. If start with an @ it is a path to an .ohm grammar file."`
	GrammarName       string `help:"For multiple grammars per file, the name of the grammar to generate code for. Only used for multi-grammar files."`
	HandCodedWalk     bool   `opts:"short=H" help:"if set, use the hand coded walk. Default is to use the visitors. Useful for benchmarking the difference."`

	sbldr *strings.Builder
}

func NewRuleAstCmd() *RuleAstCmd {
	return &RuleAstCmd{
		sbldr: &strings.Builder{},
	}
}

func (vc *RuleAstCmd) outf(format string, a ...any) {
	_, file, line, _ := runtime.Caller(1) // for line numbers to be correct when SuffixOutfLineNos is true
	parts := strings.Split(file, "/")
	parts = parts[len(parts)-2:]
	file = strings.Join(parts, "/")
	callerLine := fmt.Sprintf("%s:%d", file, line)
	f0 := format
	if vc.SuffixOutfLineNos {
		f0 = strings.ReplaceAll(format, "\n", fmt.Sprintf(" // %%[%d]s\n", len(a)+1))
		a = append(a, callerLine)
	}
	fmt.Fprintf(vc.sbldr, f0, a...)
}

func (vc *RuleAstCmd) Run() error {
	if vc.Grammar[:1] == "@" {
		barr, err := os.ReadFile(vc.Grammar[1:])
		if err != nil {
			return fmt.Errorf("Error reading grammar file. %v", err)
		}
		vc.Grammar = string(barr)
	}
	out, err := vc.Process()
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", out)
	return nil
}

func (vc *RuleAstCmd) Process() (string, error) {
	vc.sbldr = &strings.Builder{}
	ctx := context.Background()
	var (
		gmr  *ohm.Grammar
		mr   *ohm.MatchResult
		err  error
		root ohm.Node
		gAst *GrammarsNode
	)
	if gmr, err = ohm.NewGrammar(ctx, utils.OhmGrammarWasmBytes()); err != nil {
		return "", fmt.Errorf("creating grammar: %v", err)
	}
	defer gmr.Close()
	if mr, err = gmr.Match(vc.Grammar); err != nil {
		return "", fmt.Errorf("matching: %v", err)
	}
	defer mr.Close()
	if !mr.Succeeded() {
		return "", fmt.Errorf("match failed")
	}
	if root, err = mr.GetCstRoot(); err != nil {
		return "", fmt.Errorf("Error getting cst root. %v", err)
	}
	if vc.HandCodedWalk {
		gAst, err = BuildGrammarsHandCodedWalk(root).BuildRuleAst(root)
		if err != nil {
			return "", err
		}
	} else {
		gmrs := MakeNodeFromRoot[any, *GrammarsNode](root)
		v := &ruleAstBuilder{}
		gAst, err = gmrs.Accept(root, v, nil)
		if err != nil {
			return "", err
		}
	}
	if len(gAst.Gmr_names) > 1 {
		if vc.GrammarName == "" {
			return "", fmt.Errorf("For files with multiple grammars, the --grammar-name flag is required")
		}
	}
	if vc.GrammarName != "" {
		if _, ok := gAst.Grammars[vc.GrammarName]; !ok {
			return "", fmt.Errorf("grammar-name not found. Asked for '%s', grammar names are '%v'", vc.GrammarName, gAst.Gmr_names)
		}
	}
	var gname string
	if vc.GrammarName != "" {
		gname = vc.GrammarName
	} else {
		gname = gAst.Gmr_names[0]
	}
	v := gAst.Grammars[gname]
	vc.outf("%[1]s\n", gname)
	for _, name := range v.Rule_names {
		rule0 := v.Rules[name]
		branch := rule0.GetBranch()
		descr := branch.Descr("(", ") ")
		Handle_RuleNode[any](
			rule0,
			func(rule BareRuleNode) any {
				if !vc.SkipSource {
					vc.outf("```\n%[1]s\n```\n", rule._BareRuleNode.Source)
				}
				vc.outf("%[1]s @bare %[2]s{\n", rule.RuleName(), descr)
				for _, a := range rule.Args {
					vc.outf("  %[1]s %+[2]v\n", strings.ToLower(a.Name[:1])+a.Name[1:], argNodeStr(a.Node))
				}
				vc.outf("}\n")
				return nil
			},
			func(rule VirtRuleNode) any {
				if !vc.SkipSource {
					vc.outf("```\n%s\n```\n", rule.Node._BareRuleNode.Source)
				}
				vc.outf("%[1]s @virt %[2]s %[3]s{\n", rule._Named.Name, rule.Node._BareRuleNode.Name, descr)
				for _, a := range rule.Node.Args {
					vc.outf("  %[1]s %+[2]v\n", strings.ToLower(a.Name[:1])+a.Name[1:], argNodeStr(a.Node))
				}
				vc.outf("}\n")
				return nil
			},
			func(rule CasesRuleNode) any {
				if !vc.SkipSource {
					vc.outf("```\n%[1]s\n```\n", rule._CasesRuleNode.Source)
				}
				vc.outf("%[1]s @cases %[2]s{\n", rule.RuleName(), descr)
				for _, c := range rule.Cases {
					vc.outf("  %[1]s @inline\n", c.Case_name)
				}
				for _, a := range rule.Args {
					vc.outf("  %[1]s %+[2]v\n", strings.ToLower(a.Name[:1])+a.Name[1:], argNodeStr(a.Node))
				}
				vc.outf("}\n")
				return nil
			},
			nil,
		)
	}
	return vc.sbldr.String(), nil
}

func argNodeStr(arg ArgNode) string {
	return Handle_ArgNode[string](
		arg,
		func(nobj NodeArgNode) string {
			return "@node"
		},
		func(rule RuleArgNode) string {
			return "@rule " + rule.Rule
		},
		func(term TermArgNode) string {
			return "@term"
		},
		func(list ListArgNode) string {
			return "@list " + argNodeStr(list.Elem)
		},
		func(opt OptArgNode) string {
			return "@opt " + argNodeStr(opt.Elem)
		},
		func(bhor BuiltinHorArgNode) string {
			return fmt.Sprintf("@hor %s<%s, %s>",
				bhor.List_type,
				argNodeStr(bhor.Elem.Node),
				argNodeStr(bhor.Sep.Node),
			)
		},
		func(alt_rules AltUnaryRules) string {
			names := lo.Map[RuleArgNode, string](alt_rules.Rules, func(item RuleArgNode, index int) string {
				return item.Rule
			})
			return "@rules " + strings.Join(names, " ")
		},
		nil,
	)
}
