# Error reporting — PRD

This PRD specifies how the `ohm` Go package reports a failed match. When
input does not match a grammar, a `MatchResult` today says only that it
failed. After this work it says where it failed, as a line and a column
with an excerpt of the input, and what the grammar would have accepted
at that point, as a list of the next valid tokens. The work is in two
parts. Part A (sections 4 to 6) gives the position. Part B (sections 7
and 8) gives the next valid tokens. Section 9 puts both into `ohm-cli`.
The [plan](plan.md) is the checklist.

**Status (2026-10-24).** Nothing is built. The Wasm modules that the
runtime accepts already export everything this work needs (section 3).
A probe in Go against `arith.wasm` returned the failure position and the
expected tokens. One blocker, below, stops a clean test run.

## 0. Blockers

**`go test ./...` in `ohm/` fails before any change.** `TestES5Match`
reads `./es5.wasm`, which is not in the repository. The only `es5.wasm`
in this repository, in `ohm-cli/es5/`, is built by `18.0.0-beta.16`,
which `acceptedVersions` comments out. The `ohm` package has no Wasm
fixture that its tests can load.

- Fix: add `ohm/testdata/arith.wasm` (a copy of
  `examples/generic-methods/arith.wasm`, `18.0.0-beta.15`). Make
  `TestES5Match` and `BenchmarkES5Match` call `t.Skip` when the file is
  absent and `OHM_WASM` is unset.
- Checked with: `cd ohm && go test ./...` →
  `grammar_test.go:95: reading wasm file: open ./es5.wasm: no such file or directory`.
- Date: 2026-10-24.

## Decisions at a glance

| concern | decision | runners-up |
|---|---|---|
| Source of failure data | The Wasm exports: `rightmostFailurePos`, `recordFailures`, `getRecordedFailuresLength`, `recordedFailuresAt`, `isFluffy`, and the `strings` custom section (section 3) | Re-parse in Go; port the ohm-js failure tracer |
| When the position is read | At the end of `Match`, stored on `MatchResult` (section 4) | Read the global on demand: wrong after the next `Match` |
| Offset unit kept | UTF-16 code units, as the Wasm reports, plus a byte offset into the Go string (section 4) | Byte offset only |
| Column unit | 1-based, in Unicode code points; `\r` is not counted (section 5) | UTF-16 units (ohm-js and LSP parity); bytes (`go/token`) |
| Error surface | `MatchResult.Err()` returns `*MatchError`, which implements `error`; `nil` on success (section 6) | `Message()` string only; `Match` returns an error on mismatch (breaks every caller) |
| Message text | Same shape as ohm-js `message` and `shortMessage` (section 6) | A Go-specific shape |
| When expected tokens are computed | On first call, then cached; the result must be the most recent (section 8) | At `Match` time: a second parse on every failure, wanted or not |
| Start rule for the second parse | The rule that `Match` used (section 8) | `ruleNames[0]`, as ohm-js does: wrong when a start rule is given |
| Shape of a next valid token | `Expected{Kind, Text, Literal}`, with the kind parsed from the description (section 7) | Raw strings; an upstream table of kinds |
| Fluffy failures | Left out of `Expected()`; kept, marked, in `RightmostFailures()` (section 7) | Dropped |
| CLI format | `path:line:col: expected …`, then the excerpt (section 9) | ohm-js long message only |

## 1. Problem

A caller of `ohm` that gets a failed match has one bit: `Failed()`. Every
call site in `ohm-cli` turns that bit into `match failed`, with no
position. Six call sites do this: `sexpr/sexpr.go`, `candp/candp.go`,
`es5/roundtrip.go`, `ruleast/gen_go.go`, `ruleast/gen_go_types_tmpl.go` and
`ruleast/exec_grammar2ruleast.go`. A user with a 400-line grammar file
gets no line, no column and no hint. They bisect the file by hand.

ohm-js solves this with `getRightmostFailurePosition()`,
`getRightmostFailures()`, `getExpectedText()`, `message` and
`shortMessage` on `FailedMatchResult` (`packages/runtime/src/miniohm.ts`).
The Go runtime ports `Grammar` and `MatchResult` from that file, but not
these methods.

## 2. Users and use cases

1. **A user of `ohm-cli`** runs a command on a grammar or an input that
   does not parse. They want `file:line:col` so their editor jumps there.
2. **A Go program that embeds a grammar** wants to show its own user
   where their input is wrong, and in what way.
3. **A tool author** wants the next valid tokens at the failure point,
   to build hints, completions or a "did you mean" message.

Use cases 1 and 2 need Part A. Use case 3 needs Part B. Part A ships
first, because it is cheap and serves the most callers.

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
  position after it had already succeeded. ohm-js drops them from the
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
| `1 +\n (2 * x` | 11 | `an alpha-numeric character` (fluffy) |

The last row is open decision 11.1.

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
// failure. It mirrors the ohm-js method of the same name.
func (r *MatchResult) RightmostFailurePosition() int

// FailurePosition returns the rightmost failure as a Position.
func (r *MatchResult) FailurePosition() Position
```

Both methods are valid for the life of the result, also after `Close`,
because they read only Go memory. On a successful match they return the
end of the input. A caller must not read meaning into that.

The conversion from a UTF-16 offset to a `Position` is a pure function
in a new file, `ohm/position.go`. It takes the input string and the
offset, so it can be tested without Wasm.

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

For ASCII input, the column is the same as the ohm-js column. For input
with characters outside the Basic Multilingual Plane, the column is
smaller than the ohm-js column. `Position.Offset` keeps the ohm-js unit
for a caller that needs it.

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

The message text follows ohm-js, so that tests and users see one shape
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
	Text    string // the description, as ohm-js prints it
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

// ExpectedText joins Expected as ohm-js getExpectedText does:
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
copy is fluffy, as in ohm-js. Order is the order of first recording,
which is the order ohm-js prints.

## 8. Part B: lifetime and cost

`recordFailures` parses the input a second time. Three facts follow.

- **Most recent only.** The second parse uses the Wasm heap, the memo
  table and the grammar's current input. These belong to the most recent
  `MatchResult`. `RightmostFailures` returns an error when the result is
  not on top of `Grammar.resultStack`, or is closed. The error text
  follows ohm-js `_assertMostRecent`.
- **Cached.** The first call stores the failures on the `MatchResult`.
  Later calls, also after `Close`, return the cache.
- **Start rule.** `Match` stores the start rule id on the
  `MatchResult`, and `recordFailures` gets that id. ohm-js passes
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

## 9. ohm-cli

Each of the six call sites in section 1 replaces its
`match failed` error with an error that wraps `mr.Err()` and
names the input. The format is `path:line:col: expected …`, the form
that `go vet` and most editors parse. The `Message` excerpt follows on
the next lines. Where the input has no path, such as a string from a
txtar archive, the archive and member name take its place:
`tests/grammar_mixed_01.txtar:file1.txt:1:1: expected …`.

`test txtar` prints the same error for an input that does not parse.

## 10. Tests

- **Pure tests**, with no Wasm, for the offset-to-position function:
  ASCII, `\r\n`, a tab, an emoji, the end of input, an empty input, and
  an offset inside a surrogate pair.
- **Message tests** that compare `ShortMessage` and `Message` with
  strings produced by ohm-js for the same grammar and input. The
  expected strings go into the test file as constants. The test names
  the ohm-js version that made them.
- **Wasm tests** on `ohm/testdata/arith.wasm` for the four probe rows in
  section 3, for a start rule other than the default, for the
  most-recent rule, and for use after `Close`.
- **CLI tests**: one txtar fixture with an input that does not parse,
  run under `test txtar --strict`.

## 11. Open decisions

No `docs/kb/` page analyses these yet. Each one names the page that
should.

1. **An empty or fluffy-only expected set.** For `1 +\n (2 * x` on
   `arith`, the only failure recorded at the rightmost position is
   fluffy. `Expected()` is then empty, and the message reads
   `expected ` with nothing after it. The left-recursive rules
   `AddExp` and `MulExp` may hide `")"`, `"*"` and the rest. Check
   ohm-js on the same input. If ohm-js gives the same result, fall back
   to the fluffy failures when no other failure exists, and say so in
   the message. If ohm-js differs, the bug is upstream. Page to write:
   `docs/kb/ohm-failure-recording.md`.
2. **Column unit.** Code points (decided above) or UTF-16 units. A
   language server needs UTF-16 by default. `Position.Offset` covers
   that need today. Revisit if an LSP user appears.

## 12. Out of scope

- Next valid tokens at a cursor in input that matched, for completion
  in an editor. The Wasm records failures only at the rightmost failure
  position.
- More than one error per parse, or error recovery.
- An upstream change to the compiler, such as a table of failure kinds.
- `.ohm` grammar syntax errors from the Docker compiler. Those come from
  ohm-js, which already reports a position.
