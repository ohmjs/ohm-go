package tests

import (
	"strings"
	"testing"

	goohm "github.com/ohmjs/ohm-go/ohm"
)

// mismatchVisitor has a method named VisitFoo, but with a signature that
// matches none of the generated Visitor*_Foo interfaces. In the real Accept
// dispatch this is exactly the case TypeCheckMethod exists to catch: a Visit*
// method that would otherwise be silently skipped because its signature is
// wrong.
type mismatchVisitor struct{}

func (mismatchVisitor) VisitFoo() {}

// skippedVisitor also has a wrongly-typed VisitFoo, but opts out of the check
// via the SkipCheckName marker interface.
type skippedVisitor struct{}

func (skippedVisitor) VisitFoo()      {}
func (skippedVisitor) SkipCheckName() {}

// cleanVisitor has no Visit method matching the queried node type, which is
// the normal case for a node the visitor does not handle.
type cleanVisitor struct{}

func (cleanVisitor) VisitBar() {}

var _ goohm.SkipCheckName = skippedVisitor{}

// recoverPanic runs fn and returns the recovered panic value, or nil if fn
// returned normally.
func recoverPanic(fn func()) (recovered any) {
	defer func() { recovered = recover() }()
	fn()
	return nil
}

func TestTypeCheckMethod_PanicsOnSignatureMismatch(t *testing.T) {
	got := recoverPanic(func() {
		goohm.TypeCheckMethod[any, any](mismatchVisitor{}, "Foo")
	})
	if got == nil {
		t.Fatal("TypeCheckMethod did not panic on a mismatched VisitFoo method")
	}
	if msg, ok := got.(string); !ok || !strings.Contains(msg, "VisitFoo") {
		t.Fatalf("panic message %q does not mention the offending method VisitFoo", got)
	}
}

// TestTypeCheckMethod_StillRunsAfterMemoization guards the memoization added to
// TypeCheckMethod: a genuine mismatch is never cached (only successful checks
// are), so the panic must fire on every call, not just the first.
func TestTypeCheckMethod_StillRunsAfterMemoization(t *testing.T) {
	for i := 0; i < 3; i++ {
		got := recoverPanic(func() {
			goohm.TypeCheckMethod[any, any](mismatchVisitor{}, "Foo")
		})
		if got == nil {
			t.Fatalf("call %d: TypeCheckMethod did not panic; memoization is suppressing the check", i+1)
		}
	}
}

func TestTypeCheckMethod_SkipCheckNameOptsOut(t *testing.T) {
	got := recoverPanic(func() {
		goohm.TypeCheckMethod[any, any](skippedVisitor{}, "Foo")
	})
	if got != nil {
		t.Fatalf("TypeCheckMethod panicked despite SkipCheckName: %v", got)
	}
}

func TestTypeCheckMethod_NoMatchingMethod(t *testing.T) {
	// Called twice to also exercise the memoized success path.
	for i := 0; i < 2; i++ {
		got := recoverPanic(func() {
			goohm.TypeCheckMethod[any, any](cleanVisitor{}, "Foo")
		})
		if got != nil {
			t.Fatalf("call %d: TypeCheckMethod panicked for a visitor without VisitFoo: %v", i+1, got)
		}
	}
}
