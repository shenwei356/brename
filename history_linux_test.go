package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

// init constrains only the CLI subprocess, leaving the test runner unrestricted.
func init() {
	if os.Getenv("BRENAME_TEST_HELPER") != "1" || os.Getenv("BRENAME_TEST_FSIZE") == "" {
		return
	}
	size, err := strconv.ParseUint(os.Getenv("BRENAME_TEST_FSIZE"), 10, 64)
	if err != nil {
		panic(err)
	}
	signal.Ignore(syscall.SIGXFSZ)
	if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &syscall.Rlimit{Cur: size, Max: size}); err != nil {
		panic(err)
	}
}

func TestCLIHistoryWriteFailure(t *testing.T) {
	for _, overwrite := range []bool{false, true} {
		t.Run(fmt.Sprintf("first overwrite=%v", overwrite), func(t *testing.T) {
			dir := t.TempDir()
			writeTestFile(t, filepath.Join(dir, "a"), "source")
			previous := "old" + historyDelimiter + "older\n"
			writeTestFile(t, filepath.Join(dir, LastOpDetailFile), previous)
			args := []string{"-p", "^a$", "-r", "new_a", "a"}
			if overwrite {
				writeTestFile(t, filepath.Join(dir, "new_a"), "target")
				args = append(args, "-o", "1")
			}
			t.Setenv("BRENAME_TEST_FSIZE", "10")
			out, err := runCLI(t, dir, args...)
			if err == nil || !strings.Contains(out, "file too large") {
				t.Fatalf("expected history write failure: %v: %s", err, out)
			}
			requireFile(t, filepath.Join(dir, "a"), "source")
			requireFile(t, filepath.Join(dir, LastOpDetailFile), previous)
			if overwrite {
				requireFile(t, filepath.Join(dir, "new_a"), "target")
			} else if _, err := os.Lstat(filepath.Join(dir, "new_a")); !os.IsNotExist(err) {
				t.Fatalf("unrecorded rename: %v", err)
			}
		})
	}
	t.Run("after successful move", func(t *testing.T) {
		dir := t.TempDir()
		writeTestFile(t, filepath.Join(dir, "a"), "one")
		writeTestFile(t, filepath.Join(dir, "b"), "two")
		firstRecord := "a" + historyDelimiter + "new_a\n"
		// Permit the first record and a partial second record, then force a write error.
		t.Setenv("BRENAME_TEST_FSIZE", strconv.Itoa(len(firstRecord)+3))
		out, err := runCLI(t, dir, "-p", "^", "-r", "new_", "a", "b")
		if err == nil || !strings.Contains(out, "file too large") {
			t.Fatalf("expected second history write failure: %v: %s", err, out)
		}
		requireFile(t, filepath.Join(dir, "new_a"), "one")
		requireFile(t, filepath.Join(dir, "b"), "two")
		requireFile(t, filepath.Join(dir, LastOpDetailFile), firstRecord)
		t.Setenv("BRENAME_TEST_FSIZE", "")
		if out, err := runCLI(t, dir, "-u"); err != nil {
			t.Fatalf("undo completed move: %v: %s", err, out)
		}
		requireFile(t, filepath.Join(dir, "a"), "one")
	})
}

func TestCLIUndoHistoryWriteFailure(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a1", "a2"} {
		writeTestFile(t, filepath.Join(dir, name), name)
	}
	if out, err := runCLI(t, dir, "-p", "^a", "-r", "b", "a1", "a2"); err != nil {
		t.Fatalf("rename: %v: %s", err, out)
	}
	path := filepath.Join(dir, LastOpDetailFile)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(dir, "a1"), "conflict")
	t.Setenv("BRENAME_TEST_FSIZE", "10")
	if out, err := runCLI(t, dir, "-U"); err == nil {
		t.Fatalf("expected history write failure: %s", out)
	}
	// Reject unwritable recovery history before completing any undo operation.
	requireFile(t, filepath.Join(dir, "b2"), "a2")
	requireFile(t, path, string(before))
	t.Setenv("BRENAME_TEST_FSIZE", "")
	if err := os.Remove(filepath.Join(dir, "a1")); err != nil {
		t.Fatal(err)
	}
	if out, err := runCLI(t, dir, "-u"); err != nil {
		t.Fatalf("retry: %v: %s", err, out)
	}
	for _, name := range []string{"a1", "a2"} {
		requireFile(t, filepath.Join(dir, name), name)
	}
}
