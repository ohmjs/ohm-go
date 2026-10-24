# Ohm failure recording in Wasm grammars

How miniohm, the ohm-js runtime for Wasm grammars, reports the failures
at the rightmost failure position, and where the Go runtime agrees. This
page is the analysis behind error-reporting prd open decision 11.1: an
empty or a short expected set.

**Answer.** The Go runtime gives the same failures as miniohm, in the
same order, with the same fluffy flags, on every input checked. A short
expected set, such as `"}"` alone for `G {\n  S = *x\n}`, is what
miniohm gives too. An empty expected set happens in miniohm as well, and
miniohm then prints `expected ` with nothing after it. The arith input
of the PRD, `1 +\n (2 * x`, cannot be checked: `recordFailures` traps
on it, in miniohm and in Go.

Checked on 2026-10-25 with miniohm from the `ohmjs/ohm:18.0.0-beta.15`
docker image. Its runtime `package.json` says `ohm-js`
`18.0.0-beta.13`; the file is
`/ohm/packages/runtime/dist/miniohm-BkWK2lEN.js`. The Wasm modules are
`examples/generic-methods/arith.wasm` and
`ohm-cli/utils/ohm-grammar.wasm`, both from compiler `18.0.0-beta.15`.

## 1. How miniohm gets the failures

`FailedMatchResult.getRightmostFailures()` calls
`exports.recordFailures(inputLength, ruleIds.get(ruleNames[0]))`. It
then reads `getRecordedFailuresLength()`, `recordedFailuresAt(i)` and
`isFluffy(i)`. It merges the failures by description in a `Map`. A
merged failure is fluffy only when every copy is fluffy. The order is
the order of first recording.

`getExpectedText()` keeps the non-fluffy failures only. It joins them as
`a`, `a or b`, `a, b, or c`. For no failure it returns the empty string,
so `shortMessage` reads `Line 1, col 6: expected ` and `message` ends in
`Expected `. There is no fall back to the fluffy failures.

miniohm passes `ruleNames[0]`, not the start rule of the match. For
arith, `ruleNames[0]` is `Exp`. On `x +` with start rule `PriExp`,
miniohm still gives `end of input`, the same as Go, which passes
`PriExp`.

## 2. The two inputs of open decision 11.1

Each input runs on a fresh `Grammar`.

| grammar | input | position | non-fluffy | fluffy, in order |
|---|---|---|---|---|
| Ohm | `G {\n  S = *x\n}` | 10 (line 2, col 7) | `"}"` | `"\|"`, `"~"`, `"&"`, `"#"`, `an identifier`, `"\""`, `"("`, `"--"` |
| arith | `1 +\n (2 * x` | 11 | trap | trap |

On the Ohm input, Go gives the same nine failures in the same order.
The PRD's earlier count of "about 200, over 15 distinct texts" was
before the merge by description. Classic ohm-js documents a longer list
for this input in `doc/errors.md`; that difference is upstream.

On the arith input, miniohm throws `RangeError: Maximum call stack size
exceeded`, and Go gets `error calling recordFailures: stack overflow`.

## 3. The trap

`recordFailures` traps on many arith inputs, not only those with an
unclosed `(`. A Go run over every input of 1 to 4 characters from
`1 x + * ( ) space \n $`, 7380 inputs, each on a fresh `Grammar`, found
1442 traps. Examples: `1)`, `(x`, `(2`, `1 + (2`, `1 +\n 2 * x$`,
`1 +\n 23$`. On a fresh `Grammar`, miniohm throws `Maximum call stack
size exceeded` on each example. On a `Grammar` that trapped before, it
can throw `Error: Wasm abort:  at :114:42` instead. Inputs that do not
trap include `1 )`, `(1)$`, `((1)` and `(1 +\n (2 * x)`.

Only 2 of the 7380 inputs, `((1)` and `((x)`, have a fluffy failure and
do not trap. None has an empty expected set. The arith fixtures use
`(1 +\n (2 * x)`, with no trailing newline, as the multi-line input with
fluffy failures:

| input | position | non-fluffy | fluffy |
|---|---|---|---|
| `(1 +\n (2 * x)` | 13 (line 2, col 9) | `")"` | `"*"`, `"/"`, `"+"`, `"-"` |
| `(1 +\n (2 * x)\n` | 14 (line 3, col 1) | `"*"`, `"/"`, `"+"`, `"-"`, `")"` | none |

## 4. Earlier matches change the result

The failures can depend on earlier matches on the same `Grammar`. On a
fresh arith `Grammar`, `1+(1*` traps. After a match of `1+(1+` on the
same `Grammar`, `1+(1*` does not trap, and every failure is fluffy:

```
"1+(1+" pos 5 ["\"(\"","an identifier","a number"]
"1+(1*" pos 5 ["F \"(\"","F an identifier","F a number"] "Line 1, col 6: expected "
```

miniohm and Go give the same output for this sequence. It is the only
empty expected set found. `ohm-cli test match_error` makes a new
`Grammar` for each input, so the txtar fixtures do not see this.

## 5. Decision for 11.1

`Expected()` keeps the miniohm rule: the non-fluffy failures, and no
fall back to the fluffy ones when none is left. The reasons:

- Go agrees with miniohm on every input checked, and section 10.8 of
  the PRD asks for that parity.
- No empty expected set was found on a fresh `Grammar`. The one found
  comes from the state of earlier matches, a bug upstream.
- The fluffy set is noisy where it was checked: eight texts beside
  `"}"` on the Ohm input. A fall back would hide the bug, not fix it.

A caller that wants the fluffy failures calls `RightmostFailures()`.
The arith half stays open until the compiler stops trapping on
`1 +\n (2 * x`. Then run section 6 on it again.

## 6. How to check

The script prints `message`, `shortMessage` and the ordered failures for
each case, each on a fresh `Grammar`. Put the Wasm modules and the
script in one directory, `<dir>`:

```js
// msg.mjs: node msg.mjs <cases.json>; cases: [{wasm, input, start?}]
import {Grammar} from '/ohm/packages/runtime/dist/index.js';
import fs from 'node:fs';
const cases = JSON.parse(fs.readFileSync(process.argv[2], 'utf8'));
for (const c of cases) {
  const g = await Grammar.instantiate(fs.readFileSync(c.wasm));
  console.log('=== ' + JSON.stringify(c.input) + (c.start ? ' start=' + c.start : ''));
  const r = c.start ? g.match(c.input, c.start) : g.match(c.input);
  try {
    if (r.succeeded()) { console.log('match succeeded'); continue; }
    console.log(r.message); console.log('--'); console.log(r.shortMessage); console.log('--');
    for (const f of r.getRightmostFailures()) console.log(String(f) + (f.isFluffy() ? ' (fluffy)' : ''));
  } catch (e) { console.log('ERROR ' + String(e).slice(0, 200)); }
}
```

```sh
docker run --rm -v <dir>:/w --entrypoint node ohmjs/ohm:18.0.0-beta.15 \
  /w/msg.mjs /w/cases.json
```

with `cases.json` such as
`[{"wasm":"/w/arith.wasm","input":"(1 +\n (2 * x)"}]`. The Go side is
`ohm-cli test match_error --grammar arith.ohm --expected --fluffy
$'(1 +\n (2 * x)'`, run from `ohm-cli`. For section 4, match the inputs
in order on one `Grammar` and call `dispose()` after each.
