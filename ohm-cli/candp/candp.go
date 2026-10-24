package candp

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ohmjs/ohm-go/ohm"
	"github.com/ohmjs/ohm-go/ohm-cli/utils"
	"github.com/samber/lo"
)

type CompileAndParseCmd struct {
	KeepWasm           bool   `opts:"short=K"`
	ExcludeTerminals   bool   `opts:"short=t"`
	SyntacticRulesOnly bool   `opts:"short=S"`
	ExcludeBuiltin     bool   `opts:"short=b"`
	PruneLists         bool   `opts:"short=l"`
	NoClosingNewline   bool   `opts:"short=n"`
	Verbose            bool   `opts:"short=v" help:"Output source (start & end) and source string for every node"`
	GrammarFile        string `opts:"mode=arg"`
	Source             string `opts:"mode=arg"`
	// InputName names the input in a match error: its path, or
	// `archive:member` for a txtar input. See utils.NewMatchError.
	InputName string `opts:"-"`
}

var builtins = map[string]struct{}{
	"any":      {},
	"letter":   {},
	"lower":    {},
	"upper":    {},
	"digit":    {},
	"hexDigit": {},
	"alnum":    {},
	"space":    {},
	"end":      {},
}

func NewCompileAndParseCmd() *CompileAndParseCmd {
	return &CompileAndParseCmd{}
}

func (vc *CompileAndParseCmd) Run() error {
	if len(vc.Source) > 2 && vc.Source[:1] == "@" {
		vc.InputName = vc.Source[1:]
		barr, err := os.ReadFile(vc.Source[1:])
		if err != nil {
			return fmt.Errorf("Error reading grammar file. %v", err)
		}
		vc.Source = string(barr)
	}
	out, err := vc.Process()
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", out)
	return nil
}

// wasmCache holds each grammar compiled in this process, keyed by the
// docker tag and the grammar file's content. `test txtar` makes one
// CompileAndParseCmd per input, all on one grammar, so without it every
// input paid a docker compile of about half a second. The key is the one
// file alone: a grammar that inherits from another file in its directory
// is compiled once per process, as it was before for a single command.
var (
	wasmCacheMu sync.Mutex
	wasmCache   = map[[sha256.Size]byte][]byte{}
)

func (vc *CompileAndParseCmd) Process() (string, error) {
	wasmBytes, err := vc.compile()
	if err != nil {
		return "", err
	}
	ctx := context.Background()
	var (
		gmr  *ohm.Grammar
		mr   *ohm.MatchResult
		root ohm.Node
	)
	if gmr, err = ohm.NewGrammar(ctx, wasmBytes); err != nil {
		return "", fmt.Errorf("creating grammar: %v", err)
	}
	defer gmr.Close()
	if mr, err = gmr.Match(vc.Source); err != nil {
		return "", fmt.Errorf("matching: %v", err)
	}
	defer mr.Close()
	if !mr.Succeeded() {
		return "", utils.NewMatchError(vc.InputName, mr)
	}
	if root, err = mr.GetCstRoot(); err != nil {
		return "", fmt.Errorf("Error getting cst root. %v", err)
	}
	var result strings.Builder
	vc.toSexprNode(root, 0, &result)
	return result.String()[1:], nil
}

// compile answers the grammar's wasm. See Compile.
func (vc *CompileAndParseCmd) compile() ([]byte, error) {
	return Compile(vc.GrammarFile, vc.KeepWasm)
}

// Compile answers the wasm of grammarFile, from wasmCache when this process
// has compiled the same grammar before, and from the ohmjs/ohm docker
// image otherwise. The lock is held over the compile, so two callers of
// one grammar run docker once. With keepWasm, it always runs docker and
// keeps the working directory.
func Compile(grammarFile string, keepWasm bool) ([]byte, error) {
	absGrammarPath, err := filepath.Abs(grammarFile)
	if err != nil {
		return nil, fmt.Errorf("resolving grammar file path: %v", err)
	}
	src, err := os.ReadFile(absGrammarPath)
	if err != nil {
		return nil, fmt.Errorf("reading grammar file: %v", err)
	}
	dockerTag := ((*ohm.Grammar)(nil)).MatchingDockerImageTags()[0]
	key := sha256.Sum256(append([]byte(dockerTag+"\x00"), src...))
	wasmCacheMu.Lock()
	defer wasmCacheMu.Unlock()
	if wasm, ok := wasmCache[key]; ok && !keepWasm {
		return wasm, nil
	}
	// create a tempdir, get the absolute path of the directory GrammarFile is in
	// run `docker run --rm -v "<basedir>:/local" -v "<tempdir>:/dst" ohmjs/ohm:<tag> compile GrammarFile`
	tempdir, err := os.MkdirTemp("", "ohm-compile-*")
	if err != nil {
		return nil, fmt.Errorf("creating tempdir: %v", err)
	}
	if !keepWasm {
		defer os.RemoveAll(tempdir)
	} else {
		fmt.Fprintf(os.Stderr, "working dir %s\n", tempdir)
	}
	basedir := filepath.Dir(absGrammarPath)
	grammarBasename := filepath.Base(absGrammarPath)
	cmd := exec.Command("docker", "run", "--rm",
		"-v", fmt.Sprintf("%s:/local", basedir),
		"-v", fmt.Sprintf("%s:/dst", tempdir),
		fmt.Sprintf("ohmjs/ohm:%s", dockerTag),
		"compile", grammarBasename,
	)
	// docker's progress goes to stderr, so stdout holds only the output
	// of the command that compiled.
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("docker compile: %v", err)
	}
	// get the basename of GrammarFile without the extension
	// read <tempdir>/<basename>.wasm
	baseNoExt := strings.TrimSuffix(grammarBasename, filepath.Ext(grammarBasename))
	wasmPath := filepath.Join(tempdir, baseNoExt+".wasm")
	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		return nil, fmt.Errorf("reading compiled wasm at %s: %v", wasmPath, err)
	}
	wasmCache[key] = wasmBytes
	return wasmBytes, nil
}

func (vc *CompileAndParseCmd) toSexprNode(node ohm.Node, depth int, result *strings.Builder) {
	kids := node.Children()
	if vc.ExcludeTerminals {
		kids = lo.Filter(kids, func(node ohm.Node, i int) bool {
			_, is := node.(ohm.TerminalNode)
			return !is
		})
	}
	if vc.SyntacticRulesOnly {
		kids = lo.Filter(kids, func(node ohm.Node, i int) bool {
			ctor := node.CtorName()
			return strings.ToUpper(ctor[:1]) == ctor[:1]
		})
	}
	if vc.ExcludeBuiltin {
		kids = lo.Filter(kids, func(node ohm.Node, i int) bool {
			ctor := node.CtorName()
			_, ex := builtins[ctor]
			return !ex
		})
	}
	ctor := node.CtorName()
	if vc.PruneLists && ctor == "_list" {
		for _, n := range kids {
			vc.toSexprNode(n, depth, result)
		}
		return
	}
	switch {
	case len(kids) != 0:
		result.WriteString("\n")
		result.WriteString(strings.Repeat("  ", depth))
		result.WriteString("(")
		result.WriteString(ctor)
		if vc.Verbose {
			s, f := node.Source()
			result.WriteString(fmt.Sprintf(" %d-%d ", s, f))
			src := node.SourceString()
			src = sexprStrEncode(src)
			if len(src) > 40 {
				src = src[:37] + "..."
			}
			result.WriteString("'")
			result.WriteString(src)
			result.WriteString("'")
		}
	case ctor == "_terminal":
		result.WriteString(" (")
		result.WriteString(ctor)
	default:
		result.WriteString(" ")
		result.WriteString(ctor)
	}

	if ctor == "_terminal" {
		addTerm(node, result)
	}
	for _, n := range kids {
		vc.toSexprNode(n, depth+1, result)
	}
	switch {
	case len(kids) != 0:
		if !vc.NoClosingNewline {
			result.WriteString("\n")
			result.WriteString(strings.Repeat("  ", depth))
		}
		result.WriteString(")")
	case ctor == "_terminal":
		result.WriteString(")")
	}
}

func addTerm(node ohm.Node, result *strings.Builder) {
	val := node.SourceString()
	switch val {
	case "\t":
		result.WriteString(" TAB")
	case "\n":
		result.WriteString(" NL")
	case "(":
		result.WriteString(" OP")
	case ")":
		result.WriteString(" CP")
	case "{":
		result.WriteString(" OC")
	case "}":
		result.WriteString(" CC")
	default:
		result.WriteString(" '")
		result.WriteString(val)
		result.WriteString("'")
	}
	result.WriteString(" ")
	s, f := node.Source()
	result.WriteString(fmt.Sprintf("%d-%d", s, f))
}

func sexprStrEncode(val string) string {
	val = strings.ReplaceAll(val, "\t", "\\t")
	val = strings.ReplaceAll(val, "\n", "\\n")
	val = strings.ReplaceAll(val, "(", "OP")
	val = strings.ReplaceAll(val, ")", "CP")
	val = strings.ReplaceAll(val, "{", "OC")
	val = strings.ReplaceAll(val, "}", "CC")
	return val
}
