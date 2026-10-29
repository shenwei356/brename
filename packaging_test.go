package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPackaging(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("release script requires a POSIX shell")
	}
	for _, command := range []string{"sh", "tar", "mktemp", "md5sum"} {
		if _, err := exec.LookPath(command); err != nil {
			t.Skipf("%s unavailable", command)
		}
	}
	script, err := os.ReadFile("packaging.sh")
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"success", "build failure", "output alias", "archive alias"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			writeTestFile(t, filepath.Join(dir, "packaging.sh"), string(script))
			writeTestFile(t, filepath.Join(dir, "brename_test.go"), "source must remain")
			if err := os.Mkdir(filepath.Join(dir, "tools"), 0700); err != nil {
				t.Fatal(err)
			}
			// Stub only the compiler; run the real release script and archiver.
			stub := `#!/bin/sh
set -eu
printf '%s\n' "$GOOS/$GOARCH" >> calls
if [ "$GOOS/$GOARCH" = "${FAIL_TARGET:-}" ]; then exit 7; fi
while [ "$#" -gt 0 ]; do
    if [ "$1" = -o ]; then
        shift
        printf '%s' "$GOOS/$GOARCH" > "$1"
        exit 0
    fi
    shift
done
exit 8
`
			goPath := filepath.Join(dir, "tools", "go")
			writeTestFile(t, goPath, stub)
			if err := os.Chmod(goPath, 0700); err != nil {
				t.Fatal(err)
			}
			if scenario == "output alias" {
				if err := os.Symlink(".", filepath.Join(dir, "binaries")); err != nil {
					t.Fatal(err)
				}
			} else if err := os.Mkdir(filepath.Join(dir, "binaries"), 0700); err != nil {
				t.Fatal(err)
			}
			old := filepath.Join(dir, "binaries", "unrelated.tar.gz")
			writeTestFile(t, old, "previous release")
			if scenario == "archive alias" {
				if err := os.Symlink("..", filepath.Join(dir, "binaries", "brename_linux_amd64.tar.gz")); err != nil {
					t.Fatal(err)
				}
			} else {
				writeTestFile(t, filepath.Join(dir, "binaries", "brename_linux_amd64.tar.gz"), "old amd64")
			}
			cmd := exec.Command("sh", "packaging.sh")
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "PATH="+filepath.Join(dir, "tools")+string(os.PathListSeparator)+os.Getenv("PATH"), "FAIL_TARGET=")
			if scenario == "build failure" {
				cmd.Env = append(cmd.Env, "FAIL_TARGET=linux/arm64")
			}
			out, err := cmd.CombinedOutput()
			if (err == nil) != (scenario == "success") {
				t.Fatalf("%s: %v: %s", scenario, err, out)
			}
			requireFile(t, filepath.Join(dir, "brename_test.go"), "source must remain")
			requireFile(t, old, "previous release")
			if scenario == "success" {
				calls, err := os.ReadFile(filepath.Join(dir, "calls"))
				if err != nil {
					t.Fatal(err)
				}
				targets := strings.Fields(string(calls))
				if len(targets) != 6 {
					t.Fatalf("build targets: %q", calls)
				}
				for _, target := range targets {
					suffix := ""
					if strings.HasPrefix(target, "windows/") {
						suffix = ".exe"
					}
					asset := "brename_" + strings.ReplaceAll(target, "/", "_") + suffix + ".tar.gz"
					archiveData, err := os.ReadFile(filepath.Join(dir, "binaries", asset))
					if err != nil {
						t.Fatal(err)
					}
					requireFile(t, filepath.Join(dir, "binaries", asset+".md5.txt"), fmt.Sprintf("%x  %s\n", md5.Sum(archiveData), asset))
					fh, err := os.Open(filepath.Join(dir, "binaries", asset))
					if err != nil {
						t.Fatal(err)
					}
					gz, err := gzip.NewReader(fh)
					if err != nil {
						fh.Close()
						t.Fatal(err)
					}
					tr := tar.NewReader(gz)
					header, err := tr.Next()
					if err != nil || header.Name != "brename"+suffix {
						t.Fatalf("%s: header %v, error %v", asset, header, err)
					}
					data, err := io.ReadAll(tr)
					if err != nil || string(data) != target {
						t.Fatalf("%s: content %q, error %v", asset, data, err)
					}
					if _, err := tr.Next(); err != io.EOF {
						t.Fatalf("unexpected extra archive entry: %v", err)
					}
					gz.Close()
					fh.Close()
				}
			} else if scenario != "archive alias" {
				requireFile(t, filepath.Join(dir, "binaries", "brename_linux_amd64.tar.gz"), "old amd64")
			}
			if leftovers, err := filepath.Glob(filepath.Join(dir, "binaries", ".brename-packaging.*")); err != nil || len(leftovers) != 0 {
				t.Fatalf("staging leftovers: %v, %v", leftovers, err)
			}
		})
	}
}
