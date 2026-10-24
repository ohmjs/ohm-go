# Error reporting — PRD

This PRD specifies how the `ohm` Go package reports a failed match, and
how `ohm-cli` tests that report. When input does not match a grammar, a
`MatchResult` today says only that it failed. After this work it says
where it failed, as a line and a column with an excerpt of the input,
and what the grammar would have accepted at that point, as a list of the
next valid tokens. Part A (sections 4 to 6) gives the position. Part B
(sections 7 and 8) gives the next valid tokens. Section 9 puts both into
the `ohm-cli` commands. Section 10 extends the `ohm-cli` txtar tests so a
fixture can state the expected error and the expected next tokens. All
tests for this feature live in `ohm-cli`, not in `ohm`. The
[plan](plan.md) is the checklist.

**Status (2026-10-25).** Part A is built in `ohm` (sections 4 to 6,
plan Phase 1): `Position`, `FailurePosition`, `MatchError` and `Err()`.
Part B is built in `ohm` (sections 7 and 8, plan Phase 3):
`RightmostFailures`, `Expected`, `ExpectedText`, and `Err()` fills
`MatchError.Expected`. Fixtures test both (plan Phases 2 and 4): the
`error` and `expected` txtar kinds with their flags `o`, `N`, `s`, `f`
and `start=`, the command `ohm-cli test match_error` with `--expected`
and `--fluffy`, and the Go tests in `ohm-cli/tests/match_error_test.go`
(all of section 10). Each fixture matches miniohm in the
`ohmjs/ohm:18.0.0-beta.15` image, except the emoji column (section 5).
Open decision 11.1 is closed, with its arith half left open by an
upstream trap (section 3). Section 9 is built (plan Phase 5): the
six commands print `path:line:col: expected …` and the excerpt, and
`ohm-cli test txtar` names an input `archive:member`. Every phase of the
plan is done.

## 0. Blockers

None. The `recordFailures` trap that blocked plan Phase 4 is worked
around; section 3 records it.

`go test ./...` in `ohm/` fails before any change, because
`TestES5Match` reads an `es5.wasm` that is not in the repository. That
does not block this feature, because its tests live in `ohm-cli`. The
plan lists it under **Not in any phase**.

## Decisions at a glance

| concern | decision | runners-up |
|---|---|---|
| Source of failure data | The Wasm exports: `rightmostFailurePos`, `recordFailures`, `getRecordedFailuresLength`, `recordedFailuresAt`, `isFluffy`, and the `strings` custom section (section 3) | Re-parse in Go; port the ohm-js failure tracer |
| When the position is read | At the end of `Match`, stored on `MatchResult` (section 4) | Read the global on demand: wrong after the next `Match` |
| Offset unit kept | UTF-16 code units, as the Wasm reports, plus a byte offset into the Go string (section 4) | Byte offset only |
| Column unit | 1-based, in Unicode code points; `\r` is not counted (section 5) | UTF-16 units (ohm-js and LSP parity); bytes (`go/token`) |
| Error surface | `MatchResult.Err()` returns `*MatchError`, which implements `error`; `nil` on success (section 6) | `Message()` string only; `Match` returns an error on mismatch (breaks every caller) |
| Message text | Same shape as `message` and `shortMessage` in miniohm, the ohm-js Wasm runtime (section 6) | A Go-specific shape |
| When expected tokens are computed | On first call, then cached; the result must be the most recent (section 8) | At `Match` time: a second parse on every failure, wanted or not |
| Start rule for the second parse | The rule that `Match` used (section 8) | `ruleNames[0]`, as miniohm does: wrong when a start rule is given |
| Shape of a next valid token | `Expected{Kind, Text, Literal}`, with the kind parsed from the description (section 7) | Raw strings; an upstream table of kinds |
| Fluffy failures | Left out of `Expected()`; kept, marked, in `RightmostFailures()` (section 7) | Dropped |
| Empty or short expected set | No fall back: `Expected()` stays empty when every failure is fluffy, as in miniohm (section 11.1) | Fall back to the fluffy failures when no other failure exists |
| CLI format | `path:line:col: expected …`, then the excerpt (section 9) | The long message only |
| Where tests live | `ohm-cli`: txtar fixtures, plus Go tests in `ohm-cli/tests` for what a fixture cannot state (section 10) | New tests in `ohm/`, with a Wasm fixture copied into `ohm/testdata` |
| How a fixture states an error | Two new expectation kinds, `<test>.error` and `<test>.expected`, beside `c_and_p` and `rule_ast` (section 10.2) | One kind that holds both; a `want-error` flag on `c_and_p` |
| Command behind the new kinds | A new command, `ohm-cli test match_error`, as each existing kind has one (section 10.5) | Code in `tests.go` only, with no command |
| Grammar a fixture matches against | The archive's `.ohm`, compiled by docker as for `c_and_p`; or, with flag `o`, the embedded Ohm grammar, with no docker (section 10.3) | A precompiled `.wasm` named in the archive |

## 1. Problem

A caller of `ohm` that gets a failed match has one bit: `Failed()`. Every
call site in `ohm-cli` turns that bit into `match failed`, with no
position. Six call sites do this: `sexpr/sexpr.go`, `candp/candp.go`,
`es5/roundtrip.go`, `ruleast/gen_go.go`, `ruleast/gen_go_types_tmpl.go`
and `ruleast/exec_grammar2ruleast.go`. A user with a 400-line grammar
file gets no line, no column and no hint. They bisect the file by hand.

The txtar tests have the same gap. A fixture can state the CST of an
input that matches. It cannot state what happens to an input that does
not. An input that fails to match is a processing error, which `go test`
does not count (section 10.1).

miniohm, the ohm-js runtime for Wasm grammars
(`packages/runtime/src/miniohm.ts`), solves the first problem with
`getRightmostFailurePosition()`, `getRightmostFailures()`,
`getExpectedText()`, `message` and `shortMessage` on
`FailedMatchResult`. The Go runtime ports `Grammar` and `MatchResult`
from that file, but not these methods.

## 2. Users and use cases

1. **A user of `ohm-cli`** runs a command on a grammar or an input that
   does not parse. They want `file:line:col` so their editor jumps there.
2. **A Go program that embeds a grammar** wants to show its own user
   where their input is wrong, and in what way.
3. **A tool author** wants the next valid tokens at the failure point,
   to build hints, completions or a "did you mean" message.
4. **A grammar author** wants a fixture that pins the error for a bad
   input, so a grammar change that makes the error worse fails a test.

Use cases 1 and 2 need Part A. Use case 3 needs Part B. Use case 4 needs
section 10. Part A ships first, because it is cheap and serves the most
callers.

## 3. What the Wasm already gives us

The compiler emits failure tracking into every grammar module. The
runtime source is `packages/compiler/runtime/ohmRuntime.ts` in the Ohm
repository.

- **`rightmostFailurePos`** (exported global, `i32`). The rightmost input
  position, in UTF-16 code units, at which any expression failed.
  `match` resets it to `-1` and sets it during the parse. The next
  `match` overwrites it.
- **`recordFailures(inputLength, startRuleId)`** (exported function). It
  runs the parse again with `errorMessagePos` set to the saved
  `rightmostFailurePos`. In that run, each expression that fails at that
  position pushes a failure id. The run allocates on the Wasm heap above
  the current bump pointer, and calls `fillInputBuffer` again.
- **`getRecordedFailuresLength()`, `recordedFailuresAt(i)`,
  `isFluffy(i)`** (exported functions). They read the recorded ids. A
  failure id indexes the `strings` custom section, which `NewGrammar`
  already parses into `Grammar.strings`.
- **Fluffy failures** come from a repetition that failed at the error
  position after it had already succeeded. miniohm drops them from the
  expected text.
- **Failure descriptions** come from `toFailureDescription` in
  `packages/compiler/src/Compiler.ts`. A terminal is JSON-quoted
  (`"("`). A range is `"a".."z"`. A rule application is the rule's
  description, or `a <rule>` / `an <rule>`. The end is `end of input`.
  Id 0 is always `end of input`.

The probe on 2026-10-24 checked `arith.wasm`, `ohm-grammar.wasm` (both
`18.0.0-beta.15`) and `es5.wasm` (`18.0.0-beta.16`). All three export
every name above. Probe output for `arith.wasm`:

| input | `rightmostFailurePos` | recorded failures |
|---|---|---|
| `1 + * 2` | 4 | `"("`, `an identifier`, `a number` |
| `abc 12` | 4 | `end of input` |
| empty | 0 | `"("`, `an identifier`, `a number` |
| `1 +\n (2 * x` | 11 | traps; see below |

Probe output for the embedded Ohm grammar, `ohm-cli/utils/ohm-grammar.wasm`:

| input | `rightmostFailurePos` | non-fluffy failures | fluffy failures |
|---|---|---|---|
| `G {\n  S = *x\n}` | 10 | `"}"` | 8 distinct texts |
| `G {\n  S = "a"` | 13 | `"}"` | 12 distinct texts |
| `G { S = "a" }\nH` | 15 | `"{"` | 3 distinct texts |

Both tables fed open decision 11.1. miniohm gives the same failures on
each input that does not trap
([ohm-failure-recording](../../kb/ohm-failure-recording.md)).

**Known upstream bug: `recordFailures` traps.** On `1 +\n (2 * x`, and
on 1442 of the 7380 arith inputs of up to four characters, the second
parse traps with a Wasm stack overflow. Examples are `(x`,
`1 + (2` and `1)`. miniohm in the `ohmjs/ohm:18.0.0-beta.15` image traps
on each example checked, so the bug is in the compiler runtime. The Go
runtime returns the error from `RightmostFailures`, and `Err()` gives
the position only. The recorded failures can also depend on earlier
matches on the same `Grammar`, in miniohm and in Go. The arith fixtures
use `(1 +\n (2 * x)` for the fluffy case, and pin the trap with test
`unclosed`. Checked 2026-10-25; the kb page has the commands.

## 4. Part A: the position API

`Match` reads `rightmostFailurePos` after the parse and stores it on the
`MatchResult`. This must happen in `Match`, because the next `Match` or
`recordFailures` call resets the global.

```go
// Position is a place in the input of a MatchResult.
type Position struct {
	Offset     int // UTF-16 code units from the start, as the Wasm reports
	ByteOffset int // bytes from the start of the Go string
	Line       int // 1-based
	Column     int // 1-based, in code points; see section 5
}

// RightmostFailurePosition returns the UTF-16 offset of the rightmost
// failure. It mirrors the miniohm method of the same name.
func (r *MatchResult) RightmostFailurePosition() int

// FailurePosition returns the rightmost failure as a Position.
func (r *MatchResult) FailurePosition() Position
```

Both methods are valid for the life of the result, also after `Close`,
because they read only Go memory. On a successful match they return the
end of the input. A caller must not read meaning into that.

The conversion from a UTF-16 offset to a `Position` is a pure function,
`positionAt`, in a new file, `ohm/position.go`.

## 5. Line and column rules

- Lines are split at `\n`. A `\r` is not counted as a column, so `\r\n`
  input gives the same columns as `\n` input. This matches
  `getLineAndColumn` in ohm-js `extras.ts`.
- A column counts code points, 1-based. A character outside the Basic
  Multilingual Plane, such as an emoji, is one column, not two.
- A tab is one column.
- An offset at the end of the input points one column past the last
  character.
- An offset inside a surrogate pair rounds down to the start of the
  pair. The Wasm should not report one; the rule makes the function
  total.

For ASCII input, the column is the same as the miniohm column. For input
with characters outside the Basic Multilingual Plane, the column is
smaller than the miniohm column. `Position.Offset` keeps the miniohm
unit for a caller that needs it.

## 6. Part A: the error and its message

```go
// MatchError describes a failed match.
type MatchError struct {
	Input    string
	Pos      Position
	Expected []Expected // nil until Part B; see section 8
}

func (e *MatchError) Error() string        // the short message
func (e *MatchError) ShortMessage() string // Line 2, col 9: expected ...
func (e *MatchError) Message() string      // excerpt with a caret, then Expected ...

// Err returns nil when the match succeeded, else a *MatchError.
func (r *MatchResult) Err() error
```

The message text follows miniohm, so that tests and users see one shape
across the two runtimes.

`ShortMessage`:

```
Line 1, col 5: expected "(", an identifier, or a number
```

`Message`:

```
Line 1, col 5:
> 1 | 1 + * 2
          ^
Expected "(", an identifier, or a number
```

The excerpt shows the line before and the line after, when they exist,
as `getLineAndColumnMessage` does. A line number is right-aligned to
the width of the largest line number shown.

Until Part B lands, the expected text is absent. `ShortMessage` reads
`Line 1, col 5: match failed`. `Message` ends with the caret line.

## 7. Part B: the next valid tokens

```go
// ExpectedKind classifies one next valid token.
type ExpectedKind int

const (
	ExpectedLiteral     ExpectedKind = iota // a terminal, such as "("
	ExpectedRange                           // a range, such as "a".."z"
	ExpectedEnd                             // end of input
	ExpectedDescription                     // a rule or its description, such as "a number"
)

// Expected is one thing the grammar accepts at the failure position.
type Expected struct {
	Kind    ExpectedKind
	Text    string // the description, as miniohm prints it
	Literal string // the unquoted terminal; set for ExpectedLiteral only
}

// Failure is one recorded failure at the rightmost position.
type Failure struct {
	Expected
	Fluffy bool
}

// RightmostFailures returns every failure recorded at the rightmost
// position, fluffy ones included, with duplicates merged.
func (r *MatchResult) RightmostFailures() ([]Failure, error)

// Expected returns the next valid tokens: the non-fluffy failures.
func (r *MatchResult) Expected() ([]Expected, error)

// ExpectedText joins Expected as miniohm getExpectedText does:
// "a", "a or b", "a, b, or c".
func (r *MatchResult) ExpectedText() (string, error)
```

The kind comes from the description text, because the Wasm records
only a string id. The rules, in order:

1. Id 0, or the text `end of input`: `ExpectedEnd`.
2. Text that matches `"x".."y"`, with both sides valid JSON strings:
   `ExpectedRange`.
3. Text that is one valid JSON string: `ExpectedLiteral`. `Literal` is
   the decoded string. A case-insensitive terminal also lands here.
4. Anything else: `ExpectedDescription`.

A rule description that is itself a quoted string falls into rule 3.
That is rare, and `Text` still prints right.

Duplicates merge by `Text`. A merged failure is fluffy only if every
copy is fluffy, as in miniohm. Order is the order of first recording,
which is the order miniohm prints.

`ExpectedKind` has a `String()` method that gives `literal`, `range`,
`end` and `description`. The `.expected` fixtures in section 10.4 use
these words.

## 8. Part B: lifetime and cost

`recordFailures` parses the input a second time. Five facts follow.

- **Most recent only.** The second parse uses the Wasm heap, the memo
  table and the grammar's current input. These belong to the most recent
  `MatchResult`. `RightmostFailures` returns an error when the result is
  not on top of `Grammar.resultStack`, or is closed. The error text
  follows miniohm `_assertMostRecent`.
- **Cached.** The first call stores the failures on the `MatchResult`.
  Later calls, also after `Close`, return the cache.
- **Start rule.** `Match` stores the start rule id on the
  `MatchResult`, and `recordFailures` gets that id. miniohm passes
  `ruleNames[0]`; that is wrong when the caller gave a start rule.
- **Heap.** The second parse allocates above the watermark of this
  result. `Close` resets to the watermark, so it frees both parses.
- **Cost.** The second parse costs about as much as the first. It runs
  only when a caller asks.

`Err()` calls `RightmostFailures` when the result is the most recent.
It fills `MatchError.Expected` from the cache. When the result is not
the most recent, it returns the position only, with `Expected` nil. So a
caller that calls `Err()` before the next `Match` always gets the full
message.

## 9. ohm-cli commands

Each of the six call sites in section 1 replaces its `match failed`
error with an error that wraps `mr.Err()` and names the input. The
format is `path:line:col: expected …`, the form that `go vet` and most
editors parse. The `Message` excerpt follows on the next lines. Where
the input has no path, such as a string from a txtar archive, the
archive and member name take its place:
`tests/grammar_mixed_01.txtar:file1.txt:1:1: expected …`. The excerpt
leaves out the `Line L, col C:` header and the `Expected …` line of
`Message`, because the first line already gives both. An input given on
the command line as text, not as `@path`, is named `<input>`.

## 10. Testing through ohm-cli

All tests for this feature live in `ohm-cli`. Most are txtar fixtures.
A few lifetime rules cannot be stated as a fixture; they are Go tests in
`ohm-cli/tests` (section 10.6).

### 10.1 The mechanism today

`ohm-cli test txtar <files>` and the Go test `TestTxtar` run each
`ohm-cli/tests/*.txtar` archive through `testFile` in `tests/tests.go`.
An archive holds one grammar, `<name>.ohm` (or `@<path>` to a file on
disk). Each other member is an input or an expectation. Its name is
`<test>.<flags>.<kind>`:

| kind | input | grammar | output |
|---|---|---|---|
| `txt` | — | — | an input for the expectations of the same `<test>` |
| `sexpr` | the archive grammar | the Ohm grammar | CST of the grammar |
| `c_and_p` | `<test>.txt` | the archive grammar, compiled by docker | CST of the input |
| `rule_ast` | `<test>.txt` | the Ohm grammar | rule AST |
| `go_types`, `go_interfaces`, `go_accepts` | `<test>.txt` | the Ohm grammar | generated Go |

Each flag is one letter between dots. `RunTests` compares the output
with the member, after `strings.TrimSpace` on both. A processing error,
such as an input that does not match, is printed and counted apart. It
fails the run only under `--strict`. `TestTxtar` does not pass
`--strict`, so an input that does not match passes `go test`.

### 10.2 Two new kinds

- **`<test>.<flags>.error`**: the error from matching `<test>.txt`. By
  default it is `MatchError.Message()`. Flag `s` gives
  `ShortMessage()`.
- **`<test>.<flags>.expected`**: the next valid tokens from matching
  `<test>.txt`. One token per line, as `<kind> <text>`. Flag `f` adds
  fluffy failures, each marked ` (fluffy)` at the end of its line.

When the match succeeds, the output of either kind is the one line
`match succeeded`. The expectation then shows as a mismatch with a
diff, not as a processing error, so `go test` fails.

### 10.3 Flags for both new kinds

| flag | effect |
|---|---|
| `o` | Match against the embedded Ohm grammar, not the archive grammar. Needs no docker. |
| `N` | Remove one trailing `\n` from the input. A txtar member always ends in `\n`; this flag lets a fixture test the end of input. |
| `start=<Rule>` | Use `<Rule>` as the start rule. |
| `s` | `.error` only: the short message. |
| `f` | `.expected` only: add fluffy failures. |

`start=<Rule>` is the first flag that is not one letter. A rule name
holds no `.`, so the split on `.` still works.

With `o`, the archive needs no grammar file. Today `testFile` panics
when an archive has no `.ohm`, or when a `.txt` comes before it. Both
checks move: the grammar is required only by a kind that uses it.

### 10.4 Example

```
-- grammar.ohm --
Arithmetic { … as examples/generic-methods/arith.ohm … }
-- bad_op.txt --
1 + * 2
-- bad_op.error --
Line 1, col 5:
> 1 | 1 + * 2
          ^
Expected "(", an identifier, or a number
-- bad_op.s.error --
Line 1, col 5: expected "(", an identifier, or a number
-- bad_op.expected --
literal "("
description an identifier
description a number
-- trailing.txt --
abc 12
-- trailing.expected --
end end of input
-- prim.txt --
x +
-- prim.start=PriExp.expected --
end end of input
```

A fixture for a grammar syntax error needs no `.ohm` and no docker:

```
-- unclosed.txt --
G {
  S = "a"
-- unclosed.o.N.s.error --
Line 2, col 10: expected "}"
```

### 10.5 The command behind the kinds

Each existing kind is backed by an `ohm-cli` command, and its fixture
is that command's output. The new kinds follow the same pattern with a
new command, in a new package `ohm-cli/matcherr`:

```
ohm-cli test match_error [--grammar <file.ohm>] [--start <Rule>]
                         [--short] [--expected] [--fluffy] <source>
```

- Without `--grammar`, the command matches against the embedded Ohm
  grammar.
- `<source>` is the input text, or `@<path>` to read a file, as for
  `c_and_p`.
- `--expected` prints the tokens; otherwise the command prints the
  message.

The command compiles the grammar with the same code and the same
per-process cache as `c_and_p`. That code moves out of
`CompileAndParseCmd.compile` into an exported `candp.Compile`.

The command is useful outside tests: a grammar author can run it on a
file to see the error and the next tokens.

### 10.6 Go tests in ohm-cli/tests

A fixture states the output of one match. It cannot state a rule about
two matches, or about `Close`. These rules are Go tests in a new file,
`ohm-cli/tests/match_error_test.go`. They match against the embedded
Ohm grammar, so they need no docker. `ohm-cli/tests/typecheck_tests.go`
already tests the `ohm` package from here.

- `RightmostFailures` after a second `Match` returns the most-recent
  error.
- `Expected()` after `Close` returns the cache.
- `Close` after `RightmostFailures` restores the heap watermark.
- `FailurePosition()` after `Close` returns the same value.
- `\r\n` input gives the same line and column as `\n` input. A txtar
  member can hold `\r`, but git settings can strip it, so this case is
  Go code.
- Empty input gives line 1, column 1.

### 10.7 Docker

A default `.error` or `.expected` fixture compiles the archive grammar
with docker, as `c_and_p` does. Without docker, the compile fails, the
fixture counts as a processing error, and `go test` passes without
running it. A fixture with flag `o` always runs. CI must run
`ohm-cli test txtar --strict tests/*.txtar` so a fixture that did not
run fails the build.

### 10.8 Parity with miniohm

The text in an `.error` or `.expected` fixture should match what
miniohm prints for the same grammar and input. The fixture's txtar
comment, the text before the first member, names the miniohm version
that produced it. A difference is a bug in the Go runtime, or the
column unit of section 5.

## 11. Open decisions

`docs/kb/ohm-failure-recording.md` analyses decision 1. No page
analyses decision 2 yet.

1. **Closed 2026-10-25: an empty or a short expected set.** Decided:
   `Expected()` keeps the miniohm rule, the non-fluffy failures only,
   with no fall back to the fluffy failures. It moved to Decisions at a
   glance. Evidence, in `docs/kb/ohm-failure-recording.md`: for
   `S = *x` on the Ohm grammar, miniohm gives `"}"` alone, as Go does,
   with the same eight fluffy texts. The arith input `1 +\n (2 * x`
   traps in miniohm and in Go, so that half is not decided by a run; no
   arith input of up to four characters gives an empty set on a fresh
   `Grammar`. The one empty set found comes from the state of an earlier
   match, in miniohm too, where the message reads `expected ` with
   nothing after it. Rerun the arith half when the compiler stops
   trapping.
2. **Column unit.** Code points (decided above) or UTF-16 units. A
   language server needs UTF-16 by default. `Position.Offset` covers
   that need today. Revisit if an LSP user appears.

## 12. Out of scope

- Next valid tokens at a cursor in input that matched, for completion
  in an editor. The Wasm records failures only at the rightmost failure
  position.
- More than one error per parse, or error recovery.
- An upstream change to the compiler, such as a table of failure kinds.
- `.ohm` grammar syntax errors from the docker compiler. Those come from
  ohm-js, which already reports a position.
- A precompiled `.wasm` member in a txtar archive, to run fixtures
  without docker.
- Unit tests in the `ohm` package for this feature.
