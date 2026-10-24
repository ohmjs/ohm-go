package ohm

import (
	"encoding/json"
	"strings"
)

// ExpectedKind classifies one next valid token.
// See error-reporting prd section 7.
type ExpectedKind int

const (
	ExpectedLiteral     ExpectedKind = iota // a terminal, such as "("
	ExpectedRange                           // a range, such as "a".."z"
	ExpectedEnd                             // end of input
	ExpectedDescription                     // a rule or its description, such as "a number"
)

// String returns literal, range, end or description.
// The .expected txtar fixtures use these words.
func (k ExpectedKind) String() string {
	switch k {
	case ExpectedLiteral:
		return "literal"
	case ExpectedRange:
		return "range"
	case ExpectedEnd:
		return "end"
	case ExpectedDescription:
		return "description"
	}
	return "unknown"
}

// Expected is one thing the grammar accepts at the failure position.
type Expected struct {
	Kind    ExpectedKind
	Text    string // the description, as miniohm prints it
	Literal string // the unquoted terminal; set for ExpectedLiteral only
}

// Failure is one recorded failure at the rightmost position.
// Fluffy failures come from a repetition that failed at the failure
// position after it had already succeeded. miniohm leaves them out of
// the expected text.
type Failure struct {
	Expected
	Fluffy bool
}

// endOfInputText is the description of failure id 0, and of the end rule.
const endOfInputText = "end of input"

// classify makes an Expected from a failure id and its description.
// The Wasm records only a string id, so the kind comes from the text, as
// toFailureDescription in the compiler writes it. The rules, in order:
//  1. id 0, or the text "end of input": ExpectedEnd;
//  2. `"x".."y"`, both sides valid JSON strings: ExpectedRange;
//  3. one valid JSON string: ExpectedLiteral, with the decoded Literal;
//  4. anything else: ExpectedDescription.
func classify(id int, text string) Expected {
	if id == 0 || text == endOfInputText {
		return Expected{Kind: ExpectedEnd, Text: text}
	}
	lo, n, ok := jsonStringPrefix(text)
	if !ok {
		return Expected{Kind: ExpectedDescription, Text: text}
	}
	if n == len(text) {
		return Expected{Kind: ExpectedLiteral, Text: text, Literal: lo}
	}
	if rest, found := strings.CutPrefix(text[n:], ".."); found {
		if _, m, ok := jsonStringPrefix(rest); ok && m == len(rest) {
			return Expected{Kind: ExpectedRange, Text: text}
		}
	}
	return Expected{Kind: ExpectedDescription, Text: text}
}

// jsonStringPrefix decodes the JSON string at the start of text. It
// returns the decoded string and the number of bytes it takes.
func jsonStringPrefix(text string) (string, int, bool) {
	if !strings.HasPrefix(text, `"`) {
		return "", 0, false
	}
	// Find the closing quote; a backslash escapes the next byte.
	for i := 1; i < len(text); i++ {
		switch text[i] {
		case '\\':
			i++
		case '"':
			var s string
			if err := json.Unmarshal([]byte(text[:i+1]), &s); err != nil {
				return "", 0, false
			}
			return s, i + 1, true
		}
	}
	return "", 0, false
}
