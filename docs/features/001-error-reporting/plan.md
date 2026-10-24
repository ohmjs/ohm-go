# Error reporting — plan

This plan is the checklist for [prd.md](prd.md). That document is the
argument; this one is the work. Tick a bullet in the commit that lands
its code. Tick a bullet that is partly done as `[~]`, and say what is
missing.

**Next:** Phase 1, first bullet.

## Legend

- `[ ]` not started.
- `[~]` partly done. The bullet says what is missing.
- `[x]` done.
- `[N]` after a phase name gives the PRD sections that the phase builds.

## Before starting

- Read PRD sections 3, 8 and 10.
- Use the `go.work` at the repository root. `ohm-cli` then builds against the local `ohm`.
- Start docker. The default `.error` and `.expected` fixtures compile their grammar with it.
- Run `cd ohm-cli && go test ./...`. Expect a pass in about 5 seconds.
- Do not add tests to `ohm/`. All tests for this feature go in `ohm-cli`.

## Phase 1 — Position and error [4, 5, 6]

- [ ] `ohm/position.go`: add type `Position` with `Offset`, `ByteOffset`, `Line` and `Column`.
- [ ] `ohm/position.go`: add `positionAt(input string, utf16Offset int) Position`. Follow the rules in PRD section 5.
- [ ] `ohm/grammar.go`: add fields `rightmostFailurePos` and `startRuleId` to `MatchResult`.
- [ ] `ohm/grammar.go`: in `Match`, read the `rightmostFailurePos` global after the call to `match`. Store it.
- [ ] `ohm/grammar.go`: in `Match`, store the resolved start rule id.
- [ ] `ohm/grammar.go`: add `MatchResult.RightmostFailurePosition() int`.
- [ ] `ohm/grammar.go`: add `MatchResult.FailurePosition() Position`.
- [ ] `ohm/match_error.go`: add type `MatchError` with `Input`, `Pos` and `Expected`.
- [ ] `ohm/match_error.go`: add `MatchError.ShortMessage()`. Write `match failed` when `Expected` is nil.
- [ ] `ohm/match_error.go`: add `MatchError.Message()`. Show the line before, the failure line, the caret and the line after.
- [ ] `ohm/match_error.go`: in `Message()`, right-align line numbers to the widest number shown.
- [ ] `ohm/match_error.go`: add `MatchError.Error()`. Return `ShortMessage()`.
- [ ] `ohm/grammar.go`: add `MatchResult.Err() error`. Return nil on success.

**Done when** `cd ohm && go build ./...` passes. Phase 2 tests this code.

**Documentation** `ohm/README.md`: add a "Match failures" section. Show `FailurePosition` and `Err()`.

## Phase 2 — Error fixtures [10.1–10.5, 10.7]

- [ ] `ohm-cli/candp/candp.go`: move the body of `compile` into an exported `Compile(grammarFile string, keepWasm bool)`.
- [ ] `ohm-cli/candp/candp.go`: make `CompileAndParseCmd.compile` call `Compile`.
- [ ] `ohm-cli/matcherr/matcherr.go`: add type `MatchErrorCmd` with `Grammar`, `Start`, `Short` and `Source`.
- [ ] `ohm-cli/matcherr/matcherr.go`: in `Process`, use `candp.Compile` for `Grammar`. Use the embedded Ohm grammar when `Grammar` is empty.
- [ ] `ohm-cli/matcherr/matcherr.go`: in `Process`, print `Message()`, or `ShortMessage()` when `Short` is set.
- [ ] `ohm-cli/matcherr/matcherr.go`: in `Process`, print `match succeeded` when the match succeeds.
- [ ] `ohm-cli/matcherr/matcherr.go`: in `Run`, read `Source` from a file when it starts with `@`.
- [ ] `ohm-cli/main.go`: add the command `test match_error`.
- [ ] `ohm-cli/tests/tests.go`: add a parser for the flag part `start=<Rule>`.
- [ ] `ohm-cli/tests/tests.go`: in `testFile`, add the kind `error`. Read `<test>.txt`. Accept the flags `o`, `N`, `s` and `start=`.
- [ ] `ohm-cli/tests/tests.go`: in `testFile`, under flag `N`, remove one trailing `\n` from the input.
- [ ] `ohm-cli/tests/tests.go`: in `testFile`, write the grammar temp file only when the archive has a grammar.
- [ ] `ohm-cli/tests/tests.go`: in `testFile`, require a grammar only for a kind that uses it.
- [ ] `ohm-cli/tests/error_arith.txtar`: add the arith grammar and `.error` fixtures for `1 + * 2`, `abc 12` and empty input.
- [ ] `ohm-cli/tests/error_arith.txtar`: add a fixture with `start=PriExp`.
- [ ] `ohm-cli/tests/error_arith.txtar`: add a fixture with the `s` flag.
- [ ] `ohm-cli/tests/error_arith.txtar`: add a fixture for an input that matches. Expect `match succeeded`.
- [ ] `ohm-cli/tests/error_ohm_grammar.txtar`: add `o` fixtures for the three Ohm grammar inputs in PRD section 3.
- [ ] `ohm-cli/tests/error_ohm_grammar.txtar`: add an `o` fixture with a tab before the failure.
- [ ] `ohm-cli/tests/error_ohm_grammar.txtar`: add an `o` fixture with an emoji before the failure.
- [ ] `ohm-cli/tests/error_ohm_grammar.txtar`: add an `o` fixture that fails on line 10 or later. It tests number alignment.
- [ ] `ohm-cli/tests/error_ohm_grammar.txtar`: add an `o` fixture with the `N` flag at the end of input.
- [ ] `ohm-cli/tests/match_error_test.go`: test that `\r\n` and `\n` input give the same line and column.
- [ ] `ohm-cli/tests/match_error_test.go`: test that empty input gives line 1, column 1.
- [ ] `ohm-cli/tests/match_error_test.go`: test that `FailurePosition` gives the same value after `Close`.

**Done when** `cd ohm-cli && go test ./...` passes. `ohm-cli test txtar --strict tests/*.txtar` passes. `bad_op.s.error` reads `Line 1, col 5: match failed`.

**Documentation**
- `ohm-cli/docs/txtar-tests.md`: describe the archive format, every kind, every flag and `--strict`.
- `ohm-cli/README.md`: link `docs/txtar-tests.md`. Show `test match_error`.

## Phase 3 — Next valid tokens [7, 8]

- [ ] `ohm/expected.go`: add `ExpectedKind` and its four constants.
- [ ] `ohm/expected.go`: add `ExpectedKind.String()`. Return `literal`, `range`, `end` or `description`.
- [ ] `ohm/expected.go`: add types `Expected` and `Failure`.
- [ ] `ohm/expected.go`: add `classify(id int, text string) Expected`. Follow the four rules in PRD section 7.
- [ ] `ohm/grammar.go`: add field `failures []Failure` to `MatchResult`. It is the cache.
- [ ] `ohm/grammar.go`: add `MatchResult.RightmostFailures()`. Call `recordFailures` with the stored start rule id.
- [ ] `ohm/grammar.go`: in `RightmostFailures`, return an error when the result is closed or not the most recent.
- [ ] `ohm/grammar.go`: in `RightmostFailures`, merge duplicates by `Text`. Keep `Fluffy` only when every copy is fluffy.
- [ ] `ohm/grammar.go`: add `MatchResult.Expected()`. Return the non-fluffy failures.
- [ ] `ohm/grammar.go`: add `MatchResult.ExpectedText()`. Join as miniohm `getExpectedText` does.
- [ ] `ohm/grammar.go`: in `Err()`, fill `MatchError.Expected` when the result is the most recent.

**Done when** `cd ohm && go build ./...` passes. Phase 4 tests this code.

**Documentation** `ohm/README.md`: show `Expected()` and the fluffy rule in the "Match failures" section.

## Phase 4 — Next-token fixtures [10.2–10.4, 10.6, 10.8, 11.1]

- [ ] `ohm-cli/matcherr/matcherr.go`: add fields `Expected` and `Fluffy` to `MatchErrorCmd`.
- [ ] `ohm-cli/matcherr/matcherr.go`: in `Process`, under `Expected`, print one `<kind> <text>` line for each token.
- [ ] `ohm-cli/matcherr/matcherr.go`: in `Process`, under `Fluffy`, add fluffy failures. End each line with ` (fluffy)`.
- [ ] `ohm-cli/tests/tests.go`: in `testFile`, add the kind `expected`. Accept the flags `o`, `N`, `f` and `start=`.
- [ ] `ohm-cli/tests/error_arith.txtar`: add `.expected` fixtures for the four arith inputs in PRD section 3.
- [ ] `ohm-cli/tests/error_arith.txtar`: add a `.expected` fixture with `start=PriExp` for `x +`.
- [ ] `ohm-cli/tests/error_arith.txtar`: add a `.f.expected` fixture for `1 +\n (2 * x`.
- [ ] `ohm-cli/tests/error_arith.txtar`: change each `.error` fixture from `match failed` to the expected text.
- [ ] `ohm-cli/tests/error_ohm_grammar.txtar`: add `.o.expected` fixtures for the three Ohm grammar inputs.
- [ ] `ohm-cli/tests/error_ohm_grammar.txtar`: change each `.error` fixture from `match failed` to the expected text.
- [ ] `ohm-cli/tests/error_arith.txtar`: in the archive comment, name the miniohm version that the text matches.
- [ ] `ohm-cli/tests/error_ohm_grammar.txtar`: in the archive comment, name the miniohm version that the text matches.
- [ ] `ohm-cli/tests/match_error_test.go`: test the error from `RightmostFailures` after a second `Match`.
- [ ] `ohm-cli/tests/match_error_test.go`: test that `Expected()` after `Close` returns the cache.
- [ ] `ohm-cli/tests/match_error_test.go`: test that `Close` after `RightmostFailures` restores the heap watermark.
- [ ] `docs/kb/ohm-failure-recording.md`: run miniohm on the two inputs in open decision 11.1. Record the expected text.
- [ ] `ohm/grammar.go`: apply the result of open decision 11.1 to `Expected()`.
- [ ] `docs/features/001-error-reporting/prd.md`: close open decision 11.1. Move it to Decisions at a glance.

**Done when** `cd ohm-cli && go test ./...` passes. `ohm-cli test txtar --strict tests/*.txtar` passes. `bad_op.s.error` reads `Line 1, col 5: expected "(", an identifier, or a number`.

**Documentation**
- `ohm-cli/docs/txtar-tests.md`: add the `expected` kind and the `f` flag.
- `ohm-cli/README.md`: show `test match_error --expected`.

## Phase 5 — ohm-cli commands [9]

- [ ] `ohm-cli/sexpr/sexpr.go`: in `Process`, replace `match failed` with `mr.Err()`. Prefix the input name.
- [ ] `ohm-cli/candp/candp.go`: in `Process`, replace the `match failed for grammar` error with `mr.Err()`. Prefix the source path.
- [ ] `ohm-cli/es5/roundtrip.go`: in `Run`, replace `match failed` with `mr.Err()`. Prefix the input path.
- [ ] `ohm-cli/ruleast/gen_go.go`: in `Run`, replace `match failed` with `mr.Err()`. Prefix the grammar path.
- [ ] `ohm-cli/ruleast/gen_go_types_tmpl.go`: in `Process`, replace `match failed` with `mr.Err()`. Prefix the grammar path.
- [ ] `ohm-cli/ruleast/exec_grammar2ruleast.go`: in `Process`, replace `match failed` with `mr.Err()`. Prefix the grammar path.
- [ ] `ohm-cli/tests/tests.go`: in `RunTests`, print the `path:line:col` error for an expectation that fails to process.

**Done when** `cd ohm-cli && go test ./...` passes. `ohm-cli generate go` on a bad grammar prints `path:line:col: expected …`.

**Documentation** `ohm-cli/README.md`: add an example of the error output.

## Not in any phase

- `ohm/grammar_test.go`: `TestES5Match` fails, because `es5.wasm` is absent. Fix it apart from this feature.
- Tag `ohm` `v1.1.0`, then move `ohm-cli/go.mod` from `ohm v1.0.0` to it. Do this on "release".
- Next valid tokens at a cursor in input that matched. See PRD section 12.
- A precompiled `.wasm` member in a txtar archive. See PRD section 12.
- A fix to the `ruleNames[0]` start rule in miniohm `recordFailures`. Report it upstream.
- Load `ohm-cli/es5/es5.wasm`, which is `18.0.0-beta.16`. That needs `acceptedVersions` to change.
