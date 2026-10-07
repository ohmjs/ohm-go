# generic-methods: one node type, many visitors

An arithmetic evaluator and pretty-printer built on a visitor package that
`ohm-cli generate go --generic-methods` produced. It needs Go 1.27, which
added methods with type parameters.

| file | role |
|---|---|
| `arith.ohm` | the grammar (hand-written) |
| `arith.wasm` | parser compiled from `arith.ohm` by the `ohmjs/ohm` docker image |
| `arith_visitor/{types,interfaces,accepts}.go` | generated from `arith.ohm`, do not edit |
| `arith.go` | the two visitors and the public `Evaluate`, `Print` and `EvaluateAndPrint` functions |
| `arith_test.go` | tests |
| `generate.go` | `go generate` directives that rebuild the wasm and the visitor together |

## What generic methods change

With the default flavour every generated node struct carries the payload
and result types, so a node value is bound to one visitor signature:

```go
type AddExpPlus[P, R any] struct { ... }

func (node *AddExpPlus[P, R]) Accept(this ohm.Node, visitor any, payload P) (result R, err error)
```

With `--generic-methods` the struct is plain and the type parameters move
onto the methods:

```go
type AddExpPlus struct { ... }

func (node *AddExpPlus) Accept[P, R any](this ohm.Node, visitor any, payload P) (result R, err error)
func (node *AddExpPlus) AcceptAddExp[P, R any](visitor any, payload P) (result R, err error)
```

`R` cannot be inferred from the arguments, so calls instantiate explicitly:

```go
l, err = node.AcceptAddExp[Env, int](e, env)
```

## What the example shows

- `EvaluateAndPrint` builds one `*arith_visitor.Exp` and calls
  `exp.Accept[Env, int]` with the evaluator, then `exp.Accept[any, string]`
  with the printer. The same node value serves both.
- `evaluator.VisitMulExpDivide` is a method of a `(P=Env, R=int)` visitor,
  yet on a zero divisor it calls `node.AcceptPriExp[any, string](&printer{}, nil)`
  to render the divisor for the error message. With struct generics this
  needs an `unsafe.Pointer` cast between two instantiations of the node type.
- The printer passes instantiated method values around:
  `p.binary(node.AcceptAddExp[any, string], "+", node.AcceptMulExp[any, string])`.
- `MakeNodeFromRoot[P, R]` returns an `ohm.AcceptorFunc[P, R]`, the `Accept`
  method value of the root node, so the entry point is
  `MakeNodeFromRoot[Env, int](root)(root, &evaluator{}, env)`.

Run the tests from the `examples` module:

```sh
go test ./generic-methods/
```

Regenerate after editing the grammar (needs docker):

```sh
go generate ./generic-methods/
```

The generated visitor shapes are documented in
[ohm-cli/docs/visitors.md](../../ohm-cli/docs/visitors.md).
