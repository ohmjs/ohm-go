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

- Read PRD sections 0, 3 and 8.
- Run `cd ohm && go test ./...`. Expect one failure: `TestES5Match`, no `es5.wasm`.
- Run `cd ohm-cli && go build ./...`. Expect no errors.
- Use the `go.work` at the repository root. `ohm-cli` then builds against the local `ohm`.

## Phase 1 — Test fixture [0]

- [ ] `ohm/testdata/arith.wasm`: copy `examples/generic-methods/arith.wasm` to this path.
- [ ] `ohm/testdata/README.md`: say where `arith.wasm` comes from and which compiler version made it.
- [ ] `ohm/grammar_test.go`: in `TestES5Match`, call `t.Skip` when `es5.wasm` is absent and `OHM_WASM` is unset.
- [ ] `ohm/grammar_test.go`: in `BenchmarkES5Match`, call `b.Skip` under the same condition.
- [ ] `ohm/fixture_test.go`: add `loadArith(t)`, which returns a `*Grammar` from `testdata/arith.wasm`.
- [ ] `docs/features/001-error-reporting/prd.md`: remove the item from section 0. Move the status paragraph.

**Done when** `cd ohm && go test ./...` passes.

**Documentation** None.

## Phase 2 — Position [4, 5]

- [ ] `ohm/position.go`: add type `Position` with `Offset`, `ByteOffset`, `Line` and `Column`.
- [ ] `ohm/position.go`: add `positionAt(input string, utf16Offset int) Position`. Follow the rules in PRD section 5.
- [ ] `ohm/position_test.go`: test `positionAt` with ASCII, `\r\n`, a tab and an emoji.
- [ ] `ohm/position_test.go`: test `positionAt` at the end of input, on empty input, and inside a surrogate pair.
- [ ] `ohm/grammar.go`: add fields `rightmostFailurePos` and `startRuleId` to `MatchResult`.
- [ ] `ohm/grammar.go`: in `Match`, read the `rightmostFailurePos` global after the call to `match`. Store it.
- [ ] `ohm/grammar.go`: in `Match`, store the resolved start rule id.
- [ ] `ohm/grammar.go`: add `MatchResult.RightmostFailurePosition() int`.
- [ ] `ohm/grammar.go`: add `MatchResult.FailurePosition() Position`.
- [ ] `ohm/failure_test.go`: test the four inputs in PRD section 3. Check the offset, line and column.
- [ ] `ohm/failure_test.go`: test that `FailurePosition` gives the same value after `Close`.

**Done when** `cd ohm && go test ./...` passes. `FailurePosition` gives line 1, column 5 for `1 + * 2`.

**Documentation** `ohm/README.md`: add a "Match failures" section. Show `FailurePosition`.

## Phase 3 — Error and message [6]

- [ ] `ohm/match_error.go`: add type `MatchError` with `Input`, `Pos` and `Expected`.
- [ ] `ohm/match_error.go`: add `MatchError.ShortMessage()`. Write `match failed` when `Expected` is nil.
- [ ] `ohm/match_error.go`: add `MatchError.Message()`. Show the line before, the failure line, the caret and the line after.
- [ ] `ohm/match_error.go`: add `MatchError.Error()`. Return `ShortMessage()`.
- [ ] `ohm/grammar.go`: add `MatchResult.Err() error`. Return nil on success.
- [ ] `ohm/match_error_test.go`: compare `Message()` with the ohm-js `getLineAndColumnMessage` output for three inputs.
- [ ] `ohm/match_error_test.go`: name the ohm-js version that made the expected strings.
- [ ] `ohm/match_error_test.go`: test line-number alignment across a change in digit count, such as lines 9 to 11.

**Done when** `cd ohm && go test ./...` passes. `Err()` on `1 + * 2` gives `Line 1, col 5: match failed`.

**Documentation** `ohm/README.md`: show `Err()` in the "Match failures" section.

## Phase 4 — Next valid tokens [7, 8]

- [ ] `ohm/expected.go`: add `ExpectedKind` and its four constants.
- [ ] `ohm/expected.go`: add types `Expected` and `Failure`.
- [ ] `ohm/expected.go`: add `classify(id int, text string) Expected`. Follow the four rules in PRD section 7.
- [ ] `ohm/expected_test.go`: test `classify` with a literal, a range, `end of input` and a description.
- [ ] `ohm/grammar.go`: add field `failures []Failure` to `MatchResult`. It is the cache.
- [ ] `ohm/grammar.go`: add `MatchResult.RightmostFailures()`. Call `recordFailures` with the stored start rule id.
- [ ] `ohm/grammar.go`: in `RightmostFailures`, return an error when the result is closed or not the most recent.
- [ ] `ohm/grammar.go`: in `RightmostFailures`, merge duplicates by `Text`. Keep `Fluffy` only when every copy is fluffy.
- [ ] `ohm/grammar.go`: add `MatchResult.Expected()`. Return the non-fluffy failures.
- [ ] `ohm/grammar.go`: add `MatchResult.ExpectedText()`. Join as ohm-js `getExpectedText` does.
- [ ] `ohm/grammar.go`: in `Err()`, fill `MatchError.Expected` when the result is the most recent.
- [ ] `ohm/failure_test.go`: test `Expected()` on the four inputs in PRD section 3.
- [ ] `ohm/failure_test.go`: test `Expected()` with a start rule other than the default.
- [ ] `ohm/failure_test.go`: test the error from `RightmostFailures` after a second `Match`.
- [ ] `ohm/failure_test.go`: test that `Expected()` after `Close` returns the cache.
- [ ] `ohm/failure_test.go`: test that `Close` after `RightmostFailures` restores the heap watermark.
- [ ] `docs/kb/ohm-failure-recording.md`: run ohm-js on `1 +\n (2 * x` with `arith.ohm`. Record its expected text.
- [ ] `ohm/grammar.go`: apply the result of open decision 11.1 to `Expected()`.
- [ ] `docs/features/001-error-reporting/prd.md`: close open decision 11.1. Move it to Decisions at a glance.

**Done when** `cd ohm && go test ./...` passes. `Err()` on `1 + * 2` gives `Line 1, col 5: expected "(", an identifier, or a number`.

**Documentation** `ohm/README.md`: show `Expected()` and the fluffy rule in the "Match failures" section.

## Phase 5 — ohm-cli [9]

- [ ] `ohm-cli/sexpr/sexpr.go`: in `Process`, replace `match failed` with `mr.Err()`. Prefix the input name.
- [ ] `ohm-cli/candp/candp.go`: in `Process`, replace the `match failed for grammar` error with `mr.Err()`. Prefix the source path.
- [ ] `ohm-cli/es5/roundtrip.go`: in `Run`, replace `match failed` with `mr.Err()`. Prefix the input path.
- [ ] `ohm-cli/ruleast/gen_go.go`: in `Run`, replace `match failed` with `mr.Err()`. Prefix the grammar path.
- [ ] `ohm-cli/ruleast/gen_go_types_tmpl.go`: in `Process`, replace `match failed` with `mr.Err()`. Prefix the grammar path.
- [ ] `ohm-cli/ruleast/exec_grammar2ruleast.go`: in `Process`, replace `match failed` with `mr.Err()`. Prefix the grammar path.
- [ ] `ohm-cli/tests/tests.go`: in `RunTests`, print the `path:line:col` error for an input that does not parse.
- [ ] `ohm-cli/tests/grammar_error_01.txtar`: add a grammar and an input that does not parse.
- [ ] `ohm-cli/tests/txtar_test.go`: check that `grammar_error_01.txtar` reports line and column under `--strict`.

**Done when** `cd ohm-cli && go test ./...` passes. A bad grammar file gives `path:line:col: expected …`.

**Documentation** `ohm-cli/README.md`: add an example of the error output.

## Not in any phase

- Tag `ohm` `v1.1.0`, then move `ohm-cli/go.mod` from `ohm v1.0.0` to it. Do this on "release".
- Next valid tokens at a cursor in input that matched. See PRD section 12.
- A fix to the `ruleNames[0]` start rule in ohm-js `recordFailures`. Report it upstream.
- Load `ohm-cli/es5/es5.wasm`, which is `18.0.0-beta.16`. That needs `acceptedVersions` to change.
