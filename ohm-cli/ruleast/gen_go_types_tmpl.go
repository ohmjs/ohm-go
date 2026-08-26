package ruleast

import (
	"context"
	"crypto/sha256"
	"embed"
	"fmt"
	"go/format"
	"os"
	"strings"

	"github.com/millergarym/gotmpl/text/template"
	"github.com/ohmjs/ohm-go/ohm"
	"github.com/ohmjs/ohm-go/ohm-cli/utils"
)

var (
	//go:embed templates/*
	tmpls embed.FS
)

type genTmplCmd struct {
	GenCmd     genCmd `opts:"mode=embedded"`
	NoGenerics bool   `opts:"short=g" help:"Do not generate generic types (ie with [P any, R any])"`
	Template   string `help:"valid options are GoTypes|GoInterfaces|GoAccepts"`
	OutputFile string
	// for accepts only
	SkipTypeCheckMethod bool `help:"Don't generate TypeCheckMethod code (type checking uses 'MethodByName' which prevents DCE)"`
	ExcludeCli          bool

	gmrsAst   GrammarsNode
	shaSource string
	cli       string

	sbldr *strings.Builder
}

func NewGenInterfaceCmd() any {
	vc := &genTmplCmd{
		GenCmd: genCmd{
			GoRuntimeImport:  "github.com/ohmjs/ohm-go/ohm",
			GoRuntimePackage: "goohm",
			// sbldr:            &strings.Builder{},
			// SuffixOutfLineNos: true,
		},
		Template:   "GoInterfaces",
		OutputFile: "-",
		sbldr:      &strings.Builder{},
		cli:        strings.Join(os.Args, " "),
	}
	// vc.Cli("ohmgo generate parts go_interfaces")
	return vc
}

func NewGenAcceptsCmd() *genTmplCmd {
	vc := &genTmplCmd{
		GenCmd: genCmd{
			GoRuntimeImport:  "github.com/ohmjs/ohm-go/ohm",
			GoRuntimePackage: "goohm",
			// sbldr:            &strings.Builder{},
			// SuffixOutfLineNos: true,
		},
		Template:   "GoAccepts",
		OutputFile: "-",
		sbldr:      &strings.Builder{},
	}
	vc.cli = strings.Join(os.Args, " ")
	// vc.Cli("ohmgo generate parts go_accepts")
	return vc
}

func NewGenTypesCmd() any {
	vc := &genTmplCmd{
		GenCmd: genCmd{
			GoRuntimeImport:  "github.com/ohmjs/ohm-go/ohm",
			GoRuntimePackage: "goohm",
			// sbldr:            &strings.Builder{},
			// SuffixOutfLineNos: true,
		},
		Template:   "GoTypes",
		OutputFile: "-",
		sbldr:      &strings.Builder{},
		cli:        strings.Join(os.Args, " "),
	}
	// vc.Cli("ohmgo generate parts go_types")
	return vc
}

func (vc *genTmplCmd) Cli(prefix string) {
	sb := strings.Builder{}
	sb.WriteString(prefix)
	vc.GenCmd.Cli(&sb)
	if vc.SkipTypeCheckMethod {
		sb.WriteString(" \\\n --skip-type-check-method")
	}
	// if vc.FilePrefix != "" {
	// 	sb.WriteString(" \\\n --file-prefix ")
	// 	sb.WriteString(vc.FilePrefix)
	// }
	if vc.OutputFile != "-" {
		sb.WriteString(" \\\n --output-file ")
		sb.WriteString(vc.OutputFile)
	}
	// sb.WriteString(" \\\n ")
	// sb.WriteString(vc.GenCmd.Grammar)
	vc.cli = sb.String()
}

func (vc *genTmplCmd) Run() error {
	if vc.GenCmd.Grammar[:1] == "@" {
		barr, err := os.ReadFile(vc.GenCmd.Grammar[1:])
		if err != nil {
			return fmt.Errorf("Error reading grammar file. %v", err)
		}
		vc.GenCmd.Grammar = string(barr)
	}
	vc.shaSource = fmt.Sprintf("%x", sha256.Sum256([]byte(vc.GenCmd.Grammar)))
	out, err := vc.Process()
	if err != nil {
		return err
	}
	if vc.OutputFile == "-" {
		fmt.Printf("%[1]s\n", out)
		return nil
	}
	fi, err := os.Create(vc.OutputFile)
	if err != nil {
		return fmt.Errorf("error creating file %w", err)
	}
	defer fi.Close()
	_, err = fi.WriteString(out)
	if err != nil {
		return fmt.Errorf("error writing file %w", err)
	}
	return nil
}

func (vc *genTmplCmd) Process() (string, error) {
	if vc.ExcludeCli {
		vc.cli = ""
	}
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
	if mr, err = gmr.Match(vc.GenCmd.Grammar); err != nil {
		return "", fmt.Errorf("matching: %v", err)
	}
	defer mr.Close()
	if !mr.Succeeded() {
		return "", fmt.Errorf("match failed")
	}
	if root, err = mr.GetCstRoot(); err != nil {
		return "", fmt.Errorf("Error getting cst root. %v", err)
	}
	if vc.GenCmd.HandCodedWalk {
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
		if vc.GenCmd.GrammarName == "" {
			return "", fmt.Errorf("For files with multiple grammars, the --grammar-name flag is required")
		}
	}
	if vc.GenCmd.GrammarName != "" {
		if _, ok := gAst.Grammars[vc.GenCmd.GrammarName]; !ok {
			return "", fmt.Errorf("grammar-name not found. Asked for '%s', grammar names are '%v'", vc.GenCmd.GrammarName, gAst.Gmr_names)
		}
	}
	vc.gmrsAst = *gAst
	err = gAst.GenGoTypes2(vc)
	if err != nil {
		return "", err
	}
	out, err := format.Source([]byte(vc.sbldr.String()))
	if err != nil {
		return vc.sbldr.String(), err
	}
	return string(out), nil
}

func (gmrs GrammarsNode) GenGoTypes2(vc *genTmplCmd) error {
	var gname string
	if vc.GenCmd.GrammarName != "" {
		gname = vc.GenCmd.GrammarName
	} else {
		gname = vc.gmrsAst.Gmr_names[0]
	}
	gmrNode := vc.gmrsAst.Grammars[gname]
	if vc.GenCmd.GoTypePackage == "" {
		vc.GenCmd.GoTypePackage = strings.ToLower(gname)
	}

	// fs.WalkDir(tmpls, ".", func(path string, d fs.DirEntry, err error) error {
	// 	fmt.Fprintf(os.Stderr, "!!%s\n", path)
	// 	return nil
	// })
	global := map[string]any{}
	tmpl := template.Must(template.New("gen", template.WithDynamicScopedVars()).
		Funcs(template.FuncMap{
			"set": func(k string, v any) {
				global[k] = v
			},
			"get": func(k string) any {
				return global[k]
			},
			"trim": func(s string) string {
				return strings.TrimSpace(s)
			},
			"splitnl": func(s string) []string {
				return strings.Split(s, "\n")
			},
			"title": func(s string) string {
				return strings.ToUpper(s[:1]) + s[1:]
			},
			"rulename": func(s string) string {
				if s[0] >= 'A' && s[0] <= 'Z' {
					return s
				}
				return "Lex" + strings.ToUpper(s[:1]) + s[1:]
			},
			"go_rt_pkg": func() string {
				return vc.GenCmd.GoRuntimePackage
			},
			"ruleBranch": func(gmr GrammarNode, name string) GoTypedRule {
				r, ok := gmr.Rules[name]
				if !ok {
					return nil
				}
				return r.GetBranch()
			},
			"panic": func(s string) {
				panic(s)
			},
		}).
		ParseFS(tmpls, "templates/*.tmpl"))
	if vc.GenCmd.SuffixOutfLineNos {
		tmpl.SuffixLineNos("", 0, "", "")
	}
	data := Make_GoTarget(
		vc.GenCmd.GoTypePackage,
		vc.GenCmd.GoRuntimeImport,
		vc.GenCmd.GoRuntimePackage,
		!vc.NoGenerics,
		gmrNode,
		vc.cli,
	)
	err := tmpl.ExecuteTemplate(vc.sbldr, vc.Template, data)
	if err != nil {
		return err
	}
	return nil
}
