package ohm

// ExpectedKind classifies one next valid token.
// See error-reporting prd section 7.
type ExpectedKind int

// Expected is one thing the grammar accepts at the failure position.
type Expected struct {
	Kind    ExpectedKind
	Text    string // the description, as miniohm prints it
	Literal string // the unquoted terminal; set for ExpectedLiteral only
}
