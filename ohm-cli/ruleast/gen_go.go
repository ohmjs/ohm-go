package ruleast

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ohmjs/ohm-go/ohm"
	"github.com/ohmjs/ohm-go/ohm-cli/utils"
)

type genGoCmd struct {
	GenCmd              genCmd `opts:"mode=embedded"`
	OutputDir           string `help:"Output directory for generated files.\nDefaults to go-type-package, which defaults to the lowercase grammar name."`
	SkipTypeCheckMethod bool   `help:"Don't generate TypeCheckMethod code (type checking uses 'MethodByName' which prevents DCE)"`
	FilePrefix          string `help:"Prefix for generated files."`
	cli                 string
}

func NewGenGoCmd() *genGoCmd {
	return &genGoCmd{
		GenCmd: genCmd{
			GoRuntimeImport:  "github.com/ohmjs/ohm-go/ohm",
			GoRuntimePackage: "ohm",
			// sbldr:             &strings.Builder{},
			// SuffixOutfLineNos: true,
		},
	}
}

func (vc *genGoCmd) Cli() {
	sb := strings.Builder{}
	sb.WriteString("ohm-cli generate go")
	vc.GenCmd.Cli(&sb)
	if vc.SkipTypeCheckMethod {
		sb.WriteString(" \\\n --skip-type-check-method")
	}
	if vc.FilePrefix != "" {
		sb.WriteString(" \\\n --file-prefix ")
		sb.WriteString(vc.FilePrefix)
	}
	if vc.OutputDir != "" {
		sb.WriteString(" \\\n --output-dir ")
		sb.WriteString(vc.OutputDir)
	}
	// sb.WriteString(" \\\n ")
	// sb.WriteString(vc.GenCmd.Grammar)
	vc.cli = sb.String()
}

func (vc *genGoCmd) Run() error {
	if vc.GenCmd.GoTypePackage != "" && vc.OutputDir == "" {
		vc.OutputDir = vc.GenCmd.GoTypePackage
	}
	ctx := context.Background()
	var (
		gmr  *ohm.Grammar
		mr   *ohm.MatchResult
		err  error
		root ohm.Node
		gAst *GrammarsNode
	)
	if vc.GenCmd.Grammar[:1] == "@" {
		barr, err := os.ReadFile(vc.GenCmd.Grammar[1:])
		if err != nil {
			return fmt.Errorf("Error reading grammar file. %[1]v", err)
		}
		vc.GenCmd.Grammar = string(barr)
	}
	if gmr, err = ohm.NewGrammar(ctx, utils.OhmGrammarWasmBytes()); err != nil {
		return fmt.Errorf("creating grammar: %[1]v", err)
	}
	defer gmr.Close()
	if mr, err = gmr.Match(vc.GenCmd.Grammar); err != nil {
		return fmt.Errorf("matching: %[1]v", err)
	}
	defer mr.Close()
	if !mr.Succeeded() {
		return fmt.Errorf("match failed")
	}
	if root, err = mr.GetCstRoot(); err != nil {
		return fmt.Errorf("Error getting cst root. %[1]v", err)
	}
	if vc.GenCmd.HandCodedWalk {
		gAst, err = BuildGrammarsHandCodedWalk(root).BuildRuleAst(root)
		if err != nil {
			return err
		}
	} else {
		gmrs := MakeNodeFromRoot[any, *GrammarsNode](root)
		v := &ruleAstBuilder{}
		gAst, err = gmrs.Accept(root, v, nil)
		if err != nil {
			return err
		}
	}
	if len(gAst.Gmr_names) > 1 {
		if vc.GenCmd.GrammarName == "" {
			return fmt.Errorf("For files with multiple grammars, the --grammar-name flag is required. Found grammars '%v", gAst.Gmr_names)
		}
	}
	if vc.GenCmd.GrammarName != "" {
		if _, ok := gAst.Grammars[vc.GenCmd.GrammarName]; !ok {
			return fmt.Errorf("grammar-name not found. Asked for '%s', grammar names are '%v'", vc.GenCmd.GrammarName, gAst.Gmr_names)
		}
	}
	if vc.OutputDir == "" {
		vc.OutputDir = strings.ToLower(gAst.Gmr_names[0])
	}
	if err = os.MkdirAll(vc.OutputDir, os.ModePerm); err != nil {
		return fmt.Errorf("creating output directory: %[1]v", err)
	}
	vc.Cli()
	if err := vc.do(gAst, "GoTypes", "types.go"); err != nil {
		return err
	}
	if err := vc.do(gAst, "GoInterfaces", "interfaces.go"); err != nil {
		return err
	}
	if err := vc.do(gAst, "GoAccepts", "accepts.go"); err != nil {
		return err
	}
	return nil
}

func (vc *genGoCmd) do(gAst *GrammarsNode, tmplName, suffix string) error {
	tmplCmd := &genTmplCmd{
		GenCmd:     vc.GenCmd,
		OutputFile: filepath.Join(vc.OutputDir, vc.FilePrefix+suffix),
		gmrsAst:    *gAst,
		Template:   tmplName,
		sbldr:      &strings.Builder{},
		cli:        vc.cli,
	}
	out, err := tmplCmd.Process()
	if err != nil {
		return fmt.Errorf("process error : %[1]v\nsource: '%[2]s'", err, out)
	}
	if file, err := os.Create(tmplCmd.OutputFile); err != nil {
		return fmt.Errorf("creating file: %[1]v", err)
	} else {
		defer file.Close()
		if _, err = file.WriteString(out); err != nil {
			return fmt.Errorf("writing file: %[1]v", err)
		}
		fmt.Fprintf(os.Stderr, "wrote %s\n", tmplCmd.OutputFile)
	}
	return nil
}
