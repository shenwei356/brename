package main

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestCodeString(t *testing.T) {
	tests := []struct {
		c        code
		contains string
	}{
		{codeOK, "OK"},
		{codeUnchanged, "unchanged"},
		{codeExisted, "new path existed"},
		{codeOverwriteNewPath, "overwriting newly renamed path"},
		{codeMissingTarget, "missing target"},
		{codeEndingWithSpace, "new path ending with a space"},
		{codeEndingWithPeriod, "new path ending with a period"},
		{code(999), "undefined code"},
	}

	for _, tt := range tests {
		s := tt.c.String()
		if !strings.Contains(s, tt.contains) {
			t.Errorf("code %d String() = %q, want containing %q", tt.c, s, tt.contains)
		}
	}
}

func TestCheckOperation_BasicReplacement(t *testing.T) {
	pathTree = make(map[string]struct{})

	opt := &Options{
		Pattern:     `file_(\d+)`,
		PatternRe:   regexp.MustCompile(`file_(\d+)`),
		Replacement: `doc_$1`,
	}

	ok, op := checkOperation(opt, filepath.Join("test_dir", "file_123.txt"))
	if !ok {
		t.Fatalf("expected checkOperation to return true")
	}

	expectedTarget := filepath.Join("test_dir", "doc_123.txt")
	if op.target != expectedTarget {
		t.Errorf("got target %q, want %q", op.target, expectedTarget)
	}
	if op.code != codeOK {
		t.Errorf("got code %v, want codeOK", op.code)
	}
}

func TestCheckOperation_IgnoreExt(t *testing.T) {
	pathTree = make(map[string]struct{})

	opt := &Options{
		Pattern:     `.*`,
		PatternRe:   regexp.MustCompile(`.*`),
		Replacement: `newname`,
		IgnoreExt:   true,
	}

	ok, op := checkOperation(opt, filepath.Join("folder", "sample.tar.gz"))
	if !ok {
		t.Fatalf("expected checkOperation to return true")
	}

	expectedTarget := filepath.Join("folder", "newname.gz")
	if op.target != expectedTarget {
		t.Errorf("got target %q, want %q", op.target, expectedTarget)
	}
}

func TestCheckOperation_NumberReplacement(t *testing.T) {
	pathTree = make(map[string]struct{})

	opt := &Options{
		Pattern:       `test`,
		PatternRe:     regexp.MustCompile(`test`),
		Replacement:   `file_{nr}`,
		ReplaceWithNR: true,
		StartNum:      1,
		NRFormat:      "%03d",
	}

	ok, op1 := checkOperation(opt, filepath.Join("dir", "test_a.txt"))
	if !ok || op1.code != codeOK {
		t.Fatalf("op1 failed: %v", op1)
	}
	expected1 := filepath.Join("dir", "file_001_a.txt")
	if op1.target != expected1 {
		t.Errorf("got target %q, want %q", op1.target, expected1)
	}

	ok, op2 := checkOperation(opt, filepath.Join("dir", "test_b.txt"))
	if !ok || op2.code != codeOK {
		t.Fatalf("op2 failed: %v", op2)
	}
	expected2 := filepath.Join("dir", "file_002_b.txt")
	if op2.target != expected2 {
		t.Errorf("got target %q, want %q", op2.target, expected2)
	}
}

func TestCheckOperation_KVReplacement(t *testing.T) {
	pathTree = make(map[string]struct{})

	opt := &Options{
		Pattern:       `sample_(\w+)`,
		PatternRe:     regexp.MustCompile(`sample_(\w+)`),
		Replacement:   `data_{kv}`,
		ReplaceWithKV: true,
		KeyCaptIdx:    1,
		KVs: map[string]string{
			"alpha": "A",
			"beta":  "B",
		},
	}

	ok, op := checkOperation(opt, "sample_alpha.txt")
	if !ok || op.code != codeOK {
		t.Fatalf("op failed: %v", op)
	}
	if op.target != "data_A.txt" {
		t.Errorf("got target %q, want data_A.txt", op.target)
	}

	// Missing key with KeepKey: true
	opt.KeepKey = true
	ok, opMissing := checkOperation(opt, "sample_gamma.txt")
	if !ok || opMissing.code != codeOK {
		t.Fatalf("opMissing failed: %v", opMissing)
	}
	if opMissing.target != "data_gamma.txt" {
		t.Errorf("got target %q, want data_gamma.txt", opMissing.target)
	}
}

func TestCheckOperation_ValidationEdgeCases(t *testing.T) {
	pathTree = make(map[string]struct{})

	// Target ending with period
	optPeriod := &Options{
		Pattern:     `txt`,
		PatternRe:   regexp.MustCompile(`txt`),
		Replacement: `txt.`,
	}
	_, opPeriod := checkOperation(optPeriod, "hello.txt")
	if opPeriod.code != codeEndingWithPeriod {
		t.Errorf("got code %v, want codeEndingWithPeriod", opPeriod.code)
	}

	// Target ending with space
	optSpace := &Options{
		Pattern:     `txt`,
		PatternRe:   regexp.MustCompile(`txt`),
		Replacement: `txt `,
	}
	_, opSpace := checkOperation(optSpace, "hello.txt")
	if opSpace.code != codeEndingWithSpace {
		t.Errorf("got code %v, want codeEndingWithSpace", opSpace.code)
	}

	// Unchanged
	optUnchanged := &Options{
		Pattern:     `foo`,
		PatternRe:   regexp.MustCompile(`foo`),
		Replacement: `foo`,
	}
	_, opUnchanged := checkOperation(optUnchanged, "foo.txt")
	if opUnchanged.code != codeUnchanged {
		t.Errorf("got code %v, want codeUnchanged", opUnchanged.code)
	}

	// Missing target (empty filename2)
	optEmpty := &Options{
		Pattern:     `.*`,
		PatternRe:   regexp.MustCompile(`.*`),
		Replacement: ``,
	}
	_, opEmpty := checkOperation(optEmpty, "foo")
	if opEmpty.code != codeMissingTarget {
		t.Errorf("got code %v, want codeMissingTarget", opEmpty.code)
	}
}

func TestCheckOperation_CollisionDetection(t *testing.T) {
	pathTree = make(map[string]struct{})

	opt := &Options{
		Pattern:     `foo\d+`,
		PatternRe:   regexp.MustCompile(`foo\d+`),
		Replacement: `same_target`,
	}

	ok1, op1 := checkOperation(opt, "foo1.txt")
	if !ok1 || op1.code != codeOK {
		t.Fatalf("first op should be OK, got %v", op1.code)
	}

	// Second file renaming to the same target should trigger codeOverwriteNewPath
	ok2, op2 := checkOperation(opt, "foo2.txt")
	if !ok2 || op2.code != codeOverwriteNewPath {
		t.Errorf("second op should be codeOverwriteNewPath, got %v", op2.code)
	}
}

func TestIgnore_FilterPrecedence(t *testing.T) {
	opt := &Options{
		SkipFilterRes:    []*regexp.Regexp{regexp.MustCompile(`^\.`)},
		ExcludeFilterRes: []*regexp.Regexp{regexp.MustCompile(`\.bak$`)},
		IncludeFilterRes: []*regexp.Regexp{regexp.MustCompile(`\.txt$`)},
	}

	// Hidden file matched by SkipFilterRes should be ignored
	if !ignore(opt, ".hidden.txt") {
		t.Errorf("expected .hidden.txt to be ignored")
	}

	// .bak matched by ExcludeFilterRes should be ignored
	if !ignore(opt, "test.bak") {
		t.Errorf("expected test.bak to be ignored")
	}

	// .txt matched by IncludeFilterRes should NOT be ignored
	if ignore(opt, "valid.txt") {
		t.Errorf("expected valid.txt NOT to be ignored")
	}

	// File not in IncludeFilterRes should be ignored
	if !ignore(opt, "other.csv") {
		t.Errorf("expected other.csv to be ignored")
	}
}
