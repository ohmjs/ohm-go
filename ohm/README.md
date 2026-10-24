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
    // Line 1, col 5: match failed

    fmt.Println(err.(*ohm.MatchError).Message())
    // Line 1, col 5:
    // > 1 | 1 + * 2
    //           ^
}
```

`Message()` shows the failure line with a caret under the failure. It
also shows the line before and the line after, when they exist.

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

`FailurePosition()`, `RightmostFailurePosition()` and `Err()` read only
Go memory. They stay valid after `Close()`. On a successful match, the
position is the end of the input.

## Build and Test

From the compiler package directory (`packages/compiler`):

```bash
make go-test-es5    # build es5.wasm and run Go tests
```
