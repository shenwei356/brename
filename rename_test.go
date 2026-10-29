package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCLIPathCollisions(t *testing.T) {
	for _, mode := range []string{"0", "1", "2"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			writeTestFile(t, filepath.Join(dir, "a"), "AAA")
			writeTestFile(t, filepath.Join(dir, "b"), "BBB")
			out, err := runCLI(t, dir, "-o", mode, "-p", "^[ab]$", "-r", "same", "a", filepath.Join(dir, "b"))
			if mode == "0" {
				if err == nil {
					t.Fatalf("collision accepted: %s", out)
				}
				requireFile(t, filepath.Join(dir, "a"), "AAA")
				requireFile(t, filepath.Join(dir, "b"), "BBB")
			} else {
				if err != nil {
					t.Fatalf("%v: %s", err, out)
				}
				want := "AAA"
				if mode == "1" {
					want = "BBB"
				}
				requireFile(t, filepath.Join(dir, "same"), want)
				if mode == "2" {
					requireFile(t, filepath.Join(dir, "b"), "BBB")
				}
			}
		})
	}
}

func TestCLISymlinkCollisions(t *testing.T) {
	t.Run("parent aliases", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, "real"), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("real", filepath.Join(dir, "alias")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		writeTestFile(t, filepath.Join(dir, "real", "a"), "AAA")
		writeTestFile(t, filepath.Join(dir, "real", "b"), "BBB")
		out, err := runCLI(t, dir, "-p", "^[ab]$", "-r", "new/same", "real/a", "alias/b")
		if err == nil {
			t.Fatalf("aliased targets accepted: %s", out)
		}
		requireFile(t, filepath.Join(dir, "real", "a"), "AAA")
		requireFile(t, filepath.Join(dir, "real", "b"), "BBB")
	})
	t.Run("dangling target", func(t *testing.T) {
		dir := t.TempDir()
		writeTestFile(t, filepath.Join(dir, "a"), "AAA")
		if err := os.Symlink("missing", filepath.Join(dir, "b")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		out, err := runCLI(t, dir, "-p", "a", "-r", "b", "a")
		if err == nil {
			t.Fatalf("dangling target overwritten: %s", out)
		}
		if target, err := os.Readlink(filepath.Join(dir, "b")); err != nil || target != "missing" {
			t.Fatalf("link %q, error %v", target, err)
		}
		requireFile(t, filepath.Join(dir, "a"), "AAA")
	})
}

func TestCLIWorkingDirectoryProtection(t *testing.T) {
	dir := t.TempDir()
	child := filepath.Join(dir, "child")
	if err := os.Mkdir(child, 0700); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(child, "a"), "original")
	out, err := runCLI(t, child, "-x", "-D", "-o", "1", "-p", "^child$", "-r", "other", child)
	if err == nil {
		t.Fatalf("working directory moved: %s", out)
	}
	requireFile(t, filepath.Join(child, "a"), "original")
}

func TestCLISpecialFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("FIFO fixture requires Unix")
	}
	mkfifo, err := exec.LookPath("mkfifo")
	if err != nil {
		t.Skip("mkfifo is unavailable")
	}
	dir := t.TempDir()
	if out, err := exec.Command(mkfifo, filepath.Join(dir, "a")).CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if out, err := runCLI(t, dir, "-d", "-p", "a", "-r", "b", "a"); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if out, err := runCLI(t, dir, "--clear", "a"); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if out, err := runCLI(t, dir, "-p", "a", "-r", "b", "a"); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if info, err := os.Lstat(filepath.Join(dir, "b")); err != nil || info.Mode()&os.ModeNamedPipe == 0 {
		t.Fatalf("FIFO not renamed: %v", err)
	}
}

func TestCLIExplicitDirectorySymlink(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "real"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real", filepath.Join(dir, "alias")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	writeTestFile(t, filepath.Join(dir, "real", "a"), "original")
	if out, err := runCLI(t, dir, "-p", "^a$", "-r", "b", "alias"); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	requireFile(t, filepath.Join(dir, "real", "b"), "original")
}
