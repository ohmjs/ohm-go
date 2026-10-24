package tests

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/ohmjs/ohm-go/ohm-cli/candp"
	"github.com/ohmjs/ohm-go/ohm-cli/matcherr"
	"github.com/ohmjs/ohm-go/ohm-cli/ruleast"
	"github.com/ohmjs/ohm-go/ohm-cli/sexpr"
	"github.com/ohmjs/ohm-go/ohm-cli/utils"
	"golang.org/x/tools/txtar"
)

type testGmrCmd struct {
	ReMatchTest string
	Strict      bool     `help:"count an expectation that fails to process, such as an input that does not parse, as a failure"`
	TxTarFile   []string `opts:"mode=arg"`
}

func NewTestGmrCmd() *testGmrCmd {
	return &testGmrCmd{}
}

type Processor interface {
	Process() (string, error)
}

type Expect struct {
	file    txtar.File
	process Processor
}

func (cm *testGmrCmd) Run() error {
	// todo check the current dir is cli chdir to tests
	var (
		cwd string
		err error
	)
	if cwd, err = os.Getwd(); err != nil {
		return err
	}
	// names keeps each path as given, to name the inputs in errors.
	names := append([]string(nil), cm.TxTarFile...)
	if filepath.Base(cwd) == "ohm-cli" {
		for i, f := range cm.TxTarFile {
			if cm.TxTarFile[i], err = filepath.Abs(f); err != nil {
				return err
			}
		}
		if err = os.Chdir("tests"); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("only valid if run from cli directory. Current dir is %s", cwd)
	}
	// Every file runs, so one failing fixture does not hide the next.
	var errs []error
	for i, f := range cm.TxTarFile {
		if err = testFile(cm.ReMatchTest, f, names[i], cm.Strict); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// TestFile runs the expectations of one txtar file. An expectation that
// fails to process is reported and not counted; see RunTests.
func TestFile(reMatchTest string, txTarFile string) error {
	return testFile(reMatchTest, txTarFile, txTarFile, false)
}

// testFile runs the expectations of the txtar file at txTarFile. name is
// the path of the file as the user gave it. It names the archive in the
// output, and an input as `name:member` in a match error.
// See error-reporting prd section 9.
func testFile(reMatchTest string, txTarFile string, name string, strict bool) error {
	ar, err := txtar.ParseFile(txTarFile)
	if err != nil {
		return err
	}
	var gmr *txtar.File
	gmrName := "" // the name of the grammar in a match error
	// var gmr *string
	sepxrs := []Expect{}
	candps := map[string][]Expect{}
	txtsMap := map[string]string{}
	txtKeys := []string{}
	// txtName answers the name of the input of a test in a match error.
	txtNames := map[string]string{}
	txtName := func(test string) string { return name + ":" + txtNames[test] }
	// grammarFile writes the archive grammar to a temp file once, and
	// answers its path. Only a kind that compiles the grammar calls it, so
	// an archive of `o` fixtures needs no grammar.
	gmrPath := ""
	grammarFile := func(name string) (string, error) {
		if gmr == nil {
			panic(fmt.Errorf("expected grammar file before %s. %v", name, txTarFile))
		}
		if gmrPath != "" {
			return gmrPath, nil
		}
		tempdir, err := os.MkdirTemp("", "ohm-compile-*")
		if err != nil {
			return "", fmt.Errorf("creating tempdir: %v", err)
		}
		gmrp := filepath.Join(tempdir, gmr.Name)
		if err = os.WriteFile(gmrp, gmr.Data, 0o644); err != nil {
			os.RemoveAll(tempdir)
			return "", fmt.Errorf("writing grammar file: %v", err)
		}
		gmrPath = gmrp
		return gmrPath, nil
	}
	defer func() {
		if gmrPath != "" {
			os.RemoveAll(filepath.Dir(gmrPath))
		}
	}()
	for _, fi := range ar.Files {
		parts := strings.Split(fi.Name, ".")
		ext := parts[len(parts)-1]
		testName := parts[0]
		if ext != "ohm" && reMatchTest != "" {
			ok, err := regexp.MatchString(reMatchTest, testName)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
		}
		if fi.Name[:1] == "!" {
			fmt.Fprintf(os.Stderr, "skipping %s\n", fi.Name)
			continue
		}
		switch ext {
		case "ohm":
			if gmr != nil {
				panic(fmt.Errorf("only one grammar file per txtar. %v", txTarFile))
			}
			if strings.HasPrefix(fi.Name, "@") {
				data, err := os.ReadFile(fi.Name[1:])
				if err != nil {
					panic(err)
				}
				paths := strings.Split(fi.Name, "/")
				base := strings.TrimPrefix(paths[len(paths)-1], "@")
				gmr = &txtar.File{
					Name: base,
					Data: data,
				}
				gmrName = filepath.Join(filepath.Dir(name), fi.Name[1:])
				continue
			}
			gmr = &fi
			gmrName = name + ":" + fi.Name
		case "sexpr":
			if gmr == nil {
				panic(fmt.Errorf("expected grammar file before %s. %v", fi.Name, txTarFile))
			}
			toSexpr := &sexpr.ToSexprCmd{
				Grammar:   string(gmr.Data),
				InputName: gmrName,
			}
			for _, flag := range parts[1 : len(parts)-1] {
				switch flag {
				case "t":
					toSexpr.ExcludeTerminals = true
				case "S":
					toSexpr.SyntacticRulesOnly = true
				case "b":
					toSexpr.ExcludeBuiltin = true
				case "l":
					toSexpr.PruneLists = true
				case "v":
					toSexpr.Verbose = true
				case "n":
					toSexpr.NoClosingNewline = true
				default:
					panic("unkown short flag : " + flag)
				}
			}
			sepxrs = append(sepxrs, Expect{
				file:    fi,
				process: toSexpr,
			})
		case "txt":
			if _, ok := txtsMap[parts[0]]; ok {
				panic(fmt.Errorf("duplicate txt file. %s", parts[0]))
			}
			txtKeys = append(txtKeys, parts[0])
			txtsMap[parts[0]] = string(fi.Data)
			txtNames[parts[0]] = fi.Name
		case "c_and_p":
			if _, ok := txtsMap[parts[0]]; !ok {
				panic(fmt.Errorf("txt file doesn't exist for c and p. %s", parts[0]))
			}
			gmrp, err := grammarFile(fi.Name)
			if err != nil {
				return err
			}
			candp := &candp.CompileAndParseCmd{
				GrammarFile: gmrp,
				Source:      txtsMap[parts[0]],
				InputName:   txtName(parts[0]),
			}
			for _, flag := range parts[1 : len(parts)-1] {
				switch flag {
				case "t":
					candp.ExcludeTerminals = true
				case "S":
					candp.SyntacticRulesOnly = true
				case "b":
					candp.ExcludeBuiltin = true
				case "l":
					candp.PruneLists = true
				case "v":
					candp.Verbose = true
				case "n":
					candp.NoClosingNewline = true
				default:
					panic("unkown short flag : " + flag)
				}
			}
			candps[parts[0]] = append(candps[parts[0]], Expect{
				file:    fi,
				process: candp,
			})
		case "rule_ast":
			if _, ok := txtsMap[parts[0]]; !ok {
				panic(fmt.Errorf("txt file doesn't exist for rule ast. %s", parts[0]))
			}
			rast := &ruleast.RuleAstCmd{
				Grammar:   txtsMap[parts[0]],
				InputName: txtName(parts[0]),
			}
			for _, flag := range parts[1 : len(parts)-1] {
				switch flag {
				case "s":
					rast.SkipSource = true
				case "l":
					rast.SuffixOutfLineNos = true
				default:
					panic("unkown short flag : " + flag)
				}
			}
			candps[parts[0]] = append(candps[parts[0]], Expect{
				file:    fi,
				process: rast,
			})
		case "go_types", "go_interfaces", "go_accepts":
			if _, ok := txtsMap[parts[0]]; !ok {
				panic(fmt.Errorf("txt file doesn't exist for %s. %s", ext, parts[0]))
			}
			candps[parts[0]] = append(candps[parts[0]], Expect{
				file:    fi,
				process: newGenTmplExpect(ext, parts[1:len(parts)-1], txtsMap[parts[0]], txtName(parts[0])),
			})
		case "error", "expected":
			if _, ok := txtsMap[parts[0]]; !ok {
				panic(fmt.Errorf("txt file doesn't exist for %s. %s", ext, parts[0]))
			}
			cmd := &matcherr.MatchErrorCmd{
				Source:   txtsMap[parts[0]],
				Expected: ext == "expected",
			}
			ohmGrammar := false
			for _, flag := range parts[1 : len(parts)-1] {
				if rule, ok := startFlag(flag); ok {
					cmd.Start = rule
					continue
				}
				switch {
				case flag == "o":
					ohmGrammar = true
				case flag == "N":
					cmd.Source = strings.TrimSuffix(cmd.Source, "\n")
				case flag == "s" && ext == "error":
					cmd.Short = true
				case flag == "f" && ext == "expected":
					cmd.Fluffy = true
				default:
					panic("unknown short flag for " + ext + " : " + flag)
				}
			}
			if !ohmGrammar {
				if cmd.Grammar, err = grammarFile(fi.Name); err != nil {
					return err
				}
			}
			candps[parts[0]] = append(candps[parts[0]], Expect{
				file:    fi,
				process: cmd,
			})
		}
	}
	mismatches, errs := RunTests(name, gmr, sepxrs)
	for _, f := range txtKeys {
		m, e := RunTests(name, gmr, candps[f])
		mismatches += m
		errs += e
	}
	if strict && errs > 0 {
		return fmt.Errorf("%d expectation(s) did not match and %d failed to process in %s", mismatches, errs, name)
	}
	if mismatches > 0 {
		return fmt.Errorf("%d expectation(s) did not match in %s", mismatches, name)
	}
	return nil
}

// startFlag parses the flag part `start=<Rule>`, and answers the rule.
// A rule name holds no `.`, so it survives the split of the file name.
func startFlag(flag string) (string, bool) {
	rule, ok := strings.CutPrefix(flag, "start=")
	if !ok || rule == "" {
		return "", false
	}
	return rule, true
}

// newGenTmplExpect builds the generator command for a `<test>.<flags>.<ext>`
// file, where ext is one of go_types, go_interfaces or go_accepts.
// inputName names the grammar in a match error. Single
// letter flags between the test name and the extension set options:
//
//	m   --generic-methods
//	g   --no-generics
//	s   --skip-type-check-method
func newGenTmplExpect(ext string, flags []string, grammar, inputName string) Processor {
	var cmd *ruleast.GenTmplCmd
	switch ext {
	case "go_types":
		cmd = ruleast.NewGenTypesCmd()
	case "go_interfaces":
		cmd = ruleast.NewGenInterfaceCmd()
	case "go_accepts":
		cmd = ruleast.NewGenAcceptsCmd()
	default:
		panic("unknown generator extension : " + ext)
	}
	cmd.ExcludeCli = true
	cmd.GenCmd.SuffixOutfLineNos = false
	cmd.GenCmd.Grammar = grammar
	cmd.GenCmd.InputName = inputName
	for _, flag := range flags {
		switch flag {
		case "m":
			cmd.GenCmd.GenericMethods = true
		case "g":
			cmd.NoGenerics = true
		case "s":
			cmd.SkipTypeCheckMethod = true
		default:
			panic("unkown short flag : " + flag)
		}
	}
	return cmd
}

// RunTests processes every expectation and prints a diff for each one whose
// output differs from the fixture. It returns the number of mismatches and
// the number of processing errors. A processing error (for example the
// docker compile behind c_and_p failing, or an input that does not parse)
// is reported, and fails the run only under --strict, so an environment
// without docker still exercises the pure-Go generators. An input that
// does not parse is reported as `path:line:col: expected …` and its
// excerpt (error-reporting prd section 9).
func RunTests(
	txTarFile string,
	gmr *txtar.File,
	tests []Expect,
) (mismatches, errs int) {
	for _, exp := range tests {
		var (
			rec string
			err error
		)
		rec, err = exp.process.Process()
		if err != nil {
			errs++
			if merr := (*utils.MatchError)(nil); errors.As(err, &merr) {
				// The excerpt shows the input, so the grammar and the
				// fixture are not printed.
				fmt.Fprintf(os.Stderr, "gen result error for %s, err :\n%v\n", exp.file.Name, merr)
				continue
			}
			fmt.Fprintf(os.Stderr, "gen result error for %s, err : %v\n", exp.file.Name, err)
			if gmr != nil {
				fmt.Fprintf(os.Stderr, " gmr '%s'\n", string(gmr.Data))
			}
			fmt.Fprintf(os.Stderr, " text '%s'\n'%s'\n", exp.file.Name, string(exp.file.Data))
			continue
		}
		expected := strings.TrimSpace(string(exp.file.Data))
		received := strings.TrimSpace(rec)
		if expected != received {
			mismatches++
			fmt.Fprintf(os.Stderr, "! vv sexpr doesn't match %s %s\n", txTarFile, exp.file.Name)
			fmt.Fprintf(os.Stderr, "expected:\n")
			fmt.Fprintf(os.Stderr, "%s\n", expected)
			fmt.Fprintf(os.Stderr, "received:\n")
			fmt.Fprintf(os.Stdout, "%s\n", received)
			fmt.Fprintf(os.Stderr, "diff    :\n%s\n", cmp.Diff(expected, received))
			fmt.Fprintf(os.Stderr, "! ^^ sexpr doesn't match %s %s\n", txTarFile, exp.file.Name)
		} else {
			fmt.Fprintf(os.Stderr, "+ matches %s %s\n", txTarFile, exp.file.Name)
		}
	}
	return mismatches, errs
}
