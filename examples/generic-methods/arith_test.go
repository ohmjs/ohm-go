package arith

import (
	"context"
	"strings"
	"testing"

	"github.com/ohmjs/ohm-go/ohm"
	"github.com/ohmjs/ohmgo-examples/generic-methods/arith_visitor"
)

func newGrammar(t *testing.T) *ohm.Grammar {
	t.Helper()
	var (
		gmr *ohm.Grammar
		err error
	)
	if gmr, err = NewGrammar(context.Background()); err != nil {
		t.Fatalf("loading grammar: %v", err)
	}
	t.Cleanup(func() {
		gmr.Close()
	})
	return gmr
}

func TestEvaluate(t *testing.T) {
	var (
		gmr = newGrammar(t)
		env = Env{
			"x": 5,
			"y": 1,
		}
		cases = []struct {
			src  string
			want int
		}{
			{
				src:  "42",
				want: 42,
			},
			{
				src:  "1 + 2 * 3",
				want: 7,
			},
			{
				src:  "(1 + 2) * 3",
				want: 9,
			},
			{
				src:  "10 - 4 - 3",
				want: 3,
			},
			{
				src:  "8 / 2 / 2",
				want: 2,
			},
			{
				src:  "x * 2 + y",
				want: 11,
			},
			{
				src:  "((x))",
				want: 5,
			},
		}
	)
	for _, c := range cases {
		t.Run(c.src, func(t *testing.T) {
			var got, err = Evaluate(gmr, c.src, env)
			if err != nil {
				t.Fatalf("Evaluate(%q): %v", c.src, err)
			}
			if got != c.want {
				t.Fatalf("Evaluate(%q) = %d, want %d", c.src, got, c.want)
			}
		})
	}
}

func TestEvaluateErrors(t *testing.T) {
	var (
		gmr   = newGrammar(t)
		cases = []struct {
			src  string
			want string
		}{
			{
				// The divisor is printed by the printer visitor from inside
				// the evaluator, via node.AcceptPriExp[any, string].
				src:  "1 / (2 - 2)",
				want: "division by zero: (2 - 2) evaluates to 0",
			},
			{
				src:  "z + 1",
				want: `undefined identifier "z"`,
			},
			{
				src:  "1 +",
				want: "is not an arithmetic expression",
			},
		}
	)
	for _, c := range cases {
		t.Run(c.src, func(t *testing.T) {
			var _, err = Evaluate(gmr, c.src, nil)
			if err == nil {
				t.Fatalf("Evaluate(%q) succeeded, want error containing %q", c.src, c.want)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("Evaluate(%q) error = %q, want it to contain %q", c.src, err, c.want)
			}
		})
	}
}

func TestPrint(t *testing.T) {
	var (
		gmr   = newGrammar(t)
		cases = []struct {
			src  string
			want string
		}{
			{
				src:  "x",
				want: "x",
			},
			{
				src:  "1 + 2 * 3",
				want: "(1 + (2 * 3))",
			},
			{
				src:  "(1 + 2) * 3",
				want: "((1 + 2) * 3)",
			},
			{
				src:  "10 - 4 - 3",
				want: "((10 - 4) - 3)",
			},
			{
				src:  "a1 / (b2 * 3)",
				want: "(a1 / (b2 * 3))",
			},
		}
	)
	for _, c := range cases {
		t.Run(c.src, func(t *testing.T) {
			var got, err = Print(gmr, c.src)
			if err != nil {
				t.Fatalf("Print(%q): %v", c.src, err)
			}
			if got != c.want {
				t.Fatalf("Print(%q) = %q, want %q", c.src, got, c.want)
			}
		})
	}
}

// TestEvaluateAndPrint drives one node through both visitors, which is the
// point of generic methods: the node type is not tied to a (P, R) pair.
func TestEvaluateAndPrint(t *testing.T) {
	var (
		gmr = newGrammar(t)
		env = Env{
			"x": 5,
			"y": 1,
		}
		value, text, err = EvaluateAndPrint(gmr, "x * (y + 2)", env)
	)
	if err != nil {
		t.Fatalf("EvaluateAndPrint: %v", err)
	}
	if value != 15 {
		t.Errorf("value = %d, want 15", value)
	}
	if text != "(x * (y + 2))" {
		t.Errorf("text = %q, want %q", text, "(x * (y + 2))")
	}
}

// TestMakeNodeFromRootIsAnAcceptorFunc pins the generic-methods shape of the
// entry point: it returns a method value, not an interface.
func TestMakeNodeFromRootIsAnAcceptorFunc(t *testing.T) {
	var (
		gmr = newGrammar(t)
		err = withRoot(gmr, "2 * 21", func(root ohm.Node) error {
			var (
				accept   ohm.AcceptorFunc[Env, int] = arith_visitor.MakeNodeFromRoot[Env, int](root)
				got, err                            = accept(root, &evaluator{}, nil)
			)
			if err != nil {
				return err
			}
			if got != 42 {
				t.Errorf("got %d, want 42", got)
			}
			return nil
		})
	)
	if err != nil {
		t.Fatal(err)
	}
}
