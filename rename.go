package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// normalizePath retains concise paths unless parent components require physical
// resolution. Resolving the parent preserves a final symlink as a renameable entry.
func normalizePath(path string) (string, error) {
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == ".." {
			return pathKey(path)
		}
	}
	return filepath.Clean(path), nil
}

// pathKey identifies a directory entry through absolute, resolved parent paths.
// The last component is not resolved: renaming a symlink moves the link itself.
func pathKey(path string) (string, error) {
	if !filepath.IsAbs(path) {
		// Prefix the working directory without cleaning: a logical PWD may itself
		// contain a symlink whose parent differs from its textual parent.
		volume := filepath.VolumeName(path)
		base := "."
		if volume != "" {
			base = volume + "." // Resolve Windows drive-relative paths on that drive.
		}
		cwd, err := filepath.Abs(base)
		if err != nil {
			return "", err
		}
		if len(path) > 0 && os.IsPathSeparator(path[0]) {
			path = filepath.VolumeName(cwd) + path // Windows root on the current drive.
		} else {
			path = cwd + string(os.PathSeparator) + path[len(volume):]
		}
	}
	dir, name := filepath.Split(path)
	if name == "" || name == "." || name == ".." {
		return resolveDirectory(path)
	}
	parent, err := resolveDirectory(dir)
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, name), nil
}

// resolveDirectory resolves an absolute directory path, retaining missing suffixes.
// pathKey makes it absolute first so recursive calls need no working-directory logic.
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
	// Split the unresolved path without filepath.Dir's lexical cleaning.
	trimmed := strings.TrimRight(path, string(os.PathSeparator)+"/")
	parentPath, name := filepath.Split(trimmed)
	if !os.IsNotExist(err) || name == "" || name == "." || name == ".." {
		return "", err
	}
	if _, linkErr := os.Lstat(path); linkErr == nil {
		return "", err // A dangling directory symlink is not a missing directory.
	}
	parent, err := resolveDirectory(parentPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, name), nil
}

// containsPath reports whether child is parent itself or an entry below it.
func containsPath(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

// validateDirectoryMove protects the working directory and its ancestors, and
// rejects moving a directory into itself, including through directory aliases.
func validateDirectoryMove(source, target string) error {
	var cwd string // Resolve the working directory only for directory operations.
	for i, path := range []string{source, target} {
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
			cwd, err = pathKey(".")
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
		if i == 0 { // Only a directory source is forbidden from containing its target.
			targetKey, err := pathKey(target)
			if err != nil {
				return err
			}
			if key != targetKey && containsPath(key, targetKey) {
				return fmt.Errorf("cannot move a directory into itself: %s -> %s", source, target)
			}
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
	if !strings.EqualFold(sourceKey, targetKey) {
		return true, nil
	}
	sourceInfo, err := os.Lstat(source)
	if err != nil || !os.SameFile(sourceInfo, info) {
		return true, nil
	}
	// A case-only rename stays in the same directory. Matching basenames
	// in different case-distinct directories can still be hard links.
	sourceParent, err := os.Stat(filepath.Dir(sourceKey))
	if err != nil {
		return false, err
	}
	targetParent, err := os.Stat(filepath.Dir(targetKey))
	if err != nil {
		return false, err
	}
	if !os.SameFile(sourceParent, targetParent) {
		return true, nil
	}
	sourceName, targetName := filepath.Base(sourceKey), filepath.Base(targetKey)
	if sourceName == targetName {
		return false, nil
	}
	// Two distinct directory entries can share an inode on a case-sensitive
	// filesystem. Only one spelling exists for a true case-only rename.
	entries, err := os.ReadDir(filepath.Dir(targetKey))
	if err != nil {
		return false, err
	}
	var sourceFound, targetFound bool
	for _, entry := range entries {
		sourceFound = sourceFound || entry.Name() == sourceName
		targetFound = targetFound || entry.Name() == targetName
	}
	return sourceFound && targetFound, nil
}

// orderOperations keeps moves before any renamed directory or symlink used to
// reach their source or target, including when explicit search roots overlap.
// Moves touching the same entry retain their original order.
func orderOperations(ops []operation) ([]operation, error) {
	if len(ops) < 2 {
		return ops, nil
	}
	keys := make([][4]string, len(ops)) // Physical and lexical source and target paths.
	sources := make(map[string]int, len(ops))
	for i, op := range ops {
		for j, path := range []string{op.source, op.target} {
			physical, err := pathKey(path)
			if err != nil {
				return nil, err
			}
			lexical, err := filepath.Abs(path)
			if err != nil {
				return nil, err
			}
			keys[i][2*j], keys[i][2*j+1] = physical, lexical
		}
		for _, key := range keys[i][:2] {
			sources[key] = i
		}
	}
	dependencies := make([][]int, len(ops)) // Moves that must finish before each operation.
	hasDependencies := false
	addDependency := func(after, before int) {
		if after == before {
			return
		}
		// Several path spellings can identify the same dependency.
		previous := dependencies[after]
		if len(previous) == 0 || previous[len(previous)-1] != before {
			dependencies[after] = append(previous, before)
			hasDependencies = true
		}
	}
	for i, paths := range keys {
		for _, path := range paths {
			for parent := filepath.Dir(path); parent != path; parent = filepath.Dir(path) {
				if ancestor, ok := sources[parent]; ok {
					addDependency(ancestor, i)
				}
				path = parent
			}
		}
	}
	// Append entry-order dependencies after directory dependencies so traversal
	// retains its existing order, including which operations precede a failure.
	lastUse := make(map[string]int, len(ops))
	for i, paths := range keys {
		for _, path := range paths {
			if previous, ok := lastUse[path]; ok {
				addDependency(i, previous)
			}
			lastUse[path] = i
		}
	}
	if !hasDependencies {
		return ops, nil
	}
	ordered := make([]operation, 0, len(ops))
	state := make([]uint8, len(ops)) // 0: unseen, 1: visiting, 2: ordered.
	var visit func(int) error
	visit = func(i int) error {
		if state[i] == 2 {
			return nil
		}
		if state[i] == 1 {
			return fmt.Errorf("cyclic directory dependencies: %s", ops[i].source)
		}
		state[i] = 1
		for _, before := range dependencies[i] {
			if err := visit(before); err != nil {
				return err
			}
		}
		state[i] = 2
		ordered = append(ordered, ops[i])
		return nil
	}
	for i := range ops {
		if err := visit(i); err != nil {
			return nil, err
		}
	}
	return ordered, nil
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
		if err := validateDirectoryMove(op.source, op.target); err != nil {
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
		if exists {
			sourceInfo, err := os.Lstat(op.source)
			if err != nil {
				return n, err
			}
			targetInfo, err := os.Lstat(op.target)
			if err != nil {
				return n, err
			}
			if os.SameFile(sourceInfo, targetInfo) {
				return n, fmt.Errorf("source and target are distinct hard links to the same file: %s -> %s", op.source, op.target)
			}
		}
		if err := os.MkdirAll(filepath.Dir(op.target), 0755); err != nil {
			return n, err
		}
		var recordOffset int64 // End of the last complete, successful record.
		if journal != nil {
			recordOffset, err = journal.Seek(0, io.SeekEnd)
			if err != nil {
				return n, err
			}
			// Reserve and write the record before changing either file name. A full
			// disk or file-size limit must not leave an unrecorded rename behind.
			if err := writeHistoryRecord(journal, op, quoted); err != nil {
				return n, discardHistoryRecord(journal, recordOffset, fmt.Errorf("record %s -> %s: %w", op.source, op.target, err))
			}
		}
		if err := os.Rename(op.source, op.target); err != nil {
			if journal != nil {
				err = discardHistoryRecord(journal, recordOffset, err)
			}
			return n, err
		}
		n++
		if journal != nil {
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

// discardHistoryRecord removes an incomplete write or an unexecuted operation.
func discardHistoryRecord(journal *os.File, offset int64, cause error) error {
	if err := journal.Truncate(offset); err != nil {
		return fmt.Errorf("%w; removing pending history record: %v", cause, err)
	}
	return cause
}
