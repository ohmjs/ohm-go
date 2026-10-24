# txtar tests

The fixtures in `ohm-cli/tests/*.txtar` test `ohm-cli` and the `ohm`
runtime. Each archive holds inputs and the expected output of an
`ohm-cli` command on them. This page describes the archive format, every
kind of member, every flag and `--strict`.

## Running the tests

From the `ohm-cli` directory:

```sh
go test ./tests/                                   # TestTxtar, one subtest per archive
go test ./tests/ -run TestTxtar/error_arith        # one archive
go run . test txtar tests/*.txtar                  # the same, from the CLI
go run . test txtar --strict tests/*.txtar         # also fail on a processing error
go run . test txtar -r '^bad_' tests/error_arith.txtar   # only tests whose name matches
```

`test txtar` must run from the `ohm-cli` directory. It changes to
`tests/` before it reads the archives, so an `@<path>` in an archive is
relative to `tests/`.

## The archive

An archive is a [txtar](https://pkg.go.dev/golang.org/x/tools/txtar)
file. The text before the first member is a comment; use it to say what
the archive tests. Each member is one of:

- the grammar, `<name>.ohm`. An archive has at most one grammar.
- an input, `<test>.txt`.
- an expectation, `<test>.<flags>.<kind>`. Its content is the expected
  output of the command behind `<kind>` (see below).

The name splits on `.`. The first part is the test name. The last part
is the kind. The parts between are the flags. A test can have many
expectations, each with other flags:

```
-- grammar.ohm --
G { S = "a" }
-- one.txt --
a
-- one.c_and_p --
(S (_terminal 'a' 0-1)
)
-- bad.txt --
b
-- bad.s.error --
Line 1, col 1: match failed
```

An expectation follows the `.txt` of its test. When the kind uses the
archive grammar, it also follows the `.ohm`.

A member whose name starts with `!` is skipped. A member of an unknown
kind is ignored; `<test>.comment` is a common way to add notes.

**The grammar from a file.** A grammar member named `@<path>` reads the
grammar from `<path>`, relative to `tests/`. Its content is ignored. See
`tests/ohm-grammar.txtar`.

**Trailing newline.** A txtar member always ends in `\n`, so each input
does too. Flag `N` of the `error` kind removes it, so a fixture can test
the end of input. An empty member, with the next `--` line right after
its own, is the empty input.

## Kinds

| kind | input | grammar | output, as the command prints it |
|---|---|---|---|
| `txt` | — | — | an input for the expectations of the same test |
| `sexpr` | the archive grammar | the Ohm grammar | the CST of the grammar: `test to_sexpr` |
| `c_and_p` | `<test>.txt` | the archive grammar, compiled by docker | the CST of the input: `test c_and_p` |
| `rule_ast` | `<test>.txt`, a grammar | the Ohm grammar | the rule AST: `test rule_ast` |
| `go_types`, `go_interfaces`, `go_accepts` | `<test>.txt`, a grammar | the Ohm grammar | the generated Go: `generate parts <kind>` |
| `error` | `<test>.txt` | the archive grammar, compiled by docker; the Ohm grammar with flag `o` | the error message: `test match_error` |

"The Ohm grammar" is the grammar of Ohm itself, embedded in `ohm-cli` as
`utils/ohm-grammar.wasm`. It needs no docker.

**`sexpr`** is the one kind with no `.txt`. Its input is the archive
grammar, as text.

**`error`** matches the input and prints `MatchError.Message()`: the
line and column, the failure line with a caret under the failure, and
the lines before and after. When the match succeeds, it prints
`match succeeded`. A fixture that expects an error, on an input that
matches, then fails as a mismatch with a diff:

```
-- bad_op.txt --
1 + * 2
-- bad_op.error --
Line 1, col 5:
> 1 | 1 + * 2
          ^
  2 | 
-- bad_op.s.error --
Line 1, col 5: match failed
```

Line 2 is shown because the input ends in `\n`. See
`tests/error_arith.txtar` and `tests/error_ohm_grammar.txtar`.

## Flags

Each flag is one letter between dots, except `start=<Rule>`. A rule
name holds no `.`, so the split on `.` still works. A flag that the
kind does not know panics.

| kind | flag | effect |
|---|---|---|
| `sexpr`, `c_and_p` | `t` | exclude terminals |
| | `S` | syntactic rules only (names that start upper case) |
| | `b` | exclude built-in rules such as `letter` and `space` |
| | `l` | prune lists: put the children of `_list` in its parent |
| | `n` | no newline before a closing `)` |
| | `v` | verbose: the source range and text of each node |
| `rule_ast` | `s` | skip the source of each rule |
| | `l` | end each line with the Go line that wrote it |
| `go_types`, `go_interfaces`, `go_accepts` | `m` | `--generic-methods` |
| | `g` | `--no-generics` |
| | `s` | `--skip-type-check-method` |
| `error` | `o` | match against the Ohm grammar, not the archive grammar; needs no docker |
| | `N` | remove one trailing `\n` from the input |
| | `s` | print the one-line `ShortMessage()` |
| | `start=<Rule>` | use `<Rule>` as the start rule |

With flag `o` on each `error` member, an archive needs no grammar. The
grammar is required only by a kind that uses it: `sexpr`, `c_and_p` and
`error` without `o`.

## Comparison, processing errors and `--strict`

The output and the member are compared after `strings.TrimSpace` on
both. A difference is a mismatch: the run prints a diff and fails.

A command that returns an error is a processing error. Examples are a
docker compile that fails, and a `c_and_p` input that does not match.
The run prints the error and counts it apart. Without `--strict`, a
processing error does not fail the run. So `go test` passes on a
machine without docker, and still runs the kinds that need none.

`--strict` fails the run on any processing error too. Use it in CI and
before a commit, so that a fixture that did not run fails:

```sh
go run . test txtar --strict tests/*.txtar
```

`TestTxtar` does not pass `--strict`.

## Docker

`c_and_p` and `error` without `o` compile the archive grammar with the
`ohmjs/ohm` docker image, at the tag that matches the `ohm` runtime.
Each grammar is compiled once per process. Docker's own output goes to
stderr.

## Writing the expected output

No tool writes the expected output. Run the command behind the kind,
check its output, and paste it into the member. For an `error` member:

```sh
go run . test match_error --grammar grammar.ohm '1 + * 2'
go run . test match_error --short $'G {\n  S = "a"'     # Ohm grammar: flag o
go run . test match_error --start PriExp @input.txt     # flag start=PriExp
```

Mind the trailing `\n` of the `.txt` member: pass it, or use flag `N`.
