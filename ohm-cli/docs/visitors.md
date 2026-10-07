# Walking an Ohm parse tree in Go with generated visitors

`ohm-cli generate go` reads an `.ohm` grammar and writes a Go package that
lets you walk the concrete syntax tree (CST) produced by the
`github.com/ohmjs/ohm-go/ohm` runtime with typed visitor methods. For every
rule in the grammar you get a struct whose fields are the rule's children,
a set of `Visit<Rule>` interfaces to implement, and an `Accept` method that
dispatches to whichever of those your visitor provides and otherwise walks
the children for you.

A visitor is any Go value. You implement `Visit<Rule>` methods only for the
rules you care about, pick a payload type `P` that flows down the tree and a
result type `R` that flows back up, and start the walk from the match root.

This document covers:

1. [The pipeline](#1-the-pipeline): grammar to wasm, grammar to Go, and why both must be regenerated together.
2. [How rules become Go types](#2-how-rules-become-go-types).
3. [The eight visitor interfaces and `Accept`](#3-the-eight-visitor-interfaces-and-accept).
4. [Runtime helper interfaces](#4-runtime-helper-interfaces) for terminals and nodes with no rule struct.
5. [Writing a visitor](#5-writing-a-visitor): the entry point, choosing `R`, building children inline.
6. [The two generics flavours](#6-the-two-generics-flavours): struct generics and `--generic-methods`.
7. [CLI usage](#7-cli-usage).

The snippets use the arithmetic grammar in
[`examples/generic-methods/arith.ohm`](../../examples/generic-methods/arith.ohm):

```
Arithmetic {
  Exp     = AddExp
  AddExp  = AddExp "+" MulExp  -- plus
          | AddExp "-" MulExp  -- minus
          | MulExp
  MulExp  = MulExp "*" PriExp  -- times
          | MulExp "/" PriExp  -- divide
          | PriExp
  PriExp  = "(" Exp ")"        -- paren
          | ident
          | number
  ident  (an identifier) = letter alnum*
  number  (a number)     = digit+
}
```

## 1. The pipeline

Two artefacts are derived from the same grammar file:

```
arith.ohm --(docker ohmjs/ohm compile)--> arith.wasm       parser, loaded by ohm.NewGrammar
arith.ohm --(ohm-cli generate go)-------> arith_visitor/   types.go, interfaces.go, accepts.go
```

The parser is the Ohm compiler's wasm output. `ohm-cli generate command`
prints the docker invocation, pinned to the image tag that matches the
runtime version built into the CLI (keep that in step with the `ohm`
version in your own `go.mod`):

```sh
docker run --rm -v "$PWD":/local ohmjs/ohm:18.0.0-beta.15 compile arith.ohm
```

`ohm.NewGrammar` reads the compiler version recorded in the wasm and refuses
to load a module whose tag is not in `(*ohm.Grammar).MatchingDockerImageTags()`.
Matching gives you an `ohm.Node`:

```go
var (
	gmr  *ohm.Grammar
	mr   *ohm.MatchResult
	root ohm.Node
	err  error
)
if gmr, err = ohm.NewGrammar(ctx, wasmBytes); err != nil {
	return err
}
defer gmr.Close()
if mr, err = gmr.Match(input); err != nil || !mr.Succeeded() {
	return fmt.Errorf("match failed: %v", err)
}
defer mr.Close()
if root, err = mr.GetCstRoot(); err != nil {
	return err
}
```

Every CST node implements `ohm.Node` (`CtorName()`, `Children()`,
`SourceString()`, `StartIdx()`, `MatchLength()`). The runtime refines it
into `ohm.RuleNode`, `ohm.TerminalNode`, `ohm.ListNode` (`*` and `+`),
`ohm.OptNode` (`?`, with `IsPresent()`), `ohm.BHorNode` (the built-in
`ListOf` family, with `Elems()` and `Seps()`) and `ohm.SeqNode`.
`CtorName()` is the rule name as Ohm spells it: `"AddExp_plus"` for a case,
`"ident"` for a lexical rule.

The generated Go is the other half. It hard-codes the CST shape each rule
produces: `kids[1].(ohm.TerminalNode)`, `case "AddExp_plus":` and so on.
Those indices and names come from the grammar text, not from the wasm, so
after any edit to the `.ohm` you must rerun **both** the docker compile and
`ohm-cli generate go`. If only one is rerun the mismatch does not show up at
compile time; it shows up as a failed type assertion or an `ohm.AssertName`
panic during the walk. Each generated file starts with a comment holding
the exact `ohm-cli` command that produced it, so the regeneration command
is never far away.

## 2. How rules become Go types

`types.go` holds one struct per rule. Three shapes of rule are treated
differently.

### 2.1 Bare rules: one struct, one field per top-level term

```go
// PriExp_paren
// -- rule --
// "(" Exp ")"
// ----
type PriExpParen struct {
	Term1 ohm.TerminalNode
	Exp   ohm.RuleNode
	Term2 ohm.TerminalNode
}
```

Fields appear in body order, one per top-level term, typed by the kind of
CST node that term produces:

| In the rule body | Field name | Go type |
|---|---|---|
| `"("` literal | `Term`, or `Term1`, `Term2`, ... when the body has several | `ohm.TerminalNode` |
| `Exp` rule application | the rule name capitalised: `Exp`, `Ident`; numbered `Number1`, `Number2` when repeated | `ohm.RuleNode` |
| `x*`, `x+` | the name `x` would get (`Rule` for `Rule*`), `Alt` for a `(...)` group | `ohm.ListNode` |
| `x?` | the name `x` would get (`Formals` for `Formals?`, `Term` for `"\|"?`) | `ohm.OptNode` |
| `(...)` group with no suffix | `Alt`, numbered when repeated | `ohm.Node` |
| `ListOf<x, sep>`, `NonemptyListOf<x, sep>` | the built-in's name: `ListOf`, `NonemptyListOf` | `ohm.BHorNode` |
| `~x`, `&x` lookahead | no field; predicates produce no CST child | |
| unnamed alternatives whose branches disagree at a position | `Arg` | `ohm.Node` |
| unnamed alternatives that are each a single rule application | `Rule` | `ohm.RuleNode` |

A lexical rule (lower-case name) keeps its name as a *field* in its parents
(`Ident ohm.RuleNode`) but its own *type* gets a `Lex` prefix so that it is
exported: `ident` becomes `LexIdent`, `number` becomes `LexNumber`, and the
visit method is `VisitLexIdent`. At runtime `CtorName()` still returns
`"ident"`.

Iterated groups are flattened. In `Tree = "[" Node ("," Node)* ","? "]"`
the `("," Node)*` term is a single `Alt ohm.ListNode` whose children
alternate `","` terminal, `Node`, `","`, `Node`; filter them by `CtorName()`
rather than by index.

### 2.2 Rules with named cases

```
AddExp = AddExp "+" MulExp  -- plus
       | AddExp "-" MulExp  -- minus
       | MulExp
```

A rule with `-- name` cases becomes a holder struct with a single `Node`
field plus one "virtual" struct per case named `<Rule><Case>` with the case
title-cased and underscores kept (`AddExpPlus`, `AtomDigit_as_name`,
`LexEscapeCharUnicodeCodePoint`):

```go
type AddExp struct {
	Node ohm.Node
}

type AddExpPlus struct {
	AddExp ohm.RuleNode
	Term   ohm.TerminalNode
	MulExp ohm.RuleNode
}
```

The holder's `DefaultAccept` switches on `CtorName()`, builds the matching
case struct from the children and calls its `Accept`. A trailing unnamed
alternative (`| MulExp` above) is the `default:` branch and is forwarded
with the same `P` and `R`:

```go
func (node *AddExp) DefaultAccept[P, R any](visitor any, payload P) (result R, err error) {
	switch node.Node.CtorName() {
	case "AddExp":
		return (&AddExp{
			Node: node.Node.Children()[0].(ohm.RuleNode),
		}).DefaultAccept[P, R](visitor, payload)
	case "AddExp_plus":
		kids := node.Node.Children()
		return (&AddExpPlus{
			AddExp: kids[0].(ohm.RuleNode),
			Term:   kids[1].(ohm.TerminalNode),
			MulExp: kids[2].(ohm.RuleNode),
		}).Accept[P, R](node.Node, visitor, payload)
	case "AddExp_minus":
		// ... same shape, AddExpMinus
	default: // bare case after named ones
		kids := node.Node.Children()
		return (&MulExp{
			Node: kids[0],
		}).Accept[P, R](node.Node, visitor, payload)
	}
}
```

The `Node` field may hold either the `AddExp` rule node itself or its single
child (the case node); the `case "AddExp":` branch unwraps the former. If
there is no unnamed alternative the `default:` panics with `"unexpected "`
plus the constructor name.

### 2.3 Alternatives of single rules

```
PriExp = "(" Exp ")"  -- paren
       | ident
       | number
```

When every unnamed alternative is a lone rule application, the generator
unifies them into one rule-typed position instead of falling back to
`ohm.Node`. For a rule with no cases at all (`Value = ident | number`) that
is a bare struct with a single `Rule ohm.RuleNode` field and an
`AcceptRule` that switches on `node.Rule.CtorName()`. For `PriExp`, which
mixes a named case with bare rules, it is the `default:` branch of the
holder's `DefaultAccept`:

```go
	default: // bare case after named ones
		switch node.Node.CtorName() {
		case "ident":
			kids := node.Node.Children()
			result, err = (&LexIdent{
				Letter: kids[0].(ohm.RuleNode),
				Alnum:  kids[1].(ohm.ListNode),
			}).Accept[P, R](node.Node, visitor, payload)
			return
		case "number":
			// ... LexNumber
		default:
			panic("unexpected " + node.Node.CtorName())
		}
```

Where the branches do not line up like that (`Mixed = "a" ident | "b" number`)
the disagreeing position is typed `Arg ohm.Node` and only the runtime
helpers of section 4 see it.

The data model behind all of this is `ohm-cli/ruleast.adl` (`BareRuleNode`,
`VirtRuleNode`, `CasesRuleNode`, and the `ArgNode` union); the unification
is `UnifyBranches2Args` in `ohm-cli/ruleast/utils.go`; the templates are in
`ohm-cli/ruleast/templates/`. `ohm-cli test rule_ast @grammar.ohm` prints
the model the generator derived for a grammar, which is the quickest way
to see why a field got the name and type it did.

## 3. The eight visitor interfaces and `Accept`

`interfaces.go` declares eight interfaces per struct `X`. Each has one
method, `VisitX`, and the letters in the name say what that method takes
and returns: `P` a payload argument, `R` a result, `E` an error.

| Interface | Method |
|---|---|
| `Visitor_X[P, R any]` | `VisitX(node *X)` |
| `VisitorE_X[P, R any]` | `VisitX(node *X) error` |
| `VisitorP_X[P, R any]` | `VisitX(node *X, payload P)` |
| `VisitorPE_X[P, R any]` | `VisitX(node *X, payload P) error` |
| `VisitorR_X[P, R any]` | `VisitX(node *X) (result R)` |
| `VisitorRE_X[P, R any]` | `VisitX(node *X) (result R, err error)` |
| `VisitorPR_X[P, R any]` | `VisitX(node *X, payload P) (result R)` |
| `VisitorPRE_X[P, R any]` | `VisitX(node *X, payload P) (result R, err error)` |

(In the default flavour the node parameter is `*X[P, R]`; see section 6.)

`accepts.go` gives each struct an `Accept`:

```go
func (node *AddExpPlus) Accept[P, R any](this ohm.Node, visitor any, payload P) (result R, err error) {
	ohm.AssertName(this, "AddExp_plus")
	if v, ok := visitor.(Visitor_AddExpPlus[P, R]); ok {
		v.VisitAddExpPlus(node)
		return
	}
	if v, ok := visitor.(VisitorE_AddExpPlus[P, R]); ok {
		err = v.VisitAddExpPlus(node)
		return
	}
	// ... P, PE, R, RE, PR, PRE in that order
	ohm.TypeCheckMethod[P, R](visitor, "AddExpPlus")
	return node.DefaultAccept[P, R](visitor, payload)
}
```

- `visitor` is `any`. A visitor needs no base type and no registration;
  it is matched purely by type assertion against the eight interfaces, in
  the table order, and the first `VisitAddExpPlus` that fits is called.
  Since a Go type can have only one method of that name, at most one fits.
- If none fits, `DefaultAccept` runs. For a bare or case struct it calls
  `Accept<Field>` for every field in order and returns the **last** child's
  result; for a holder struct it does the `CtorName()` switch of section
  2.2. "Last child" matters: `PriExpParen`'s default walk returns the
  result of `Term2`, which is the zero `R`, so a visitor that wants the
  inner value must implement `VisitPriExpParen` and return
  `node.AcceptExp(...)`.
- The payload `P` is passed unchanged from `Accept` to `DefaultAccept` to
  each `Accept<Field>` and on into the child's `Accept`; it flows down the
  call stack. `R` and `error` are returned at every level; they flow up.
  Both are carried in Go's own call stack, so a visitor holds no state
  unless it wants to.
- `ohm.AssertName` panics if `this` is not the node the struct was built
  from, which catches passing the wrong node to `Accept`.

### `ohm.TypeCheckMethod`

The most common mistake while writing a visitor is to declare
`VisitAddExpPlus` with a `P` or `R` that differs from the ones the caller
instantiated. No interface then matches and the method would be silently
skipped. `ohm.TypeCheckMethod[P, R](visitor, "AddExpPlus")` runs just
before `DefaultAccept` and uses reflection (`MethodByName`) to look for a
method named `VisitAddExpPlus`. If one exists it panics with the expected
and received signatures and the likely call site:

```
VisitAddExpPlus. Found method by name match, but incompatibles types.
  expected func(<visitor>, AddExpPlus[main.Env,int], main.Env) int
  received func(*main.evaluator, *arith_visitor.AddExpPlus, main.Env) (string, error)
  For the likely call sight see:
    /path/to/eval.go:42
```

Successful checks are cached per (visitor type, rule, `P`, `R`), so the
reflection cost is paid once per combination. A mismatch is never cached
and panics every time. To opt out, implement the marker interface
`ohm.SkipCheckName` (a method `SkipCheckName()`) on the visitor; a `nil`
visitor is also skipped. The `--skip-type-check-method` (`-s`) flag of
`ohm-cli generate go` exists to omit the call entirely, because
`MethodByName` keeps every `Visit*` method alive through dead-code
elimination; check your generated `accepts.go`, since at the time of
writing the templates still emitted the call with the flag set.

## 4. Runtime helper interfaces

Terminals, `ohm.Node`-typed fields, list elements that are not rule
applications, and rules the generator does not know (Ohm built-ins such as
`letter`, `digit`, `any`, or rules inherited from a super grammar) have no
rule struct and no `Visit` method. For these the generated `Accept<Field>`
probes three small interfaces from the `ohm` package instead:

```go
type TerminalVisitor interface {
	Terminal(node TerminalNode)
}
type NodeVisitor interface {
	NodeVisit(node Node)
}
type SpaceBeforeVisitor interface {
	SpaceBefore(spaces string)
}
```

- A terminal field is reported with `Terminal(node.Term)` if the visitor is
  a `TerminalVisitor`, and otherwise ignored.
- A node with no struct is handed whole to `NodeVisit` if the visitor is a
  `NodeVisitor`. Otherwise, if the visitor implements `TerminalVisitor` or
  `SpaceBeforeVisitor`, the generated code calls
  `ohm.WalkCstCallTermAndSpace(n, fnTerm, fnSpaceBefore)`, which descends
  the subtree calling `fnTerm` for every terminal and `fnSpaceBefore` with
  the text of the implicit whitespace Ohm skipped before each child of a
  syntactic rule.

These helpers return no `R`; the result at that level stays zero. They are
what you use for lossless passes over the source. `ohm-cli/es5/roundtrip.go`
re-prints an ES5 program with a visitor that has no `Visit*` methods at all:

```go
var (
	_ ohm.TerminalVisitor    = &es5roundtrip{}
	_ ohm.NodeVisitor        = &es5roundtrip{}
	_ ohm.SpaceBeforeVisitor = &es5roundtrip{}
)

func (cm *es5roundtrip) Terminal(node ohm.TerminalNode) { cm.sb.WriteString(node.SourceString()) }
func (cm *es5roundtrip) NodeVisit(node ohm.Node)         { cm.sb.WriteString(node.SourceString()) }
func (cm *es5roundtrip) SpaceBefore(spaces string)       { cm.sb.WriteString(spaces) }
```

With `P` and `R` both `any`, every `Accept` falls through to `DefaultAccept`
and the three callbacks see the whole tree in source order.

## 5. Writing a visitor

### The entry point

`MakeNodeFromRoot[P, R](root)` builds the struct for the grammar's first
rule from a CST node of that rule. It is normally called on the match root,
but it works on any node of the start rule, which is handy when the start
rule also appears nested inside the tree:

```go
type Env map[string]int

func Eval(root ohm.Node, env Env) (int, error) {
	return arith_visitor.MakeNodeFromRoot[Env, int](root)(root, &evaluator{}, env)
}
```

(That is the `--generic-methods` form; in the default flavour it reads
`MakeNodeFromRoot[Env, int](root).Accept(root, &evaluator{}, env)`.)

### One visitor, one `R` per rule

The visitor is usually an empty struct. Each `Visit` method fixes `R` to
whatever its rule should produce, and because dispatch is by interface the
same struct satisfies `VisitorPRE_AddExpPlus[Env, int]` and, say,
`VisitorRE_Args[Env, []int]` at the same time. Write the choices down as a
compile-time block; it doubles as the table of `R` per rule:

```go
type evaluator struct{}

var (
	_ arith_visitor.VisitorPRE_AddExpPlus[Env, int]  = (*evaluator)(nil)
	_ arith_visitor.VisitorPRE_PriExpParen[Env, int] = (*evaluator)(nil)
	_ arith_visitor.VisitorPRE_LexIdent[Env, int]    = (*evaluator)(nil)
	_ arith_visitor.VisitorRE_LexNumber[Env, int]    = (*evaluator)(nil)
	// ... one line per Visit method
)
```

Use the `P` variants when a parent has context to push down (the
environment here); use the `E` variants and return errors rather than
panicking, since `Accept` carries them up for you.

### Let `DefaultAccept` do the pass-through dispatch

`Exp`, `AddExp`, `MulExp` and `PriExp` need no `Visit` methods: their
generated `DefaultAccept` forwards to the case struct or the bare
alternative with the same `P` and `R`, so as long as every branch of an
expression produces an `int` the holders can be skipped. When a child
should produce the same `R` as its parent, call the generated
`Accept<Field>` helper:

```go
func (e *evaluator) VisitAddExpPlus(node *arith_visitor.AddExpPlus, env Env) (int, error) {
	var (
		lhs, rhs int
		err      error
	)
	if lhs, err = node.AcceptAddExp[Env, int](e, env); err != nil {
		return 0, err
	}
	if rhs, err = node.AcceptMulExp[Env, int](e, env); err != nil {
		return 0, err
	}
	return lhs + rhs, nil
}

func (e *evaluator) VisitPriExpParen(node *arith_visitor.PriExpParen, env Env) (int, error) {
	return node.AcceptExp[Env, int](e, env)
}
```

### When the child's `R` differs, build the child yourself

`DefaultAccept` and the `Accept<Field>` helpers always forward the parent's
`P` and `R`. Whenever a child must produce something else, construct the
child's struct at the call site, copying the field list from that child's
generated `Accept<Field>`, and call `Accept` with the child's own types.
Suppose the grammar also had `Args = "<" ListOf<Exp, ","> ">"` and you
want `[]int` for it:

```go
func (e *evaluator) VisitArgs(node *arith_visitor.Args, env Env) ([]int, error) {
	var (
		out []int
		v   int
		err error
	)
	for _, elem := range node.ListOf.Elems() {
		if v, err = (&arith_visitor.Exp{
			AddExp: elem.Children()[0].(ohm.RuleNode),
		}).Accept[Env, int](elem, e, env); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}
```

Build the literal inline rather than in a helper: the construction is the
first thing a reader wants to see, and when the grammar changes the
generated field list changes and these call sites stop compiling, which is
the signal you want. For a holder struct the literal is just
`&AddExp{Node: n.Children()[0]}` (or `Node: n` itself).

The same technique covers unnamed alternatives whose branches need
different results. If `Part = Atom | number` maps to `*Atom` and `uint64`,
the generated `Part.AcceptRule` cannot be used because it forwards
`Part`'s `R` into both; implement `VisitPart`, switch on
`node.Rule.CtorName()` yourself and build each branch with its own `R`.

### Reading the CST

- `SourceString()` on any node gives the matched text; for leaves such as
  `ident` that is usually all you need.
- `OptNode.IsPresent()` or `len(Children()) > 0` tests an optional.
- `BHorNode.Elems()` returns the elements of a `ListOf` without separators.
- Iterated groups are flattened (section 2.1); filter by `CtorName()`.
- A `TypeCheckMethod` panic means a `P`/`R` mismatch in your code, not a bad
  input; input problems surface as `error` values.

A complete worked example of this style, building a typed AST from a small
JSON-like grammar through ADL-generated types, is the `samples/goohm`
module and `docs/grammar2ast.md` in the sheafdb repository; its visitor
`ast/build.go` follows every rule in this section.

## 6. The two generics flavours

`P` and `R` have to live somewhere. The generator offers two placements.

### Side by side

Default, "struct generics" (`ohm-cli generate go`):

```go
type AddExpPlus[P, R any] struct {
	AddExp ohm.RuleNode
	Term   ohm.TerminalNode
	MulExp ohm.RuleNode
}

type VisitorPRE_AddExpPlus[P, R any] interface {
	VisitAddExpPlus(node *AddExpPlus[P, R], payload P) (result R, err error)
}

func (node *AddExpPlus[P, R]) Accept(this ohm.Node, visitor any, payload P) (result R, err error)
func (node *AddExpPlus[P, R]) DefaultAccept(visitor any, payload P) (result R, err error)
func (node *AddExpPlus[P, R]) AcceptAddExp(visitor any, payload P) (result R, err error)

func MakeNodeFromRoot[P, R any](root ohm.Node) ohm.Acceptor[P, R] {
	kids := root.Children()
	return &Exp[P, R]{
		AddExp: kids[0].(ohm.RuleNode),
	}
}
```

Generic methods (`ohm-cli generate go --generic-methods`, requires Go 1.27,
which added methods with their own type parameters):

```go
type AddExpPlus struct {
	AddExp ohm.RuleNode
	Term   ohm.TerminalNode
	MulExp ohm.RuleNode
}

type VisitorPRE_AddExpPlus[P, R any] interface {
	VisitAddExpPlus(node *AddExpPlus, payload P) (result R, err error)
}

func (node *AddExpPlus) Accept[P, R any](this ohm.Node, visitor any, payload P) (result R, err error)
func (node *AddExpPlus) DefaultAccept[P, R any](visitor any, payload P) (result R, err error)
func (node *AddExpPlus) AcceptAddExp[P, R any](visitor any, payload P) (result R, err error)

func MakeNodeFromRoot[P, R any](root ohm.Node) ohm.AcceptorFunc[P, R] {
	kids := root.Children()
	return (&Exp{
		AddExp: kids[0].(ohm.RuleNode),
	}).Accept
}
```

The node structs lose their type parameters; `Accept`, `DefaultAccept` and
every `Accept<Field>` gain them. The interfaces keep `[P, R any]`, because
`P` and `R` still appear in the method signature, but they take a plain
`*AddExpPlus`. `MakeNodeFromRoot` can no longer return a node value typed
by `P` and `R`, so it returns the method value `(&Exp{...}).Accept` as an
`ohm.AcceptorFunc[P, R]`.

The call sites differ accordingly:

| | struct generics | generic methods |
|---|---|---|
| start the walk | `MakeNodeFromRoot[Env, int](root).Accept(root, v, env)` | `MakeNodeFromRoot[Env, int](root)(root, v, env)` |
| visit method | `VisitAddExpPlus(node *AddExpPlus[Env, int], env Env)` | `VisitAddExpPlus(node *AddExpPlus, env Env)` |
| same-`R` child | `node.AcceptMulExp(v, env)` | `node.AcceptMulExp[Env, int](v, env)` |
| child with its own `R` | `(&Exp[Env, int]{...}).Accept(n, v, env)` | `(&Exp{...}).Accept[Env, int](n, v, env)` |

In the generic-methods flavour the type arguments must be written at every
instantiation. Go could infer `P` from the `payload` argument, but `R`
appears only in the results, and type arguments can only be omitted from
the end of the list, so both are spelled out.

### Why choose generic methods

With struct generics a node value is tied to one `(P, R)` for life. A
`*AddExpPlus[Env, int]` can only be accepted by visitors that use `Env` and
`int`; to run a second visitor with different types over the same subtree
you either rebuild the struct literal with other type arguments or cast,
which is the `unsafe.Pointer` trick the `TypeCheckMethod` panic message
mentions:

```go
((*RuleDefine[any, string])(unsafe.Pointer(node))).AcceptRuleDescr(c, payload)
```

With generic methods a node is just data and `(P, R)` is chosen per call,
so the same node can be handed to visitors with unrelated payload and
result types, and the cast goes away. In the arithmetic example an
evaluator (`Env` in, `int` out) and a printer (`any` in, `string` out)
share one set of node types, and the evaluator can use the printer on the
subtree it is holding when it reports an error:

```go
// acceptFn is the shape of an instantiated Accept<Arg>[any, string] method
// value, which is what the printer's binary helper takes.
type acceptFn = func(visitor any, payload any) (string, error)

type printer struct{}

func (p *printer) VisitAddExpPlus(node *arith_visitor.AddExpPlus) (string, error) {
	return p.binary(node.AcceptAddExp[any, string], "+", node.AcceptMulExp[any, string])
}

func (p *printer) binary(left acceptFn, op string, right acceptFn) (string, error) {
	var (
		l, r string
		err  error
	)
	if l, err = left(p, nil); err != nil {
		return "", err
	}
	if r, err = right(p, nil); err != nil {
		return "", err
	}
	return "(" + l + " " + op + " " + r + ")", nil
}

func (e *evaluator) VisitMulExpDivide(node *arith_visitor.MulExpDivide, env Env) (int, error) {
	var (
		l, r int
		text string
		err  error
	)
	if l, err = node.AcceptMulExp[Env, int](e, env); err != nil {
		return 0, err
	}
	if r, err = node.AcceptPriExp[Env, int](e, env); err != nil {
		return 0, err
	}
	if r == 0 {
		// The same node, visited again with the printer's type arguments.
		// Nothing is cast or rebuilt; only the instantiation differs.
		if text, err = node.AcceptPriExp[any, string](&printer{}, nil); err != nil {
			return 0, err
		}
		return 0, fmt.Errorf("division by zero: %s evaluates to 0", text)
	}
	return l / r, nil
}
```

The printer also shows that an instantiated generic method such as
`node.AcceptAddExp[any, string]` is an ordinary method value and can be
passed around as a `func`.

Under struct generics the second `AcceptPriExp` call is impossible without
the cast, because `node` is a `*MulExpDivide[Env, int]`.

The trade-off is verbosity: `[P, R]` at each instantiation site, and a
`go 1.27` (or later) directive in the consuming module's `go.mod`. The
generated code is otherwise the same, and a visitor written for one
flavour converts to the other mechanically: drop or add the type
arguments on the node parameter types and on the `Accept*` calls. The
sheafdb commit `4de9438` ("manually converted Tree to use generic methods")
shows such a conversion for a single rule.
[`examples/generic-methods/`](../../examples/generic-methods/) in this
repository is the complete arithmetic example: the grammar, the generated
`arith_visitor` package, the evaluator and printer in `arith.go` (including
`EvaluateAndPrint`, which runs both visitors over one `*Exp` value), and
tests. Run it with `go test ./generic-methods/` from the `examples` module.

## 7. CLI usage

```sh
ohm-cli generate go [options] @grammar.ohm
```

The grammar argument is the grammar text; an `@` prefix reads it from a
file. Three files are written: `types.go`, `interfaces.go` and
`accepts.go`.

| Flag | Meaning |
|---|---|
| `-P`, `--go-type-package pkg` | Go package name of the generated code. Default: the grammar name lower-cased. |
| `-o`, `--output-dir dir` | Directory to write into. Default: the package name. |
| `-f`, `--file-prefix pfx` | Prefix for the three file names, for example `ohm_` to give `ohm_types.go`; useful when generated and hand-written files share a package. |
| `-g`, `--grammar-name G` | Which grammar to generate for. Required when the `.ohm` file defines more than one. |
| `-s`, `--skip-type-check-method` | Omit the `ohm.TypeCheckMethod` call (section 3). |
| `-m`, `--generic-methods` | Generate the generic-methods flavour (section 6). Requires Go 1.27. |
| `--go-runtime-import`, `--go-runtime-package` | Import path and package identifier of the runtime, if you vendor or fork `github.com/ohmjs/ohm-go/ohm`. |

Example, as used for the ohm-cli's own grammar and for the example:

```sh
ohm-cli generate go -P ruleast -f ohm_ @../ohm-repo/packages/ohm-js/src/ohm-grammar.ohm
ohm-cli generate go -P arith_visitor --generic-methods @arith.ohm
```

To regenerate a single file, or to inspect output on stdout without
writing anything, use the `parts` subcommands:

```sh
ohm-cli generate parts go_types      [-P pkg] [-o file] @grammar.ohm
ohm-cli generate parts go_interfaces [-P pkg] [-o file] @grammar.ohm
ohm-cli generate parts go_accepts    [-P pkg] [-o file] @grammar.ohm
```

`-o` defaults to `-` (stdout). These take the same `--grammar-name`,
`--skip-type-check-method` and `--generic-methods` options, plus
`-e`/`--exclude-cli` to leave the command-line comment out of the header.
In `parts` the short flag `-g` means `--no-generics` (strip `[P, R any]`
from `go_types` only, for inspection), not `--grammar-name`.

Two related commands complete the toolchain. `ohm-cli generate command
[--format command|go_generate|script] grammar.ohm` prints the docker
command that compiles the grammar to wasm; `--format go_generate` gives a
`//go:generate` line to keep next to the generated package so both halves
of the pipeline are regenerated by `go generate ./...`. `ohm-cli test
rule_ast @grammar.ohm` prints the rule model the generator derived
(section 2), which is where to look when a field name or type is not what
you expected.
