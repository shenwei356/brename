package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
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

// TestCLIProcess runs the actual command in a child process, including exit paths.
func TestCLIProcess(t *testing.T) {
	if os.Getenv("BRENAME_TEST_HELPER") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			RootCmd.SetArgs(os.Args[i+1:])
			main()
			os.Exit(0)
		}
	}
	os.Exit(2)
}

// runCLI isolates flags, global state and filesystem changes for command tests.
func runCLI(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=^TestCLIProcess$", "--"}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "BRENAME_TEST_HELPER=1", "GORACE=atexit_sleep_ms=0")
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("command timed out: %v\n%s", args, out)
	}
	return string(out), err
}

// writeTestFile creates fixture contents without invoking the command under test.
func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

// requireFile checks contents as well as existence to catch unintended overwrites.
func requireFile(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil || string(got) != want {
		t.Fatalf("%s: content %q, error %v; want %q", path, got, err, want)
	}
}

func TestCLIExplicitPaths(t *testing.T) {
	for _, root := range []string{".", "./", ".."} {
		t.Run(root, func(t *testing.T) {
			dir := t.TempDir()
			cwd := dir
			if root == ".." {
				cwd = filepath.Join(dir, "child")
				if err := os.Mkdir(cwd, 0700); err != nil {
					t.Fatal(err)
				}
			}
			writeTestFile(t, filepath.Join(dir, "a"), "original")
			if out, err := runCLI(t, cwd, "-p", "^a$", "-r", "b", root); err != nil {
				t.Fatalf("%v: %s", err, out)
			}
			requireFile(t, filepath.Join(dir, "b"), "original")
		})
	}
	t.Run("trailing separator", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, "foo"), 0700); err != nil {
			t.Fatal(err)
		}
		if out, err := runCLI(t, dir, "-D", "-p", "foo", "-r", "bar", "foo"+string(os.PathSeparator)); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
		if info, err := os.Stat(filepath.Join(dir, "bar")); err != nil || !info.IsDir() {
			t.Fatalf("directory not renamed: %v", err)
		}
	})
	t.Run("only directories", func(t *testing.T) {
		dir := t.TempDir()
		writeTestFile(t, filepath.Join(dir, "a"), "original")
		if out, err := runCLI(t, dir, "--only-dir", "-p", "a", "-r", "b", "a"); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
		requireFile(t, filepath.Join(dir, "a"), "original")
	})
}

func TestCLIRecursiveFilters(t *testing.T) {
	for _, includeDirs := range []bool{false, true} {
		t.Run(fmt.Sprint(includeDirs), func(t *testing.T) {
			dir := t.TempDir()
			for _, name := range []string{"sub", ".hidden", "excluded"} {
				if err := os.Mkdir(filepath.Join(dir, name), 0700); err != nil {
					t.Fatal(err)
				}
				writeTestFile(t, filepath.Join(dir, name, "a.txt"), name)
			}
			args := []string{"-R", "-f", `\.txt$`, "-F", "^excluded$", "-p", "a", "-r", "b"}
			if includeDirs {
				args = append(args, "-D")
			}
			if out, err := runCLI(t, dir, args...); err != nil {
				t.Fatalf("%v: %s", err, out)
			}
			requireFile(t, filepath.Join(dir, "sub", "b.txt"), "sub")
			requireFile(t, filepath.Join(dir, ".hidden", "a.txt"), ".hidden")
			requireFile(t, filepath.Join(dir, "excluded", "a.txt"), "excluded")
			if out, err := runCLI(t, dir, "-R", "-S", "", "-p", "^a", "-r", "b"); err != nil {
				t.Fatalf("%v: %s", err, out)
			}
			requireFile(t, filepath.Join(dir, ".hidden", "b.txt"), ".hidden")
		})
	}
}

func TestCLIListAbsConcurrent(t *testing.T) {
	dir := t.TempDir()
	const count = 1800 // Exceed the operation channel's capacity to exercise both goroutines.
	for i := 0; i < count; i++ {
		writeTestFile(t, filepath.Join(dir, fmt.Sprintf("f%04d", i)), "")
	}
	out, err := runCLI(t, dir, "-l", "-a", "-p", ".")
	if err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != count {
		t.Fatalf("got %d paths, want %d", len(lines), count)
	}
	for i, line := range lines {
		if line != filepath.Join(dir, fmt.Sprintf("f%04d", i)) {
			t.Fatalf("unexpected listed path: %q", line)
		}
	}
}

func TestCLIUnreadableFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file permission fixture")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "a")
	writeTestFile(t, path, "original")
	if err := os.Chmod(path, 0); err != nil {
		t.Fatal(err)
	}
	if out, err := runCLI(t, dir, "-p", "a", "-r", "b", "a"); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if err := os.Chmod(filepath.Join(dir, "b"), 0600); err != nil {
		t.Fatal(err)
	}
	requireFile(t, filepath.Join(dir, "b"), "original")
}

func TestReadKVs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kv")
	writeTestFile(t, path, "A\tfirst\r\nignored\nA\tlast\textra\nB\tsecond")
	kvs, err := readKVs(path, true)
	if err != nil || len(kvs) != 2 || kvs["a"] != "last" || kvs["b"] != "second" {
		t.Fatalf("map %v, error %v", kvs, err)
	}
	var compressed bytes.Buffer
	w := gzip.NewWriter(&compressed)
	if _, err := w.Write([]byte("a\tb\n")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	for _, corrupt := range []bool{false, true} {
		data := append([]byte(nil), compressed.Bytes()...)
		if corrupt {
			data = data[:len(data)-8]
		}
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		kvs, err = readKVs(path, false)
		if corrupt && err == nil {
			t.Fatal("accepted truncated gzip")
		}
		if !corrupt && (err != nil || kvs["a"] != "b") {
			t.Fatalf("map %v, error %v", kvs, err)
		}
	}
	writeTestFile(t, filepath.Join(dir, "a"), "original")
	if out, err := runCLI(t, dir, "-p", "^(a)$", "-r", "{kv}", "-k", "kv", "a"); err == nil {
		t.Fatalf("accepted corrupt mapping: %s", out)
	}
	requireFile(t, filepath.Join(dir, "a"), "original")
}

func TestCLINaturalNumbering(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"f1", "f2", "f10"} {
		writeTestFile(t, filepath.Join(dir, name), name)
	}
	if out, err := runCLI(t, dir, "-N", "-p", `f\d+`, "-r", "{nr}"); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	for i, name := range []string{"f1", "f2", "f10"} {
		requireFile(t, filepath.Join(dir, fmt.Sprint(i+1)), name)
	}
}
