package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// pathKey identifies a directory entry through absolute, resolved parent paths.
// The last component is not resolved: renaming a symlink moves the link itself.
func pathKey(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	parent, err := resolveDirectory(filepath.Dir(abs))
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, filepath.Base(abs)), nil
}

// resolveDirectory resolves existing parents and preserves missing directories.
func resolveDirectory(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		info, statErr := os.Stat(resolved)
		if statErr != nil {
			return "", statErr
		}
		if !info.IsDir() {
			return "", fmt.Errorf("not a directory: %s", path)
		}
		return resolved, nil
	}
	if !os.IsNotExist(err) || filepath.Dir(path) == path {
		return "", err
	}
	if _, linkErr := os.Lstat(path); linkErr == nil {
		return "", err // A dangling directory symlink is not a missing directory.
	}
	parent, err := resolveDirectory(filepath.Dir(path))
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, filepath.Base(path)), nil
}

// containsPath reports whether child is parent itself or an entry below it.
func containsPath(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

// protectWorkingDirectory prevents moving or replacing the working directory
// or one of its ancestors, including through a directory alias.
func protectWorkingDirectory(source, target string) error {
	var cwd string // Resolve the working directory only for directory operations.
	for _, path := range []string{source, target} {
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.IsDir() {
			continue
		}
		if cwd == "" {
			cwd, err = os.Getwd()
			if err != nil {
				return err
			}
			cwd, err = filepath.EvalSymlinks(cwd)
			if err != nil {
				return err
			}
		}
		key, err := pathKey(path)
		if err != nil {
			return err
		}
		if containsPath(key, cwd) {
			return fmt.Errorf("cannot rename the working directory or its ancestor: %s", path)
		}
	}
	return nil
}

// targetExists includes dangling symlinks, while allowing a case-only rename
// when the filesystem reports that both spellings refer to the same entry.
func targetExists(source, target string) (bool, error) {
	info, err := os.Lstat(target)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	sourceKey, err := pathKey(source)
	if err != nil {
		return false, err
	}
	targetKey, err := pathKey(target)
	if err != nil {
		return false, err
	}
	if strings.EqualFold(sourceKey, targetKey) {
		if sourceInfo, err := os.Lstat(source); err == nil && os.SameFile(sourceInfo, info) {
			return false, nil
		}
	}
	return true, nil
}

// executeOperations performs the checked plan and records each successful move.
// Returning errors lets the journal close normally after a partial failure.
func executeOperations(opt *Options, ops []operation) (n int, err error) {
	var journal *os.File               // Unbuffered writes preserve completed operations on errors.
	var pending string                 // Keep the previous log until the first move succeeds.
	quoted := historyNeedsQuoting(ops) // Ordinary logs remain readable by older builds.
	if !opt.DisableUndo {
		if err := validateHistoryPath(opt.LastOpDetailFile, ops); err != nil {
			return 0, err
		}
		journal, err = createHistory(opt.LastOpDetailFile, quoted)
		if err != nil {
			return 0, err
		}
		pending = journal.Name()
		defer func() {
			if journal != nil {
				if closeErr := journal.Close(); err == nil {
					err = closeErr
				}
			}
			if pending != "" {
				if n == 0 {
					os.Remove(pending)
				} else if err != nil {
					err = fmt.Errorf("%w; recovery log: %s", err, pending)
				}
			}
		}()
	}
	if !opt.Quiet {
		log.Infof("\n%s\n", bold("Renaming paths..."))
	}
	for _, op := range ops {
		if err := protectWorkingDirectory(op.source, op.target); err != nil {
			return n, err
		}
		// Check again because earlier operations may have changed the filesystem.
		exists, err := targetExists(op.source, op.target)
		if err != nil {
			return n, err
		}
		if exists && opt.OverwriteMode != 1 {
			if opt.OverwriteMode == 2 {
				continue
			}
			return n, fmt.Errorf("target already exists: %s", op.target)
		}
		if err := os.MkdirAll(filepath.Dir(op.target), 0755); err != nil {
			return n, err
		}
		if err := os.Rename(op.source, op.target); err != nil {
			return n, err
		}
		n++
		if journal != nil {
			if err := writeHistoryRecord(journal, op, quoted); err != nil {
				return n, fmt.Errorf("record %s -> %s: %w", op.source, op.target, err)
			}
			if pending != "" {
				// Close before publishing so this also works on Windows.
				closeErr := journal.Close()
				journal = nil
				if closeErr != nil {
					return n, closeErr
				}
				if err := os.Rename(pending, opt.LastOpDetailFile); err != nil {
					return n, err
				}
				pending = ""
				journal, err = os.OpenFile(opt.LastOpDetailFile, os.O_WRONLY|os.O_APPEND, 0600)
				if err != nil {
					return n, err
				}
			}
		}
		if !opt.Quiet {
			log.Infof("  [%s] %s -> %s", green("DONE"), op.source, op.target)
		}
	}
	return n, nil
}
