# ohm-cli — CLI for Ohm, for Go

`ohm-cli` is a command-line tool that helps you compile `.ohm` grammar files into WebAssembly (`.wasm`) for use with the [`goohm`](https://github.com/ohmjs/ohm-go/ohm) Go runtime.

It does not compile grammars directly. Instead, it generates the Docker command needed to compile them via the official `ohmjs/ohm` image — letting you review, adapt, or automate the compilation step however you like.

## Overview

The typical workflow for using Ohm in a Go project looks like this:

```
my-grammar.ohm  →  (docker compile)  →  my-grammar.wasm  →  embedded in Go binary
```

`ohm-cli` handles the middle step by generating the correct `docker run` invocation, pinned to the version of the `goohm` runtime you're using.

## Installation

```bash
go install github.com/ohmjs/ohm-go/ohm-cli@latest
```

Or build from source:

```bash
git clone https://github.com/ohmjs/ohm-go/ohm-cli
cd ohm-cli
go build -o ohm-cli .
```

## Commands

### `generate command`

Generates a Docker command to compile a `.ohm` grammar file into a `.wasm` file compatible with the current `goohm` runtime version.

<!--tmpl,code=bash:go run main.go generate command -h -->
``` bash 

  Usage: ohm-cli generate command [options] <source-file>

  Generate a command, in the specified format (default is 'command') to compile a .ohm grammar file
  using the ohmjs/ohm docker image. Does not actually compile the file.

  Path to .ohm grammar file to compile.

  Options:
  --debug, -d
  --docker-tag, -t    The version tag of the ohmjs/ohm docker image to use in the generated command.
                      Defaults to the version of the goohm runtime included in this cli. (default
                      18.0.0-beta.15)
  --grammar-name, -g
  --output, -o
  --format, -f        Output format. One of: command, go_generate, script. (default command)
  --help, -h          display help

```
<!--/tmpl-->

The `--docker-tag` default is derived at runtime from the `goohm` package, so it always matches the runtime version bundled with this CLI — preventing version mismatches between the compiler and the runtime.

## Output Formats

The `--format` flag controls how the generated command is wrapped.

### `command` (default)

A ready-to-run shell snippet with a comment:

<!--tmpl,code=bash:go run main.go generate command my-grammar.ohm -->
``` bash 

# To generate a .wasm file for use with this version of the runtime, run:
docker run --rm -v "$PWD":/local ohmjs/ohm:18.0.0-beta.15 compile my-grammar.ohm
```
<!--/tmpl-->

### `go_generate`

A `//go:generate` directive for embedding in a Go source file:

<!--tmpl,code=bash:go run main.go generate command --format=go_generate my-grammar.ohm -->
``` bash 

//go:generate docker run --rm -v $PWD:/local ohmjs/ohm:18.0.0-beta.15 compile my-grammar.ohm
```
<!--/tmpl-->

Paste this into any `.go` file in your package. Running `go generate ./...` will compile the grammar and produce `my-grammar.wasm` in the same directory.

### `script`

A standalone shell script with a shebang:

<!--tmpl,code=bash:go run main.go generate command --format=script my-grammar.ohm -->
``` bash 
#!/bin/sh

docker run --rm -v "$PWD":/local ohmjs/ohm:18.0.0-beta.15 compile my-grammar.ohm
```
<!--/tmpl-->

Useful when you want a reusable script checked into your repo:

```bash
ohm-cli generate command --format=script my-grammar.ohm > compile.sh
chmod +x compile.sh
./compile.sh
```

## Using the Compiled Grammar in Go

Once you have a `.wasm` file, load it with `goohm`:

```go
package main

import (
    "context"
    _ "embed"
    "log"

    "github.com/ohmjs/ohm-go/ohm"
)

//go:embed my-grammar.wasm
var wasmBytes []byte

func main() {
    ctx := context.Background()
    grmr, err := ohm.NewGrammar(ctx, wasmBytes)
    if err != nil {
        log.Fatalf("creating grammar: %v", err)
    }
    defer grmr.Close()

    result, err := grmr.Match("Hello, world!")
    if err != nil {
        log.Fatalf("matching: %v", err)
    }
    defer result.Close()

    if result.Succeeded() {
        log.Println("match succeeded")
    }
}
```

## Generating Go visitors

`generate go` turns a grammar into a Go package of node structs, visitor
interfaces and `Accept` methods, so a visitor can be written against the
rules of the grammar rather than the raw CST. The shapes it generates, and
how to write a visitor against them, are described in
[docs/visitors.md](docs/visitors.md).

```sh
ohm-cli generate go -P tree_visitor @tree.ohm
ohm-cli generate go --generic-methods -P tree_visitor @tree.ohm   # Go 1.27 generic methods
```

With `--generic-methods` the node structs have no type parameters; instead
`Accept[P, R any]`, `DefaultAccept[P, R any]` and the `Accept<Arg>[P, R any]`
helpers carry them, and `MakeNodeFromRoot` returns an `ohm.AcceptorFunc`.
One node value can then be visited with different payload and result types.
See [examples/generic-methods](../examples/generic-methods) for a worked
example.

## Errors on input that does not parse

A command that gets a grammar or an input that does not parse prints
where it failed and what the grammar expected there. The first line is
`path:line:col: expected …`, the form that `go vet` and most editors
parse. The failure line with a caret, and the lines around it, follow.
The commands are `generate go`, `generate parts`, `test rule_ast`,
`test to_sexpr`, `test c_and_p` and `es5_roundtrip`.

```sh
$ ohm-cli generate go -P bad @bad.ohm
bad.ohm:5:33: expected "~", "&", "#", an identifier, "\"", "(", "--", "|", or "}"
  4 |          | PriExp
> 5 |   PriExp = "(" Exp ")" | digit+ +
                                      ^
  6 | }
```

An input given with `@path` is named by its path. An input given as
text is named `<input>`:

<!--tmpl,code=bash:go run main.go test rule_ast $'G {\n  S = *x\n}' 2>&1 | grep -v '^exit status' -->
``` bash 
<input>:2:7: expected "}"
  1 | G {
> 2 |   S = *x
            ^
  3 | }
```
<!--/tmpl-->

`test txtar` names an input as `archive:member`. It reports the input
as a processing error, which fails the run under `--strict`:

```
gen result error for gram.rule_ast, err :
tests/bad.txtar:gram.txt:2:7: expected "}"
  1 | G {
> 2 |   S = *x
            ^
  3 | }
```

When the next valid tokens are not known, the first line ends in
`match failed` instead of `expected …`. `test match_error`, below,
prints the same failure in the form of the ohm-js message.

## Showing a match error

`test match_error` matches an input against a grammar and prints where
it failed and what the grammar expected there: the line and column, the
failure line with a caret under the failure, and the next valid tokens.
Without `--grammar` it matches against the grammar of Ohm itself, so it
checks the syntax of a `.ohm` file without docker. With `--grammar` it
compiles the grammar with docker first.

<!--tmpl,code=bash:go run main.go test match_error -h -->
``` bash 

  Usage: ohm-cli test match_error [options] <source>

  Match the source and print the error or the next valid tokens, or 'match succeeded'.

  Input text. If it starts with an @ it is a path to the input file.

  Options:
  --grammar, -g   Path to the .ohm grammar file, compiled with docker. Defaults to the embedded Ohm
                  grammar.
  --start, -r     The start rule. Defaults to the grammar's default start rule.
  --short, -s     Print the one-line short message.
  --expected, -e  Print the next valid tokens, one '<kind> <text>' line each, not the message.
  --fluffy, -f    With --expected, also print the fluffy failures, each marked ' (fluffy)'.
  --help, -h      display help

```
<!--/tmpl-->

<!--tmpl,code=bash:go run main.go test match_error $'G {\n  S = *x\n}' -->
``` bash 
Line 2, col 7:
  1 | G {
> 2 |   S = *x
            ^
  3 | }
Expected "}"
```
<!--/tmpl-->

<!--tmpl,code=bash:go run main.go test match_error --short $'G {\n  S = *x\n}' -->
``` bash 
Line 2, col 7: expected "}"
```
<!--/tmpl-->

`--expected` prints the next valid tokens, one `<kind> <text>` line
each. The kind is `literal`, `range`, `end` or `description`.
`--fluffy` adds the fluffy failures: those of a repetition that ended
at the failure position. The message leaves them out.

<!--tmpl,code=bash:go run main.go test match_error --expected $'G {\n  S = *x\n}' -->
``` bash 
literal "}"
```
<!--/tmpl-->

<!--tmpl,code=bash:go run main.go test match_error --expected --fluffy $'G {\n  S = *x\n}' -->
``` bash 
literal "|" (fluffy)
literal "~" (fluffy)
literal "&" (fluffy)
literal "#" (fluffy)
description an identifier (fluffy)
literal "\"" (fluffy)
literal "(" (fluffy)
literal "--" (fluffy)
literal "}"
```
<!--/tmpl-->

```sh
ohm-cli test match_error --grammar arith.ohm '1 + * 2'
ohm-cli test match_error --grammar arith.ohm --expected '1 + * 2'
ohm-cli test match_error --grammar arith.ohm --start PriExp 'x +'
ohm-cli test match_error @my-grammar.ohm      # read the input from a file
```

On an input that matches, it prints `match succeeded`.

## txtar tests

The fixtures in `tests/*.txtar` hold inputs and the expected output of
`ohm-cli` commands on them, including `test match_error`.
[docs/txtar-tests.md](docs/txtar-tests.md) describes the archive format,
every kind, every flag and `--strict`.

```sh
go test ./tests/
go run . test txtar --strict tests/*.txtar
```

## Shell Completion

Install or remove zsh, bash or fish completion:

```bash
ohm-cli --install    # install zsh, bash or fish completions
ohm-cli --uninstall  # remove zsh, bash or fish completion
```

### Development Notes

<!-- This is current just a placeholder for commands useful for doing dev. -->

```sh
go install
ohm-cli generate go -P ruleast -f ohm_ @../ohm-repo/packages/ohm-js/src/ohm-grammar.ohm

ohm-cli generate parts go_types -P ruleast -o ruleast/ohm_types.go @../ohm-repo/packages/ohm-js/src/ohm-grammar.ohm

# generic methods flavour (go 1.27)
ohm-cli generate go --generic-methods -P ruleast -f ohm_ @../ohm-repo/packages/ohm-js/src/ohm-grammar.ohm

# es5
ohm-cli generate go --grammar-name  ES5 @../ohm-repo/examples/ecmascript/src/es5.ohm
ohm-cli generate go --skip-type-check-method --grammar-name ES5 --hand-coded-walk @../ohm-repo/examples/ecmascript/src/es5.ohm
ohm-cli es5_roundtrip "function x(){}"
```

```sh
# genvisitor.adl -> golang
./genadl.sh

# testing collector
go install && ohm-cli test txtar tests/*.txtar
```