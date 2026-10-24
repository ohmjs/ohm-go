# Ohm Go runtime

A Go implementation of the `Grammar` and `MatchResult` classes from the [ohm-js](https://www.npmjs.com/package/ohm-js) package, for use with Ohm grammars compiled to WebAssembly.

## Usage

```go
// Instantiate a grammar from compiled Wasm bytes
wasmBytes, _ := os.ReadFile("path/to/grammar.wasm")
g, err := NewGrammar(ctx, wasmBytes)
defer g.Close()

// Match input against the grammar (optional second arg is start rule)
result, err := g.Match("var x = 1;")
defer result.Close()

if result.Succeeded() {
    cstRoot, err := result.GetCstRoot()
    // ...walk the tree
}
```

## Match failures

When the input does not match, `Err()` returns a `*MatchError`. On a
successful match it returns `nil`.

```go
result, err := g.Match("1 + * 2")
defer result.Close()

if err := result.Err(); err != nil {
    fmt.Println(err)
    // Line 1, col 5: expected "(", an identifier, or a number

    fmt.Println(err.(*ohm.MatchError).Message())
    // Line 1, col 5:
    // > 1 | 1 + * 2
    //           ^
    // Expected "(", an identifier, or a number
}
```

`Message()` shows the failure line with a caret under the failure. It
also shows the line before and the line after, when they exist. The
text follows `message` and `shortMessage` in miniohm.

`FailurePosition()` gives the place of the rightmost failure as a
`Position`:

```go
pos := result.FailurePosition()
fmt.Printf("%d:%d\n", pos.Line, pos.Column) // 1:5
```

| field        | unit                                                  |
|--------------|-------------------------------------------------------|
| `Offset`     | UTF-16 code units from the start, as the Wasm reports |
| `ByteOffset` | bytes from the start of the Go string                 |
| `Line`       | 1-based; lines split at `\n`                          |
| `Column`     | 1-based, in code points; a `\r` is not counted        |

A character outside the Basic Multilingual Plane, such as an emoji, is
one column. A tab is one column. `RightmostFailurePosition()` gives the
raw UTF-16 offset, as the miniohm method of the same name does.

`FailurePosition()` and `RightmostFailurePosition()` read only Go
memory. They stay valid after `Close()`. On a successful match, the
position is the end of the input.

### Next valid tokens

`Expected()` gives what the grammar accepts at the failure position:

```go
expected, err := result.Expected()
for _, ex := range expected {
    fmt.Println(ex.Kind, ex.Text)
}
// literal "("
// description an identifier
// description a number

text, err := result.ExpectedText()
// "(", an identifier, or a number
```

| `Kind`                | `Text`, for example | note                          |
|-----------------------|---------------------|-------------------------------|
| `ExpectedLiteral`     | `"("`               | `Literal` holds `(`, unquoted |
| `ExpectedRange`       | `"a".."z"`          |                               |
| `ExpectedEnd`         | `end of input`      |                               |
| `ExpectedDescription` | `a number`          | a rule, or its description    |

`ExpectedKind.String()` gives `literal`, `range`, `end` or
`description`.

**Fluffy failures.** A repetition that matched some input, then failed
at the failure position, records a fluffy failure. For `G {\n  S = *x\n}`
on the Ohm grammar, `"}"` is the only real failure; `"|"`, `"~"`, `"("`
and others are fluffy. `Expected()` leaves fluffy failures out, as
miniohm does. `RightmostFailures()` keeps them, with `Fluffy` set:

```go
failures, err := result.RightmostFailures()
for _, f := range failures {
    fmt.Println(f.Kind, f.Text, f.Fluffy)
}
```

Failures with the same `Text` are merged. A merged failure is fluffy
only when every copy is fluffy. The order is the order of first
recording.

When every failure is fluffy, `Expected()` is empty and the message
reads `expected ` with nothing after it, as in miniohm. `Expected()`
does not fall back to the fluffy failures; use `RightmostFailures()`
for them.

**Lifetime.** The first call to `RightmostFailures()`, `Expected()` or
`ExpectedText()` parses the input a second time. So the result must be
the most recent one, and not closed; else the call returns an error.
Later calls return a cache, also after `Close()`. `Close()` frees the
heap of both parses.

`Err()` fills `MatchError.Expected` when the result is the most recent,
or when the cache is filled. Else `Expected` is nil, and the short
message reads `Line 1, col 5: match failed`. So call `Err()` before the
next `Match`.

**Known issue.** In Wasm modules from compiler `18.0.0-beta.15`, the
second parse can trap. Examples on the arith grammar are an unclosed
`(`, such as `1 + (2`, and `1)`. miniohm traps on the same input.
`Expected()` then returns the error, and `Err()` gives the position
only. The second parse can also depend on earlier matches on the same
`Grammar`. See `docs/kb/ohm-failure-recording.md` at the repository
root.

## Build and Test

From the compiler package directory (`packages/compiler`):

```bash
make go-test-es5    # build es5.wasm and run Go tests
```
