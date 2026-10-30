package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// writeTestHistory publishes a staged log for format and legacy-reader fixtures.
func writeTestHistory(t *testing.T, path string, history []operation) {
	t.Helper()
	fh, err := stageHistory(path, history, historyNeedsQuoting(history))
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(fh.Name())
	if err := fh.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(fh.Name(), path); err != nil {
		t.Fatal(err)
	}
}

func TestHistoryRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, LastOpDetailFile)
	// Include bytes that JSON strings cannot preserve, in addition to separators.
	history := []operation{{source: "#a\n\t\xff", target: "b\r\n" + historyDelimiter}}
	writeTestHistory(t, path, history)
	got, err := readHistory(path)
	if err != nil || len(got) != 1 || got[0] != history[0] {
		t.Fatalf("history %#v, error %v", got, err)
	}
	writeTestFile(t, path, "# old log\n#a"+historyDelimiter+"#b\n")
	got, err = readHistory(path)
	if err != nil || len(got) != 1 || got[0].source != "#a" || got[0].target != "#b" {
		t.Fatalf("legacy history %#v, error %v", got, err)
	}
	writeTestFile(t, path, "a"+historyDelimiter+"b\nbroken record\n")
	if _, err := readHistory(path); err == nil {
		t.Fatal("accepted malformed history")
	}
}

func TestCLIUndoSpecialNames(t *testing.T) {
	names := []string{"#a"}
	if runtime.GOOS != "windows" {
		names = append(names, "a\nx", "a\t_shenwei356-brename_\tx", "a\xff")
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writeTestFile(t, filepath.Join(dir, name), "original")
			if out, err := runCLI(t, dir, "-p", "a", "-r", "b", name); err != nil {
				t.Fatalf("%v: %s", err, out)
			}
			if out, err := runCLI(t, dir, "-u"); err != nil {
				t.Fatalf("%v: %s", err, out)
			}
			requireFile(t, filepath.Join(dir, name), "original")
		})
	}
}

func TestCLIUndoOverwriteAndRetry(t *testing.T) {
	for _, force := range []bool{false, true} {
		t.Run(map[bool]string{false: "normal", true: "force"}[force], func(t *testing.T) {
			dir := t.TempDir()
			writeTestFile(t, filepath.Join(dir, "a"), "original")
			if out, err := runCLI(t, dir, "-p", "a", "-r", "b"); err != nil {
				t.Fatalf("%v: %s", err, out)
			}
			writeTestFile(t, filepath.Join(dir, "a"), "new")
			flag := "-u"
			if force {
				flag = "-U"
			}
			out, err := runCLI(t, dir, flag)
			if err == nil || strings.Contains(out, "[DONE]") {
				t.Fatalf("overwrite accepted: %v: %s", err, out)
			}
			requireFile(t, filepath.Join(dir, "a"), "new")
			requireFile(t, filepath.Join(dir, "b"), "original")
			if err := os.Remove(filepath.Join(dir, "a")); err != nil {
				t.Fatal(err)
			}
			if out, err := runCLI(t, dir, "-u"); err != nil {
				t.Fatalf("retry: %v: %s", err, out)
			}
			requireFile(t, filepath.Join(dir, "a"), "original")
		})
	}
}

func TestCLIUndoPartialFailure(t *testing.T) {
	for _, flag := range []string{"-u", "-U"} {
		t.Run(flag, func(t *testing.T) {
			dir := t.TempDir()
			writeTestFile(t, filepath.Join(dir, "a1"), "one")
			writeTestFile(t, filepath.Join(dir, "a2"), "two")
			writeTestFile(t, filepath.Join(dir, "a3"), "three")
			if out, err := runCLI(t, dir, "-p", "^a", "-r", "b"); err != nil {
				t.Fatalf("%v: %s", err, out)
			}
			if err := os.Remove(filepath.Join(dir, "b2")); err != nil {
				t.Fatal(err)
			}
			out, err := runCLI(t, dir, flag)
			if err == nil {
				t.Fatalf("failed undo returned success: %s", out)
			}
			requireFile(t, filepath.Join(dir, "a3"), "three")
			remaining, err := readHistory(filepath.Join(dir, LastOpDetailFile))
			want := 2
			if flag == "-U" {
				want = 1
				requireFile(t, filepath.Join(dir, "a1"), "one")
			}
			if err != nil || len(remaining) != want {
				t.Fatalf("remaining %#v, error %v", remaining, err)
			}
			writeTestFile(t, filepath.Join(dir, "b2"), "two")
			if out, err := runCLI(t, dir, "-u"); err != nil {
				t.Fatalf("retry: %v: %s", err, out)
			}
			requireFile(t, filepath.Join(dir, "a1"), "one")
			requireFile(t, filepath.Join(dir, "a2"), "two")
		})
	}
}

func TestCLIUndoQuotedPartialFailure(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"#a1", "#a2"} {
		writeTestFile(t, filepath.Join(dir, name), name)
	}
	if out, err := runCLI(t, dir, "-p", "a", "-r", "b", "#a1", "#a2"); err != nil {
		t.Fatalf("rename: %v: %s", err, out)
	}
	writeTestFile(t, filepath.Join(dir, "#a1"), "conflict")
	if out, err := runCLI(t, dir, "-U"); err == nil {
		t.Fatalf("expected conflict: %s", out)
	}
	requireFile(t, filepath.Join(dir, "#a2"), "#a2")
	history, err := readHistory(filepath.Join(dir, LastOpDetailFile))
	if err != nil || len(history) != 1 || history[0].source != "#a1" {
		t.Fatalf("remaining quoted history: %#v, %v", history, err)
	}
	if err := os.Remove(filepath.Join(dir, "#a1")); err != nil {
		t.Fatal(err)
	}
	if out, err := runCLI(t, dir, "-u"); err != nil {
		t.Fatalf("retry: %v: %s", err, out)
	}
	requireFile(t, filepath.Join(dir, "#a1"), "#a1")
}

func TestCLIPartialRenameHistory(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "a"), "original")
	for _, name := range []string{"b", "dest"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0700); err != nil {
			t.Fatal(err)
		}
		writeTestFile(t, filepath.Join(dir, name, "keep"), name)
	}
	writeTestFile(t, filepath.Join(dir, "kv"), "a\tgood\nb\tdest\n")
	out, err := runCLI(t, dir, "-D", "-o", "1", "-p", "^([ab])$", "-r", "{kv}", "-k", "kv")
	if err == nil {
		t.Fatalf("expected nonempty-directory rename failure: %s", out)
	}
	requireFile(t, filepath.Join(dir, "good"), "original")
	history, err := readHistory(filepath.Join(dir, LastOpDetailFile))
	if err != nil || len(history) != 1 || history[0].source != "a" {
		t.Fatalf("history %#v, error %v", history, err)
	}
	if out, err := runCLI(t, dir, "-u"); err != nil {
		t.Fatalf("undo: %v: %s", err, out)
	}
	requireFile(t, filepath.Join(dir, "a"), "original")
	requireFile(t, filepath.Join(dir, "dest", "keep"), "dest")
}

func TestExecuteFirstFailurePreservesHistory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, LastOpDetailFile)
	writeTestFile(t, path, "previous history")
	opt := &Options{Quiet: true, LastOpDetailFile: path}
	_, err := executeOperations(opt, []operation{{source: filepath.Join(dir, "missing"), target: filepath.Join(dir, "target")}})
	if err == nil {
		t.Fatal("expected missing source error")
	}
	requireFile(t, path, "previous history")
}

func TestCLIHistoryAliases(t *testing.T) {
	for _, kind := range []string{"symlink", "hardlink", "target", "target child"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			writeTestFile(t, filepath.Join(dir, "a"), "original")
			path := filepath.Join(dir, LastOpDetailFile)
			args := []string{"-p", "^a$", "-r", "b", "a"}
			switch kind {
			case "symlink":
				writeTestFile(t, filepath.Join(dir, "precious"), "precious")
				if err := os.Symlink("precious", path); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
			case "hardlink":
				if err := os.Link(filepath.Join(dir, "a"), path); err != nil {
					t.Skipf("hardlinks unavailable: %v", err)
				}
			case "target":
				args = []string{"-p", "^a$", "-r", LastOpDetailFile, "a"}
			case "target child":
				args = []string{"-p", "^a$", "-r", filepath.Join(LastOpDetailFile, "nested", "b"), "a"}
			}
			for _, dryRun := range []bool{true, false} {
				flags := append([]string(nil), args...)
				if dryRun {
					flags = append(flags, "-d")
				}
				if out, err := runCLI(t, dir, flags...); err == nil {
					t.Fatalf("unsafe history accepted (dry run %v): %s", dryRun, out)
				}
			}
			requireFile(t, filepath.Join(dir, "a"), "original")
			if kind == "target child" {
				if _, err := os.Lstat(path); !os.IsNotExist(err) {
					t.Fatalf("created a directory in place of the history file: %v", err)
				}
			}
			if kind == "symlink" {
				requireFile(t, filepath.Join(dir, "precious"), "precious")
			}
		})
	}
}

func TestCLIHistoryDryRun(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "a"), "original")
	if out, err := runCLI(t, dir, "-p", "a", "-r", "b"); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	path := filepath.Join(dir, LastOpDetailFile)
	history, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"-u", "-d"}, {"-U", "-d"}, {"--clear", "-d", "."}} {
		if out, err := runCLI(t, dir, args...); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
		requireFile(t, filepath.Join(dir, "b"), "original")
		requireFile(t, path, string(history))
	}
	if out, err := runCLI(t, dir, "--clear", "."); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("history still exists: %v", err)
	}
}

func TestCLIHistoryDirectoryWithUndoDisabled(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "a"), "original")
	target := filepath.Join(LastOpDetailFile, "nested", "b")
	if out, err := runCLI(t, dir, "-d", "-x", "-p", "^a$", "-r", target, "a"); err != nil {
		t.Fatalf("preview reserved the history name with undo disabled: %v: %s", err, out)
	}
	requireFile(t, filepath.Join(dir, "a"), "original")
	if out, err := runCLI(t, dir, "-x", "-p", "^a$", "-r", target, "a"); err != nil {
		t.Fatalf("history name is not reserved with undo disabled: %v: %s", err, out)
	}
	requireFile(t, filepath.Join(dir, target), "original")
}

func TestCLIClearOverlappingInputs(t *testing.T) {
	dir := t.TempDir()
	child := filepath.Join(dir, "child")
	if err := os.Mkdir(child, 0700); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{dir, child} {
		writeTestFile(t, filepath.Join(path, LastOpDetailFile), "a"+historyDelimiter+"b\n")
	}
	args := []string{"--clear", "-R", ".", filepath.Join("child", LastOpDetailFile), LastOpDetailFile, LastOpDetailFile}
	if out, err := runCLI(t, dir, args...); err != nil {
		t.Fatalf("history was already cleared by another input: %v: %s", err, out)
	}
	for _, path := range []string{dir, child} {
		if _, err := os.Lstat(filepath.Join(path, LastOpDetailFile)); !os.IsNotExist(err) {
			t.Fatalf("history remains: %v", err)
		}
	}
}

func TestCLIUndoNestedDirectories(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "a", "a"), 0700); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(dir, "a", "a", "a"), "original")
	if out, err := runCLI(t, dir, "-R", "-D", "-p", "^a$", "-r", "b"); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	requireFile(t, filepath.Join(dir, "b", "b", "b"), "original")
	if out, err := runCLI(t, dir, "-u"); err != nil {
		t.Fatalf("undo: %v: %s", err, out)
	}
	requireFile(t, filepath.Join(dir, "a", "a", "a"), "original")
}

func TestCLIForceUndoFailedParent(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "a"), 0700); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(dir, "a", "a"), "original")
	if out, err := runCLI(t, dir, "-R", "-D", "-p", "^a$", "-r", "b"); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	// A new directory occupies the old parent name; its contents must not be undone.
	if err := os.Mkdir(filepath.Join(dir, "a"), 0700); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(dir, "a", "b"), "unrelated")
	if out, err := runCLI(t, dir, "-U"); err == nil {
		t.Fatalf("expected parent conflict: %s", out)
	}
	requireFile(t, filepath.Join(dir, "a", "b"), "unrelated")
	requireFile(t, filepath.Join(dir, "b", "b"), "original")
	if err := os.Rename(filepath.Join(dir, "a"), filepath.Join(dir, "new")); err != nil {
		t.Fatal(err)
	}
	if out, err := runCLI(t, dir, "-u"); err != nil {
		t.Fatalf("retry: %v: %s", err, out)
	}
	requireFile(t, filepath.Join(dir, "a", "a"), "original")
}

func TestCLILegacyUndo(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "#b"), "original")
	writeTestFile(t, filepath.Join(dir, LastOpDetailFile), "#a"+historyDelimiter+"#b\n")
	if out, err := runCLI(t, dir, "-u"); err != nil {
		t.Fatalf("legacy undo: %v: %s", err, out)
	}
	requireFile(t, filepath.Join(dir, "#a"), "original")
}

func TestCLIForceUndoReplacedParent(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "a"), 0700); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(dir, "a", "a"), "nested")
	writeTestFile(t, filepath.Join(dir, "a2"), "independent")
	if out, err := runCLI(t, dir, "-R", "-D", "-p", "^a", "-r", "b"); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	writeTestFile(t, filepath.Join(dir, "a"), "new file")
	if out, err := runCLI(t, dir, "-U"); err == nil {
		t.Fatalf("expected parent conflict: %s", out)
	}
	requireFile(t, filepath.Join(dir, "a"), "new file")
	requireFile(t, filepath.Join(dir, "a2"), "independent")
	history, err := readHistory(filepath.Join(dir, LastOpDetailFile))
	if err != nil || len(history) != 2 {
		t.Fatalf("remaining %#v, error %v", history, err)
	}
	if err := os.Remove(filepath.Join(dir, "a")); err != nil {
		t.Fatal(err)
	}
	if out, err := runCLI(t, dir, "-u"); err != nil {
		t.Fatalf("retry: %v: %s", err, out)
	}
	requireFile(t, filepath.Join(dir, "a", "a"), "nested")
}

func TestHistoryLegacyWriteCompatibility(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, LastOpDetailFile)
	// Tabs, quotes, backslashes and arbitrary bytes are safe without a format change.
	history := []operation{
		{source: "a", target: "b"},
		{source: "dir/a\t\"\\\xff", target: "dir/#b\r"},
	}
	writeTestHistory(t, path, history)
	want := ""
	for _, op := range history {
		want += op.source + historyDelimiter + op.target + "\n"
	}
	requireFile(t, path, want)
	got, err := readHistory(path)
	if err != nil || len(got) != len(history) {
		t.Fatalf("history %#v, error %v", got, err)
	}
	for i := range history {
		if got[i] != history[i] {
			t.Fatalf("record %d: %#v, want %#v", i, got[i], history[i])
		}
	}
}

func TestCLIForceUndoTransitiveDependencies(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a", "b", "c"} {
		writeTestFile(t, filepath.Join(dir, name), name)
	}
	writeTestFile(t, filepath.Join(dir, "kv"), "a\tb\nb\tc\nc\td\n")
	if out, err := runCLI(t, dir, "-o", "1", "-p", "^([abc])$", "-r", "{kv}", "-k", "kv", "a", "b", "c"); err != nil {
		t.Fatalf("chain: %v: %s", err, out)
	}
	// A failure at the end of the chain must block every earlier dependent move.
	writeTestFile(t, filepath.Join(dir, "c"), "conflict")
	writeTestFile(t, filepath.Join(dir, "b"), "unrelated")
	if out, err := runCLI(t, dir, "-U"); err == nil {
		t.Fatalf("expected undo conflict: %s", out)
	}
	requireFile(t, filepath.Join(dir, "b"), "unrelated")
	requireFile(t, filepath.Join(dir, "c"), "conflict")
	requireFile(t, filepath.Join(dir, "d"), "a")
	history, err := readHistory(filepath.Join(dir, LastOpDetailFile))
	if err != nil || len(history) != 3 {
		t.Fatalf("remaining history: %#v, %v", history, err)
	}
	for _, name := range []string{"b", "c"} {
		if err := os.Remove(filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	if out, err := runCLI(t, dir, "-u"); err != nil {
		t.Fatalf("retry: %v: %s", err, out)
	}
	requireFile(t, filepath.Join(dir, "a"), "a")
}

func TestCLIHistoryFormatSelection(t *testing.T) {
	for _, special := range []bool{false, true} {
		name := "ordinary"
		if special {
			name = "escaped"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writeTestFile(t, filepath.Join(dir, "a"), "ordinary")
			args := []string{"-p", "a", "-r", "b", "a"}
			if special {
				writeTestFile(t, filepath.Join(dir, "#a"), "special")
				args = append(args, "#a")
			}
			if out, err := runCLI(t, dir, args...); err != nil {
				t.Fatalf("%v: %s", err, out)
			}
			data, err := os.ReadFile(filepath.Join(dir, ".brename_detail.txt"))
			if err != nil {
				t.Fatal(err)
			}
			if special {
				if !strings.HasPrefix(string(data), historyHeader+"\n") || strings.Contains(string(data), historyDelimiter) {
					t.Fatalf("mixed log must use escaping throughout: %q", data)
				}
			} else if string(data) != "a"+historyDelimiter+"b\n" {
				t.Fatalf("ordinary record format changed: %q", data)
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), ".brename_detail.txt.tmp-") || strings.HasPrefix(entry.Name(), ".brename-history-") {
					t.Fatalf("temporary history left after success: %s", entry.Name())
				}
			}
			if out, err := runCLI(t, dir, "-u"); err != nil {
				t.Fatalf("undo: %v: %s", err, out)
			}
			requireFile(t, filepath.Join(dir, "a"), "ordinary")
			if special {
				requireFile(t, filepath.Join(dir, "#a"), "special")
			}
		})
	}
}

func TestCLIHistoryLeadingBOM(t *testing.T) {
	dir := t.TempDir()
	source := "\ufeffa"
	writeTestFile(t, filepath.Join(dir, source), "original")
	if out, err := runCLI(t, dir, "-p", "^.*$", "-r", "b", source); err != nil {
		t.Fatalf("rename: %v: %s", err, out)
	}
	data, err := os.ReadFile(filepath.Join(dir, LastOpDetailFile))
	if err != nil || !strings.HasPrefix(string(data), historyHeader+"\n") {
		t.Fatalf("BOM-prefixed filename exposed to legacy BOM stripping: %q, %v", data, err)
	}
	if out, err := runCLI(t, dir, "-u"); err != nil {
		t.Fatalf("undo: %v: %s", err, out)
	}
	requireFile(t, filepath.Join(dir, source), "original")
	if _, err := os.Stat(filepath.Join(dir, "a")); !os.IsNotExist(err) {
		t.Fatalf("created a BOM-stripped filename: %v", err)
	}
}

// Failed parent restores must block paths through replacement symlinks, while
// independent files and symlink entries remain eligible for forced undo.
func TestCLIForceUndoSymlinkParent(t *testing.T) {
	for _, alias := range []string{"none", "ancestor", "target", "parent components", "cycle"} {
		t.Run(alias, func(t *testing.T) {
			dir := t.TempDir()
			for _, name := range []string{"a/sub", "a/deep", "external/sub", "external/deep"} {
				if err := os.MkdirAll(filepath.Join(dir, name), 0700); err != nil {
					t.Fatal(err)
				}
			}
			input := "a/sub"
			if alias != "none" {
				target := "."
				input = "entrance/a/sub"
				if alias == "target" || alias == "cycle" {
					target, input = "a/sub", "entrance"
				}
				if alias == "parent components" {
					if err := os.Symlink("a/deep", filepath.Join(dir, "hop")); err != nil {
						t.Skipf("symlinks unavailable: %v", err)
					}
					target, input = "hop/../sub", "entrance"
				}
				if err := os.Symlink(target, filepath.Join(dir, "entrance")); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
			}
			if err := os.Symlink("a/missing", filepath.Join(dir, "a_link")); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			writeTestFile(t, filepath.Join(dir, "a", "sub", "a_file"), "original")
			writeTestFile(t, filepath.Join(dir, "external", "sub", "b_file"), "unrelated")
			writeTestFile(t, filepath.Join(dir, "external", "a_independent"), "independent")
			if out, err := runCLI(t, dir, "-D", "-p", "^a", "-r", "b", input, "external/a_independent", "a_link", "a"); err != nil {
				t.Fatalf("rename: %v: %s", err, out)
			}
			if err := os.Symlink("external", filepath.Join(dir, "a")); err != nil {
				t.Fatal(err)
			}
			if alias == "cycle" {
				if err := os.Remove(filepath.Join(dir, "entrance")); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("entrance", filepath.Join(dir, "entrance")); err != nil {
					t.Fatal(err)
				}
			}
			if out, err := runCLI(t, dir, "-U"); err == nil {
				t.Fatalf("expected parent conflict: %s", out)
			}
			requireFile(t, filepath.Join(dir, "external", "sub", "b_file"), "unrelated")
			requireFile(t, filepath.Join(dir, "b", "sub", "b_file"), "original")
			requireFile(t, filepath.Join(dir, "external", "a_independent"), "independent")
			if target, err := os.Readlink(filepath.Join(dir, "a_link")); err != nil || target != "a/missing" {
				t.Fatalf("independent link was not restored: %q, %v", target, err)
			}
			history, err := readHistory(filepath.Join(dir, LastOpDetailFile))
			if err != nil || len(history) != 2 {
				t.Fatalf("remaining dependent records: %#v, %v", history, err)
			}
			if err := os.Remove(filepath.Join(dir, "a")); err != nil {
				t.Fatal(err)
			}
			if alias == "cycle" {
				if err := os.Remove(filepath.Join(dir, "entrance")); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("a/sub", filepath.Join(dir, "entrance")); err != nil {
					t.Fatal(err)
				}
			}
			if out, err := runCLI(t, dir, "-u"); err != nil {
				t.Fatalf("retry: %v: %s", err, out)
			}
			requireFile(t, filepath.Join(dir, "a", "sub", "a_file"), "original")
			requireFile(t, filepath.Join(dir, "external", "sub", "b_file"), "unrelated")
		})
	}
}

// Stable records for a move that invalidates its own alias must still block
// earlier alias-based operations when the parent cannot be restored.
func TestCLIForceUndoSelfDependentRoot(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"sub", "external/sub"} {
		if err := os.MkdirAll(filepath.Join(dir, name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	for name, target := range map[string]string{"a_move": "sub", "entrance": "a_move/.."} {
		if err := os.Symlink(target, filepath.Join(dir, name)); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
	}
	writeTestFile(t, filepath.Join(dir, "a_z"), "original")
	writeTestFile(t, filepath.Join(dir, "external", "b_z"), "unrelated")
	if out, err := runCLI(t, dir, "-p", "^a", "-r", "b", "entrance"); err != nil {
		t.Fatalf("rename: %v: %s", err, out)
	}
	if err := os.Symlink("external/sub", filepath.Join(dir, "a_move")); err != nil {
		t.Fatal(err)
	}
	if out, err := runCLI(t, dir, "-U"); err == nil {
		t.Fatalf("expected parent conflict: %s", out)
	}
	requireFile(t, filepath.Join(dir, "b_z"), "original")
	requireFile(t, filepath.Join(dir, "external", "b_z"), "unrelated")
	history, err := readHistory(filepath.Join(dir, LastOpDetailFile))
	if err != nil || len(history) != 2 {
		t.Fatalf("remaining dependent records: %#v, %v", history, err)
	}
	if err := os.Remove(filepath.Join(dir, "a_move")); err != nil {
		t.Fatal(err)
	}
	if out, err := runCLI(t, dir, "-u"); err != nil {
		t.Fatalf("retry: %v: %s", err, out)
	}
	requireFile(t, filepath.Join(dir, "a_z"), "original")
	requireFile(t, filepath.Join(dir, "external", "b_z"), "unrelated")
}

// Invalid path bytes anywhere in a log must be rejected before later valid
// records are undone, in either encoding and even with forced undo.
func TestCLIUndoRejectsNULPaths(t *testing.T) {
	for _, format := range []string{"legacy", "quoted"} {
		for _, field := range []string{"source", "target"} {
			for _, mode := range []string{"undo", "force", "preview"} {
				t.Run(format+"/"+field+"/"+mode, func(t *testing.T) {
					dir := t.TempDir()
					writeTestFile(t, filepath.Join(dir, "b"), "valid file")
					source, target := "invalid", "missing"
					if field == "source" {
						source += "\x00name"
					} else {
						target += "\x00name"
					}
					data := source + historyDelimiter + target + "\na" + historyDelimiter + "b\n"
					if format == "quoted" {
						data = historyHeader + "\n" + strconv.Quote(source) + "\t" + strconv.Quote(target) + "\n\"a\"\t\"b\"\n"
					}
					path := filepath.Join(dir, LastOpDetailFile)
					writeTestFile(t, path, data)
					args := []string{"-u"}
					if mode == "force" {
						args = []string{"-U"}
					} else if mode == "preview" {
						args = append(args, "-d")
					}
					if out, err := runCLI(t, dir, args...); err == nil {
						t.Fatalf("invalid log accepted: %s", out)
					}
					requireFile(t, filepath.Join(dir, "b"), "valid file")
					requireFile(t, path, data)
					if _, err := os.Lstat(filepath.Join(dir, "a")); !os.IsNotExist(err) {
						t.Fatalf("a valid record was undone before rejecting the log: %v", err)
					}
				})
			}
		}
	}
}

func TestCLIForceUndoCaseInsensitiveParent(t *testing.T) {
	dir := caseInsensitiveTestDir(t)
	if err := os.MkdirAll(filepath.Join(dir, "parent", "sub"), 0700); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(dir, "parent", "sub", "a"), "original")
	writeTestFile(t, filepath.Join(dir, "kv"), "parent\trenamed\na\tb\n")
	if out, err := runCLI(t, dir, "-w", "-D", "-p", "^(parent|a)$", "-r", "{kv}", "-k", "kv", "PARENT/sub", "parent"); err != nil {
		t.Fatalf("rename: %v: %s", err, out)
	}
	if err := os.MkdirAll(filepath.Join(dir, "parent", "sub"), 0700); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(dir, "parent", "sub", "b"), "unrelated")
	if out, err := runCLI(t, dir, "-U"); err == nil {
		t.Fatalf("expected parent conflict: %s", out)
	}
	requireFile(t, filepath.Join(dir, "parent", "sub", "b"), "unrelated")
	requireFile(t, filepath.Join(dir, "renamed", "sub", "b"), "original")
	history, err := readHistory(filepath.Join(dir, LastOpDetailFile))
	if err != nil || len(history) != 2 {
		t.Fatalf("remaining dependent records: %#v, %v", history, err)
	}
	if err := os.Rename(filepath.Join(dir, "parent"), filepath.Join(dir, "conflict")); err != nil {
		t.Fatal(err)
	}
	if out, err := runCLI(t, dir, "-u"); err != nil {
		t.Fatalf("retry: %v: %s", err, out)
	}
	requireFile(t, filepath.Join(dir, "parent", "sub", "a"), "original")
	requireFile(t, filepath.Join(dir, "conflict", "sub", "b"), "unrelated")
}
