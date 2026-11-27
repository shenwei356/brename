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
		// The tabs at both ends of the delimiter can also complete a second
		// separator across a field boundary.
		if strings.HasSuffix(op.source, strings.TrimSuffix(historyDelimiter, "\t")) ||
			strings.HasPrefix(op.target, strings.TrimPrefix(historyDelimiter, "\t")) {
			return true
		}
		if strings.HasPrefix(op.source, "#") {
			return true // Older readers treat these source names as comments.
		}
		if strings.HasPrefix(op.source, "\ufeff") {
			return true // Older readers strip a UTF-8 BOM at the start of the log.
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
func validateHistoryPath(path string, ops []operation, ignoreCase bool) error {
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
	if ignoreCase {
		key = foldKey(key)
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
			if ignoreCase {
				entryKey = foldKey(entryKey)
			}
			// A new log also reserves its name against targets beneath it.
			conflict := containsPath(key, entryKey)
			if !conflict && info != nil {
				entryInfo, statErr := os.Lstat(entry)
				conflict = statErr == nil && os.SameFile(info, entryInfo)
			}
			if conflict {
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

// stageHistory reserves space for a complete log without replacing the old one.
func stageHistory(path string, history []operation, quoted bool) (*os.File, error) {
	fh, err := createHistory(path, quoted)
	if err != nil {
		return nil, err
	}
	for _, op := range history {
		if err := writeHistoryRecord(fh, op, quoted); err != nil {
			fh.Close()
			os.Remove(fh.Name())
			return nil, err
		}
	}
	return fh, nil
}

// readHistory validates the entire log before any undo operation is attempted.
func readHistory(path string) (history []operation, err error) {
	if err := validateHistoryPath(path, nil, false); err != nil {
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
					if strings.Index(line, historyDelimiter) != strings.LastIndex(line, historyDelimiter) {
						return nil, fmt.Errorf("ambiguous undo separator at line %d", lineNumber)
					}
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
			// NUL cannot occur in a filesystem path on any supported platform.
			if strings.ContainsRune(source, 0) || strings.ContainsRune(target, 0) {
				return nil, fmt.Errorf("NUL byte in undo record at line %d", lineNumber)
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
	if err := validateHistoryPath(opt.LastOpDetailFile, history, opt.PathCaseInsensitive); err != nil {
		return err
	}
	var journal *os.File // Reserve recovery space before changing any file names.
	keepJournal := false // Retain a valid recovery log if its publication fails.
	quoted := historyNeedsQuoting(history)
	if !opt.DryRun {
		journal, err = stageHistory(opt.LastOpDetailFile, history, quoted)
		if err != nil {
			return fmt.Errorf("prepare undo history: %w", err)
		}
		defer func() {
			journal.Close()
			if !keepJournal {
				os.Remove(journal.Name())
			}
		}()
	}
	completed := make([]bool, len(history)) // Only successful moves leave the retry log.
	var blocked undoBlockers                // Entries and directory links needed to retry failed restores.
	n := 0
	var firstErr error
	for i := len(history) - 1; i >= 0; i-- {
		op := history[i]
		if opt.DryRun {
			log.Infof("  [UNDO] %s -> %s", op.target, op.source)
			continue
		}
		if blocked.dependsOn(op) {
			log.Errorf("  [SKIPPED] %s -> %s: depends on a failed restore", op.target, op.source)
			// Skipping this restore also blocks earlier moves that need its source.
			blocked.add(op)
			continue
		}
		err := validateDirectoryMove(op.target, op.source)
		if err == nil {
			var exists bool
			exists, err = targetExists(op.target, op.source)
			if err == nil && exists {
				err = fmt.Errorf("refusing to overwrite: %s", op.source)
			}
		}
		if err == nil {
			err = renameEntry(op.target, op.source)
		}
		if err != nil {
			log.Errorf("  [ERROR] %s -> %s: %s", op.target, op.source, err)
			if firstErr == nil {
				firstErr = err
			}
			if !opt.ForceUndo {
				break
			}
			blocked.add(op)
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
	if n == 0 {
		return firstErr
	}
	// Compact the reserved file without first releasing its allocated space.
	// The remaining records use the same encoding and cannot exceed its size.
	var offset int64
	if quoted {
		offset = int64(len(historyHeader) + 1)
	}
	if _, err := journal.Seek(offset, io.SeekStart); err != nil {
		return fmt.Errorf("saving remaining undo operations: %w", err)
	}
	for i, op := range history {
		if !completed[i] {
			if err := writeHistoryRecord(journal, op, quoted); err != nil {
				return fmt.Errorf("saving remaining undo operations: %w", err)
			}
		}
	}
	end, err := journal.Seek(0, io.SeekCurrent)
	if err == nil {
		err = journal.Truncate(end)
	}
	if err != nil {
		return fmt.Errorf("saving remaining undo operations: %w", err)
	}
	keepJournal = true
	err = journal.Close()
	if err == nil {
		err = os.Rename(journal.Name(), opt.LastOpDetailFile)
	}
	if err != nil {
		return fmt.Errorf("saving remaining undo operations: %w; recovery log: %s", err, journal.Name())
	}
	keepJournal = false
	if firstErr == nil {
		// Publish an empty log before removing it so a removal error cannot
		// cause a later invocation to replay already completed operations.
		if err := os.Remove(opt.LastOpDetailFile); err != nil {
			return err
		}
	}
	return firstErr
}

// undoBlockers protects both endpoints of failed restores, and the directories
// and links needed to reach them. Siblings can still be restored independently.
type undoBlockers struct {
	entries entrySet
	parents entrySet
}

// add reserves paths for a failed or skipped restore.
func (b *undoBlockers) add(op operation) {
	if b.entries == nil {
		b.entries = make(entrySet)
		b.parents = make(entrySet)
	}
	visitOperationPaths(op, func(key string) bool {
		b.entries.add(key)
		return false
	}, func(key string) bool {
		b.parents.add(key)
		return false
	})
}

// dependsOn checks whether this move needs a blocked entry or would move
// a directory or symlink still needed for a retry.
func (b *undoBlockers) dependsOn(op operation) bool {
	if len(b.entries) == 0 {
		return false
	}
	return visitOperationPaths(op, func(key string) bool {
		return b.entries.contains(key) || b.parents.contains(key)
	}, func(key string) bool {
		return b.entries.contains(key)
	})
}
