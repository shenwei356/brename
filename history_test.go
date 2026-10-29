package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestHistoryRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, LastOpDetailFile)
	// Include bytes that JSON strings cannot preserve, in addition to separators.
	history := []operation{{source: "#a\n\t\xff", target: "b\r\n" + historyDelimiter}}
	if err := writeHistory(path, history); err != nil {
		t.Fatal(err)
	}
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
	for _, kind := range []string{"symlink", "hardlink", "target"} {
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
			}
			if out, err := runCLI(t, dir, args...); err == nil {
				t.Fatalf("unsafe history accepted: %s", out)
			}
			requireFile(t, filepath.Join(dir, "a"), "original")
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
	if err := writeHistory(path, history); err != nil {
		t.Fatal(err)
	}
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
