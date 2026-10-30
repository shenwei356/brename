package main

import (
	"fmt"
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

func TestCLISymlinkParentComponents(t *testing.T) {
	for _, input := range []string{"p/link/..", "p/link/../a"} {
		t.Run(input, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, "external", "sub"), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(filepath.Join(dir, "p"), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Join("..", "external", "sub"), filepath.Join(dir, "p", "link")); err != nil {
				t.Skipf("directory symlinks unavailable: %v", err)
			}
			writeTestFile(t, filepath.Join(dir, "p", "a"), "outside requested scope")
			writeTestFile(t, filepath.Join(dir, "external", "a"), "requested")
			if out, err := runCLI(t, dir, "-p", "^a$", "-r", "b", input); err != nil {
				t.Fatalf("%v: %s", err, out)
			}
			requireFile(t, filepath.Join(dir, "p", "a"), "outside requested scope")
			requireFile(t, filepath.Join(dir, "external", "b"), "requested")
			if out, err := runCLI(t, dir, "-u"); err != nil {
				t.Fatalf("undo: %v: %s", err, out)
			}
			requireFile(t, filepath.Join(dir, "external", "a"), "requested")
		})
	}
	t.Run("replacement parent", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "external", "sub"), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join("external", "sub"), filepath.Join(dir, "link")); err != nil {
			t.Skipf("directory symlinks unavailable: %v", err)
		}
		writeTestFile(t, filepath.Join(dir, "a"), "original")
		if out, err := runCLI(t, dir, "-p", "^a$", "-r", "link/../new/b", "a"); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
		requireFile(t, filepath.Join(dir, "external", "new", "b"), "original")
		if _, err := os.Lstat(filepath.Join(dir, "new")); !os.IsNotExist(err) {
			t.Fatalf("wrong directory created: %v", err)
		}
	})
	t.Run("missing parent", func(t *testing.T) {
		dir := t.TempDir()
		writeTestFile(t, filepath.Join(dir, "a"), "original")
		if out, err := runCLI(t, dir, "-p", "a", "-r", "b", "missing/../a"); err == nil {
			t.Fatalf("accepted nonexistent traversal: %s", out)
		}
		requireFile(t, filepath.Join(dir, "a"), "original")
	})
}

func TestCLIDistinctCaseHardlinks(t *testing.T) {
	for _, mode := range []string{"0", "1", "2"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			writeTestFile(t, filepath.Join(dir, "a"), "original")
			if err := os.Link(filepath.Join(dir, "a"), filepath.Join(dir, "A")); err != nil {
				t.Skipf("case-distinct hard links unavailable: %v", err)
			}
			out, err := runCLI(t, dir, "-o", mode, "-p", "^a$", "-r", "A", "a")
			if mode != "2" && err == nil {
				t.Fatalf("reported a no-op as successful rename: %s", out)
			}
			if mode == "2" && err != nil {
				t.Fatalf("skip mode: %v: %s", err, out)
			}
			requireFile(t, filepath.Join(dir, "a"), "original")
			requireFile(t, filepath.Join(dir, "A"), "original")
			if _, err := os.Stat(filepath.Join(dir, LastOpDetailFile)); !os.IsNotExist(err) {
				t.Fatalf("history created without a rename: %v", err)
			}
		})
	}
}

func TestCLILogicalWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	physical := filepath.Join(dir, "external", "sub")
	if err := os.MkdirAll(physical, 0700); err != nil {
		t.Fatal(err)
	}
	logical := filepath.Join(dir, "link")
	if err := os.Symlink(physical, logical); err != nil {
		t.Skipf("directory symlinks unavailable: %v", err)
	}
	t.Setenv("PWD", logical)
	writeTestFile(t, filepath.Join(physical, "a"), "one")
	writeTestFile(t, filepath.Join(physical, "b"), "two")
	if out, err := runCLI(t, logical, "-p", "^[ab]$", "-r", "same", "a", filepath.Join(physical, "b")); err == nil {
		t.Fatalf("logical/physical collision accepted: %s", out)
	}
	requireFile(t, filepath.Join(physical, "a"), "one")
	requireFile(t, filepath.Join(physical, "b"), "two")
	writeTestFile(t, filepath.Join(dir, "external", "parent"), "physical parent")
	writeTestFile(t, filepath.Join(dir, "parent"), "logical parent")
	if out, err := runCLI(t, logical, "-p", "^parent$", "-r", "renamed", ".."); err != nil {
		t.Fatalf("parent search: %v: %s", err, out)
	}
	requireFile(t, filepath.Join(dir, "external", "renamed"), "physical parent")
	requireFile(t, filepath.Join(dir, "parent"), "logical parent")
}

func TestCLICaseDistinctParentHardlinks(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"lower", "LOWER"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0700); err != nil {
			t.Skipf("case-distinct directories unavailable: %v", err)
		}
	}
	source := filepath.Join(dir, "lower", "a")
	target := filepath.Join(dir, "LOWER", "a")
	writeTestFile(t, source, "original")
	if err := os.Link(source, target); err != nil {
		t.Skipf("hardlinks unavailable: %v", err)
	}
	if out, err := runCLI(t, dir, "-p", "^a$", "-r", "../LOWER/a", "lower/a"); err == nil {
		t.Fatalf("distinct parent hardlinks accepted: %s", out)
	}
	requireFile(t, source, "original")
	requireFile(t, target, "original")
	if _, err := os.Stat(filepath.Join(dir, LastOpDetailFile)); !os.IsNotExist(err) {
		t.Fatalf("history created without a rename: %v", err)
	}
}

func TestCLINoOpTargetAliasPreservesHistory(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "a"), "original")
	if out, err := runCLI(t, dir, "-p", "^a$", "-r", "b", "a"); err != nil {
		t.Fatalf("rename: %v: %s", err, out)
	}
	path := filepath.Join(dir, LastOpDetailFile)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, replacement := range []string{"./b", "../" + filepath.Base(dir) + "/b"} {
		if out, err := runCLI(t, dir, "-p", "^b$", "-r", replacement, "b"); err != nil {
			t.Fatalf("no-op: %v: %s", err, out)
		}
		requireFile(t, path, string(before))
	}
	if out, err := runCLI(t, dir, "-u"); err != nil {
		t.Fatalf("undo: %v: %s", err, out)
	}
	requireFile(t, filepath.Join(dir, "a"), "original")
}

func TestCLIDuplicateSourcesWithNumbering(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "a"), "one")
	writeTestFile(t, filepath.Join(dir, "b"), "two")
	if out, err := runCLI(t, dir, "-p", "^[ab]$", "-r", "new_{nr}", "--start-num", "1", "a", "./a", filepath.Join(dir, "a"), "b"); err != nil {
		t.Fatalf("duplicate source: %v: %s", err, out)
	}
	requireFile(t, filepath.Join(dir, "new_1"), "one")
	requireFile(t, filepath.Join(dir, "new_2"), "two")
	history, err := readHistory(filepath.Join(dir, LastOpDetailFile))
	if err != nil || len(history) != 2 {
		t.Fatalf("history: %#v, %v", history, err)
	}
	if out, err := runCLI(t, dir, "-u"); err != nil {
		t.Fatalf("undo: %v: %s", err, out)
	}
	requireFile(t, filepath.Join(dir, "a"), "one")
	requireFile(t, filepath.Join(dir, "b"), "two")
}

func TestCLIOverlappingRoots(t *testing.T) {
	for _, alias := range []bool{false, true} {
		name := "directory"
		if alias {
			name = "symlink"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			root := filepath.Join(dir, "a")
			if err := os.MkdirAll(filepath.Join(root, "sub"), 0700); err != nil {
				t.Fatal(err)
			}
			writeTestFile(t, filepath.Join(root, "sub", "a_file"), "original")
			input := "a"
			if alias {
				if err := os.Symlink("a", filepath.Join(dir, "alias")); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
				input = "alias"
			}
			if out, err := runCLI(t, dir, "-D", "-p", "^a", "-r", "b", input, filepath.Join(input, "sub")); err != nil {
				t.Fatalf("overlapping roots: %v: %s", err, out)
			}
			if !alias {
				root = filepath.Join(dir, "b")
			}
			requireFile(t, filepath.Join(root, "sub", "b_file"), "original")
			if out, err := runCLI(t, dir, "-u"); err != nil {
				t.Fatalf("undo: %v: %s", err, out)
			}
			requireFile(t, filepath.Join(dir, "a", "sub", "a_file"), "original")
			if alias {
				if target, err := os.Readlink(filepath.Join(dir, "alias")); err != nil || target != "a" {
					t.Fatalf("restored alias: %q, %v", target, err)
				}
			}
		})
	}
}

func TestCLIRenamedTargetParent(t *testing.T) {
	for _, test := range []struct {
		name   string
		alias  bool
		inputs []string
	}{
		{"directory", false, []string{"a", "z"}},
		{"symlink", true, []string{"a", "z"}},
		{"single directory", false, nil},
		{"single symlink", true, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			alias := test.alias
			dir := t.TempDir()
			parent := "a"
			if alias {
				parent = "real"
			}
			if err := os.Mkdir(filepath.Join(dir, parent), 0700); err != nil {
				t.Fatal(err)
			}
			if alias {
				if err := os.Symlink(parent, filepath.Join(dir, "a")); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
			}
			writeTestFile(t, filepath.Join(dir, "z"), "original")
			writeTestFile(t, filepath.Join(dir, "kv"), "a\tb\nz\ta/file\n")
			args := append([]string{"-D", "-p", "^([az])$", "-r", "{kv}", "-k", "kv"}, test.inputs...)
			if out, err := runCLI(t, dir, args...); err != nil {
				t.Fatalf("rename target parent: %v: %s", err, out)
			}
			if !alias {
				parent = "b"
			}
			requireFile(t, filepath.Join(dir, parent, "file"), "original")
			if out, err := runCLI(t, dir, "-u"); err != nil {
				t.Fatalf("undo: %v: %s", err, out)
			}
			requireFile(t, filepath.Join(dir, "z"), "original")
		})
	}
}

func TestCLICyclicDirectoryTargets(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a", "b"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0700); err != nil {
			t.Fatal(err)
		}
		writeTestFile(t, filepath.Join(dir, name, "keep"), name)
	}
	writeTestFile(t, filepath.Join(dir, "kv"), "a\tb/nested\nb\ta/nested\n")
	out, err := runCLI(t, dir, "-D", "-p", "^([ab])$", "-r", "{kv}", "-k", "kv")
	if err == nil {
		t.Fatalf("accepted cyclic directory dependencies: %s", out)
	}
	for _, name := range []string{"a", "b"} {
		requireFile(t, filepath.Join(dir, name, "keep"), name)
	}
	if _, err := os.Stat(filepath.Join(dir, LastOpDetailFile)); !os.IsNotExist(err) {
		t.Fatalf("history created for rejected cycle: %v", err)
	}
}

func TestCLIDistinctHardlinkSources(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "a"), "original")
	if err := os.Link(filepath.Join(dir, "a"), filepath.Join(dir, "b")); err != nil {
		t.Skipf("hardlinks unavailable: %v", err)
	}
	if out, err := runCLI(t, dir, "-p", "^[ab]$", "-r", "new_{nr}", "a", "b"); err != nil {
		t.Fatalf("distinct hardlink sources: %v: %s", err, out)
	}
	for _, name := range []string{"new_1", "new_2"} {
		requireFile(t, filepath.Join(dir, name), "original")
	}
	if out, err := runCLI(t, dir, "-u"); err != nil {
		t.Fatalf("undo: %v: %s", err, out)
	}
	for _, name := range []string{"a", "b"} {
		requireFile(t, filepath.Join(dir, name), "original")
	}
}

func TestCLIOrderingPreservesMoveChains(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "a", "sub"), 0700); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(dir, "a", "sub", "file"), "nested")
	writeTestFile(t, filepath.Join(dir, "c"), "independent")
	writeTestFile(t, filepath.Join(dir, "kv"), "a\tb\nc\td\nfile\t../../c\n")
	if out, err := runCLI(t, dir, "-D", "-o", "1", "-p", "^(a|c|file)$", "-r", "{kv}", "-k", "kv", "a", "c", "a/sub"); err != nil {
		t.Fatalf("rename chain: %v: %s", err, out)
	}
	// Moving the descendant before its parent must not move it ahead of c -> d.
	requireFile(t, filepath.Join(dir, "d"), "independent")
	requireFile(t, filepath.Join(dir, "c"), "nested")
	if out, err := runCLI(t, dir, "-u"); err != nil {
		t.Fatalf("undo: %v: %s", err, out)
	}
	requireFile(t, filepath.Join(dir, "c"), "independent")
	requireFile(t, filepath.Join(dir, "a", "sub", "file"), "nested")
}

func TestCLIOrderingPreservesOverwritePriority(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "a", "sub"), 0700); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(dir, "a", "sub", "file"), "last")
	writeTestFile(t, filepath.Join(dir, "z"), "first")
	writeTestFile(t, filepath.Join(dir, "kv"), "a\tb\nz\tsame\nfile\t../../same\n")
	if out, err := runCLI(t, dir, "-D", "-o", "1", "-p", "^(a|z|file)$", "-r", "{kv}", "-k", "kv", "a", "z", "a/sub"); err != nil {
		t.Fatalf("overwrite order: %v: %s", err, out)
	}
	requireFile(t, filepath.Join(dir, "same"), "last")
}

func TestCLIUnchangedNonportableNames(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not preserve ordinary trailing periods and spaces")
	}
	for _, unchanged := range []string{"a.", "a "} {
		t.Run(unchanged, func(t *testing.T) {
			dir := t.TempDir()
			writeTestFile(t, filepath.Join(dir, unchanged), "unchanged")
			writeTestFile(t, filepath.Join(dir, "b"), "renamed")
			writeTestFile(t, filepath.Join(dir, "kv"), unchanged+"\t"+unchanged+"\nb\tc\n")
			if out, err := runCLI(t, dir, "-p", "^(.+)$", "-r", "{kv}", "-k", "kv", unchanged, "b"); err != nil {
				t.Fatalf("unchanged filename prevented valid rename: %v: %s", err, out)
			}
			requireFile(t, filepath.Join(dir, unchanged), "unchanged")
			requireFile(t, filepath.Join(dir, "c"), "renamed")
			if out, err := runCLI(t, dir, "-u"); err != nil {
				t.Fatalf("undo: %v: %s", err, out)
			}
			requireFile(t, filepath.Join(dir, "b"), "renamed")
		})
	}
}

// A directory cannot become its own descendant, even through a symlink parent.
func TestCLIDirectoryIntoItself(t *testing.T) {
	for _, alias := range []bool{false, true} {
		for _, dryRun := range []bool{false, true} {
			t.Run(fmt.Sprintf("alias=%v/dry=%v", alias, dryRun), func(t *testing.T) {
				dir := t.TempDir()
				if err := os.Mkdir(filepath.Join(dir, "a"), 0700); err != nil {
					t.Fatal(err)
				}
				target := "a/new/moved"
				if alias {
					if err := os.Symlink("a", filepath.Join(dir, "alias")); err != nil {
						t.Skipf("symlinks unavailable: %v", err)
					}
					target = "alias/new/moved"
				}
				writeTestFile(t, filepath.Join(dir, "a", "keep"), "nested")
				writeTestFile(t, filepath.Join(dir, "z"), "independent")
				writeTestFile(t, filepath.Join(dir, "kv"), "a\t"+target+"\nz\trenamed\n")
				writeTestFile(t, filepath.Join(dir, LastOpDetailFile), "previous history\n")
				args := []string{"-D", "-k", "kv", "-p", "^(a|z)$", "-r", "{kv}", "z", "a"}
				if dryRun {
					args = append([]string{"-d"}, args...)
				}
				out, err := runCLI(t, dir, args...)
				if err == nil {
					t.Fatalf("accepted moving a directory into itself: %s", out)
				}
				requireFile(t, filepath.Join(dir, "z"), "independent")
				requireFile(t, filepath.Join(dir, "a", "keep"), "nested")
				requireFile(t, filepath.Join(dir, LastOpDetailFile), "previous history\n")
				if _, err := os.Lstat(filepath.Join(dir, "a", "new")); !os.IsNotExist(err) {
					t.Fatalf("created a destination parent for an impossible move: %v", err)
				}
			})
		}
	}
}

// Directory dependencies include links and parent components inside link targets.
func TestCLIOrderingThroughDirectoryLinks(t *testing.T) {
	for _, kind := range []string{"link chain", "parent components"} {
		for _, targetPath := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/target=%v", kind, targetPath), func(t *testing.T) {
				dir := t.TempDir()
				if err := os.MkdirAll(filepath.Join(dir, "real", "sub"), 0700); err != nil {
					t.Fatal(err)
				}
				linkTarget := "a/sub"
				if kind == "link chain" {
					if err := os.Symlink("real", filepath.Join(dir, "a")); err != nil {
						t.Skipf("symlinks unavailable: %v", err)
					}
				} else {
					if err := os.Mkdir(filepath.Join(dir, "a"), 0700); err != nil {
						t.Fatal(err)
					}
					linkTarget = "a/../real/sub"
				}
				if err := os.Symlink(linkTarget, filepath.Join(dir, "entrance")); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
				input, original := "entrance", filepath.Join("real", "sub", "a_file")
				if targetPath {
					input, original = "z", "z"
				}
				writeTestFile(t, filepath.Join(dir, original), "original")
				writeTestFile(t, filepath.Join(dir, "kv"), "a\tb\na_file\tb_file\nz\tentrance/b_file\n")
				if out, err := runCLI(t, dir, "-D", "-p", "^(a|a_file|z)$", "-r", "{kv}", "-k", "kv", "a", input); err != nil {
					t.Fatalf("rename through directory links: %v: %s", err, out)
				}
				requireFile(t, filepath.Join(dir, "real", "sub", "b_file"), "original")
				if out, err := runCLI(t, dir, "-u"); err != nil {
					t.Fatalf("undo: %v: %s", err, out)
				}
				requireFile(t, filepath.Join(dir, original), "original")
				if _, err := os.Lstat(filepath.Join(dir, "a")); err != nil {
					t.Fatalf("parent not restored: %v", err)
				}
			})
		}
	}
}

// A move can invalidate its own directory alias; both execution and undo need
// stable paths for that move, while the other entries must be renamed first.
func TestCLISelfDependentSearchRoot(t *testing.T) {
	for _, kind := range []string{"symlink", "directory"} {
		for _, otherFile := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/other=%v", kind, otherFile), func(t *testing.T) {
				dir := t.TempDir()
				if kind == "symlink" {
					if err := os.Mkdir(filepath.Join(dir, "sub"), 0700); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink("sub", filepath.Join(dir, "a_move")); err != nil {
						t.Skipf("symlinks unavailable: %v", err)
					}
				} else if err := os.Mkdir(filepath.Join(dir, "a_move"), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("a_move/..", filepath.Join(dir, "entrance")); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
				if otherFile {
					writeTestFile(t, filepath.Join(dir, "a_z"), "original")
				}
				if out, err := runCLI(t, dir, "-D", "-p", "^a", "-r", "b", "entrance"); err != nil {
					t.Fatalf("rename through its own alias: %v: %s", err, out)
				}
				if _, err := os.Lstat(filepath.Join(dir, "b_move")); err != nil {
					t.Fatalf("parent entry not renamed: %v", err)
				}
				if otherFile {
					requireFile(t, filepath.Join(dir, "b_z"), "original")
				}
				if out, err := runCLI(t, dir, "-u"); err != nil {
					t.Fatalf("undo through its own alias: %v: %s", err, out)
				}
				if _, err := os.Lstat(filepath.Join(dir, "a_move")); err != nil {
					t.Fatalf("parent entry not restored: %v", err)
				}
				if kind == "symlink" {
					if target, err := os.Readlink(filepath.Join(dir, "a_move")); err != nil || target != "sub" {
						t.Fatalf("restored symlink: %q, %v", target, err)
					}
				}
				if otherFile {
					requireFile(t, filepath.Join(dir, "a_z"), "original")
				}
			})
		}
	}
}

func TestCLISymlinkMovedIntoItsTarget(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "real"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real", filepath.Join(dir, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if out, err := runCLI(t, dir, "-D", "-p", "^link$", "-r", "link/moved", "link"); err != nil {
		t.Fatalf("moving a symlink into its target: %v: %s", err, out)
	}
	if target, err := os.Readlink(filepath.Join(dir, "real", "moved")); err != nil || target != "real" {
		t.Fatalf("moved symlink: %q, %v", target, err)
	}
	if out, err := runCLI(t, dir, "-u"); err != nil {
		t.Fatalf("undo symlink move: %v: %s", err, out)
	}
	if target, err := os.Readlink(filepath.Join(dir, "link")); err != nil || target != "real" {
		t.Fatalf("restored symlink: %q, %v", target, err)
	}
}

func TestCLINormalizedTargetNames(t *testing.T) {
	for _, test := range []struct{ replacement, target string }{
		{"bad./", ""}, {"bad.//", ""}, {"bad /", ""}, {"bad //", ""},
		{"b/.", "b"}, {"b/", "b"},
	} {
		for _, dryRun := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/dry=%v", test.replacement, dryRun), func(t *testing.T) {
				dir := t.TempDir()
				writeTestFile(t, filepath.Join(dir, "a"), "original")
				writeTestFile(t, filepath.Join(dir, LastOpDetailFile), "previous history\n")
				args := []string{"-p", "^a$", "-r", test.replacement, "a"}
				if dryRun {
					args = append([]string{"-d"}, args...)
				}
				out, err := runCLI(t, dir, args...)
				if (err == nil) != (test.target != "") {
					t.Fatalf("normalized target validation: %v: %s", err, out)
				}
				if test.target == "" || dryRun {
					requireFile(t, filepath.Join(dir, "a"), "original")
					requireFile(t, filepath.Join(dir, LastOpDetailFile), "previous history\n")
					entries, err := os.ReadDir(dir)
					if err != nil || len(entries) != 2 {
						t.Fatalf("unexpected filesystem changes: %v, %v", entries, err)
					}
				} else {
					requireFile(t, filepath.Join(dir, test.target), "original")
					if out, err := runCLI(t, dir, "-u"); err != nil {
						t.Fatalf("undo: %v: %s", err, out)
					}
					requireFile(t, filepath.Join(dir, "a"), "original")
				}
			})
		}
	}
}

// caseInsensitiveTestDir requires a real filesystem alias, not a simulated flag.
// Set TMPDIR (TMP on Windows) to a suitable writable filesystem to run these cases.
func caseInsensitiveTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	probe := filepath.Join(dir, "case_probe")
	writeTestFile(t, probe, "probe")
	lower, err := os.Stat(probe)
	if err != nil {
		t.Fatal(err)
	}
	upper, err := os.Stat(filepath.Join(dir, "CASE_PROBE"))
	if os.IsNotExist(err) || (err == nil && !os.SameFile(lower, upper)) {
		t.Skip("requires a case-insensitive filesystem for temporary files")
	}
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(probe); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestCLICaseInsensitiveAliases(t *testing.T) {
	t.Run("case-only rename", func(t *testing.T) {
		dir := caseInsensitiveTestDir(t)
		writeTestFile(t, filepath.Join(dir, "file"), "original")
		if out, err := runCLI(t, dir, "-w", "-p", "^file$", "-r", "FILE", "file"); err != nil {
			t.Fatalf("case-only rename: %v: %s", err, out)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, entry := range entries {
			found = found || entry.Name() == "FILE"
		}
		if !found {
			t.Fatal("directory entry spelling was not changed")
		}
		if out, err := runCLI(t, dir, "-u"); err != nil {
			t.Fatalf("undo: %v: %s", err, out)
		}
		entries, err = os.ReadDir(dir)
		if err != nil || len(entries) != 1 || entries[0].Name() != "file" {
			t.Fatalf("restored entry spelling: %v, %v", entries, err)
		}
		requireFile(t, filepath.Join(dir, "file"), "original")
	})
	t.Run("duplicate numbered source", func(t *testing.T) {
		dir := caseInsensitiveTestDir(t)
		writeTestFile(t, filepath.Join(dir, "file"), "original")
		if out, err := runCLI(t, dir, "-w", "-p", "^.*$", "-r", "new_{nr}", "file", "FILE"); err != nil {
			t.Fatalf("duplicate source alias: %v: %s", err, out)
		}
		requireFile(t, filepath.Join(dir, "new_1"), "original")
		history, err := readHistory(filepath.Join(dir, LastOpDetailFile))
		if err != nil || len(history) != 1 {
			t.Fatalf("one entry should produce one record: %#v, %v", history, err)
		}
		if out, err := runCLI(t, dir, "-u"); err != nil {
			t.Fatalf("undo: %v: %s", err, out)
		}
		requireFile(t, filepath.Join(dir, "file"), "original")
	})
	t.Run("unchanged parent alias", func(t *testing.T) {
		dir := caseInsensitiveTestDir(t)
		if err := os.Mkdir(filepath.Join(dir, "parent"), 0700); err != nil {
			t.Fatal(err)
		}
		writeTestFile(t, filepath.Join(dir, "parent", "file"), "original")
		writeTestFile(t, filepath.Join(dir, LastOpDetailFile), "previous history\n")
		if out, err := runCLI(t, dir, "-w", "-p", "^file$", "-r", "../PARENT/file", "parent/file"); err != nil {
			t.Fatalf("unchanged alias: %v: %s", err, out)
		}
		requireFile(t, filepath.Join(dir, "parent", "file"), "original")
		requireFile(t, filepath.Join(dir, LastOpDetailFile), "previous history\n")
	})
	t.Run("working directory ancestor", func(t *testing.T) {
		dir := caseInsensitiveTestDir(t)
		cwd := filepath.Join(dir, "parent", "child")
		if err := os.MkdirAll(cwd, 0700); err != nil {
			t.Fatal(err)
		}
		writeTestFile(t, filepath.Join(cwd, "file"), "original")
		out, err := runCLI(t, cwd, "-d", "-x", "-w", "--only-dir", "-p", "^PARENT$", "-r", "renamed", filepath.Join(dir, "PARENT"))
		if err == nil {
			t.Fatalf("working directory ancestor accepted through a case alias: %s", out)
		}
		requireFile(t, filepath.Join(cwd, "file"), "original")
	})
	t.Run("parent dependency", func(t *testing.T) {
		dir := caseInsensitiveTestDir(t)
		if err := os.MkdirAll(filepath.Join(dir, "parent", "sub"), 0700); err != nil {
			t.Fatal(err)
		}
		writeTestFile(t, filepath.Join(dir, "parent", "sub", "a"), "original")
		writeTestFile(t, filepath.Join(dir, "kv"), "parent\trenamed\na\tb\n")
		if out, err := runCLI(t, dir, "-w", "-D", "-p", "^(parent|a)$", "-r", "{kv}", "-k", "kv", "parent", "PARENT/sub"); err != nil {
			t.Fatalf("case alias dependency: %v: %s", err, out)
		}
		requireFile(t, filepath.Join(dir, "renamed", "sub", "b"), "original")
		if out, err := runCLI(t, dir, "-u"); err != nil {
			t.Fatalf("undo: %v: %s", err, out)
		}
		requireFile(t, filepath.Join(dir, "parent", "sub", "a"), "original")
	})
}
