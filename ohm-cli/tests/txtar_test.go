package tests

import (
	"path/filepath"
	"testing"
)

// TestTxtar drives every *.txtar fixture in this directory through
// testGmrCmd.TestFile, the same entry point the CLI test command uses. Each
// fixture becomes a subtest so failures are attributed to individual files and
// `go test -run TestTxtar/grammar_mixed_01` can target a single one.
func TestTxtar(t *testing.T) {
	files, err := filepath.Glob("*.txtar")
	if err != nil {
		t.Fatalf("globbing txtar fixtures: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no *.txtar fixtures found")
	}
	for _, f := range files {
		t.Run(f, func(t *testing.T) {
			t.Parallel()
			if err := TestFile("", f); err != nil {
				t.Fatalf("TestFile(%q): %v", f, err)
			}
		})
	}
}
