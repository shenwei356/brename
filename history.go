package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// historyHeader marks logs whose filenames cannot be represented by the old format.
const historyHeader = "# brename history v2"

// historyDelimiter preserves the existing format for ordinary filenames.
const historyDelimiter = "\t_shenwei356-brename_\t"

// historyNeedsQuoting selects escaping for the whole log, so an older reader
// cannot accidentally undo only the ordinary records of a mixed-format log.
func historyNeedsQuoting(ops []operation) bool {
	for _, op := range ops {
		if strings.HasPrefix(op.source, "#") {
			return true // Older readers treat these source names as comments.
		}
		for _, path := range []string{op.source, op.target} {
			if strings.Contains(path, "\n") || strings.Contains(path, historyDelimiter) {
				return true
			}
		}
	}
	return false
}

// validateHistoryPath rejects special files and aliases of renamed entries.
func validateHistoryPath(path string, ops []operation) error {
	info, err := os.Lstat(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("undo log must be a regular file: %s", path)
	}
	key, err := pathKey(path)
	if err != nil {
		return err
	}
	for _, op := range ops {
		for _, entry := range []string{op.source, op.target} {
			entryKey, err := pathKey(entry)
			if err != nil {
				// A failed undo entry can have an inaccessible or replaced parent.
				// Keep that record for retry; the move itself checks filesystem errors.
				entryKey, err = filepath.Abs(entry)
				if err != nil {
					return err
				}
			}
			entryInfo, statErr := os.Lstat(entry)
			if key == entryKey || (info != nil && statErr == nil && os.SameFile(info, entryInfo)) {
				return fmt.Errorf("undo log conflicts with renamed path: %s", entry)
			}
		}
	}
	return nil
}

// createHistory prepares a new log beside the destination without truncating it.
func createHistory(path string, quoted bool) (*os.File, error) {
	fh, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return nil, err
	}
	if quoted {
		if _, err := fmt.Fprintln(fh, historyHeader); err != nil {
			fh.Close()
			os.Remove(fh.Name())
			return nil, err
		}
	}
	return fh, nil
}

// writeHistoryRecord retains the old bytes unless this log needs escaped paths.
func writeHistoryRecord(w io.Writer, op operation, quoted bool) error {
	if quoted {
		_, err := fmt.Fprintf(w, "%q\t%q\n", op.source, op.target)
		return err
	}
	_, err := fmt.Fprintf(w, "%s%s%s\n", op.source, historyDelimiter, op.target)
	return err
}

// writeHistory replaces a log only after all remaining records have been written.
func writeHistory(path string, history []operation) error {
	if err := validateHistoryPath(path, history); err != nil {
		return err
	}
	quoted := historyNeedsQuoting(history)
	fh, err := createHistory(path, quoted)
	if err != nil {
		return err
	}
	defer os.Remove(fh.Name())
	for _, op := range history {
		if err := writeHistoryRecord(fh, op, quoted); err != nil {
			fh.Close()
			return err
		}
	}
	if err := fh.Close(); err != nil {
		return err
	}
	return os.Rename(fh.Name(), path)
}

// readHistory validates the entire log before any undo operation is attempted.
func readHistory(path string) (history []operation, err error) {
	if err := validateHistoryPath(path, nil); err != nil {
		return nil, err
	}
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := fh.Close(); err == nil {
			err = closeErr
		}
	}()
	reader := bufio.NewReader(fh)
	quoted := false // Legacy records are read until a version header is found.
	for lineNumber := 1; ; lineNumber++ {
		line, readErr := reader.ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			return nil, readErr
		}
		line = strings.TrimSuffix(line, "\n")
		if lineNumber == 1 && line == historyHeader {
			quoted = true
		} else if line != "" {
			var source, target string
			if quoted {
				parts := strings.Split(line, "\t")
				if len(parts) != 2 {
					return nil, fmt.Errorf("invalid undo record at line %d", lineNumber)
				}
				source, err = strconv.Unquote(parts[0])
				if err == nil {
					target, err = strconv.Unquote(parts[1])
				}
				if err != nil {
					return nil, fmt.Errorf("invalid undo record at line %d: %w", lineNumber, err)
				}
			} else {
				parts := strings.Split(line, historyDelimiter)
				if len(parts) == 2 {
					// A leading '#' can be part of a valid legacy filename.
					source, target = parts[0], parts[1]
				} else if strings.HasPrefix(line, "#") {
					if readErr == io.EOF {
						break
					}
					continue
				} else {
					return nil, fmt.Errorf("invalid undo record at line %d", lineNumber)
				}
			}
			if source == "" || target == "" {
				return nil, fmt.Errorf("empty path in undo record at line %d", lineNumber)
			}
			history = append(history, operation{source: source, target: target})
		}
		if readErr == io.EOF {
			return history, nil
		}
	}
	return history, nil
}

// undoOperations restores paths in reverse order and retains unsuccessful work.
// ForceUndo continues after an error; it never grants permission to overwrite.
func undoOperations(opt *Options) error {
	started := time.Now()
	history, err := readHistory(opt.LastOpDetailFile)
	if os.IsNotExist(err) || (err == nil && len(history) == 0) {
		if !opt.Quiet {
			log.Info("no brename operation to undo")
		}
		return nil
	}
	if err != nil {
		return err
	}
	if err := validateHistoryPath(opt.LastOpDetailFile, history); err != nil {
		return err
	}
	completed := make([]bool, len(history)) // Only successful moves leave the retry log.
	var blocked []string                    // Failed restores leave earlier paths at their moved location.
	n := 0
	var firstErr error
	for i := len(history) - 1; i >= 0; i-- {
		op := history[i]
		if opt.DryRun {
			log.Infof("  [UNDO] %s -> %s", op.target, op.source)
			continue
		}
		if undoDependsOn(op, blocked) {
			log.Errorf("  [SKIPPED] %s -> %s: an enclosing path could not be restored", op.target, op.source)
			continue
		}
		err := protectWorkingDirectory(op.target, op.source)
		if err == nil {
			var exists bool
			exists, err = targetExists(op.target, op.source)
			if err == nil && exists {
				err = fmt.Errorf("refusing to overwrite: %s", op.source)
			}
		}
		if err == nil {
			err = os.Rename(op.target, op.source)
		}
		if err != nil {
			log.Errorf("  [ERROR] %s -> %s: %s", op.target, op.source, err)
			if firstErr == nil {
				firstErr = err
			}
			if !opt.ForceUndo {
				break
			}
			blocked = append(blocked, op.source)
			continue
		}
		completed[i] = true
		n++
		if !opt.Quiet {
			log.Infof("  [%s] %s -> %s", green("DONE"), op.target, op.source)
		}
	}
	if opt.DryRun {
		return nil
	}
	if !opt.Quiet {
		log.Infof("%d path(s) renamed back in %.3f seconds", n, time.Since(started).Seconds())
	}
	if firstErr == nil {
		return os.Remove(opt.LastOpDetailFile)
	}
	if n > 0 {
		remaining := make([]operation, 0, len(history)-n)
		for i, op := range history {
			if !completed[i] {
				remaining = append(remaining, op)
			}
		}
		if err := writeHistory(opt.LastOpDetailFile, remaining); err != nil {
			return fmt.Errorf("%v; saving remaining undo operations: %w", firstErr, err)
		}
	}
	return firstErr
}

// undoDependsOn keeps forced undo away from unrelated files that now occupy an
// old directory name when restoring that directory failed. It also handles
// exact-path dependencies in a chain of moves.
func undoDependsOn(op operation, blocked []string) bool {
	for _, parent := range blocked {
		parentKey, err := pathKey(parent)
		if err != nil {
			parentKey, _ = filepath.Abs(parent)
		}
		for _, path := range []string{op.source, op.target} {
			key, err := pathKey(path)
			if err != nil {
				key, _ = filepath.Abs(path)
			}
			if containsPath(parentKey, key) {
				return true
			}
		}
	}
	return false
}
