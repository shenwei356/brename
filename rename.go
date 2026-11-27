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

// visitDirectoryEntries visits resolved entries used to traverse a directory,
// including intermediate symlinks and their targets. A true result stops the walk.
func visitDirectoryEntries(path string, visit func(string) bool) bool {
	expanded := make(map[string]bool) // Expand each symlink once, including cycles.
	var walk func(string) bool
	walk = func(path string) bool {
		for {
			trimmed := strings.TrimRight(path, string(os.PathSeparator)+"/")
			if trimmed != "" && trimmed != filepath.VolumeName(path) {
				path = trimmed // Preserve roots, but inspect a final directory link itself.
			}
			key, err := pathKey(path)
			if err != nil {
				// Undo must retain dependencies even after a parent becomes inaccessible.
				key, _ = filepath.Abs(path)
			}
			if visit(key) {
				return true
			}
			if !expanded[key] {
				if target, err := os.Readlink(key); err == nil {
					expanded[key] = true
					if !filepath.IsAbs(target) && filepath.VolumeName(target) == "" {
						if os.IsPathSeparator(target[0]) {
							target = filepath.VolumeName(key) + target
						} else {
							// Do not clean symlink/.. components in the link target.
							target = filepath.Dir(key) + string(os.PathSeparator) + target
						}
					}
					if walk(target) {
						return true
					}
				}
			}
			// Split without lexical cleaning so symlink/.. dependencies survive.
			parent, name := filepath.Split(path)
			if parent == "" || name == "" {
				return false
			}
			path = parent
		}
	}
	return walk(path)
}

// visitOperationPaths visits the two entries a move changes and the directory
// entries it traverses. Returning true from either callback stops the visit.
func visitOperationPaths(op operation, entry, parent func(string) bool) bool {
	for _, path := range []string{op.source, op.target} {
		key, err := pathKey(path)
		if err != nil {
			key, _ = filepath.Abs(path) // Keep inaccessible paths blocked for undo.
		}
		if entry(key) {
			return true
		}
		dir, _ := filepath.Split(path)
		if dir == "" {
			dir = "."
		}
		if visitDirectoryEntries(dir, parent) {
			return true
		}
	}
	return false
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

// entrySet compares resolved entry paths. Case-folded spelling only selects
// candidates: distinct files and hard links on case-sensitive volumes stay distinct.
type entrySet map[string][]string

// contains checks an existing bucket, including exact paths that are now missing.
func (s entrySet) contains(key string) bool {
	for _, previous := range s[foldKey(key)] {
		if previous == key {
			return true
		}
		if _, err := os.Lstat(key); err == nil {
			if exists, err := targetExists(previous, key); err == nil && !exists {
				return true
			}
		}
	}
	return false
}

// add records an entry once and reports whether it was newly added.
func (s entrySet) add(key string) bool {
	if s.contains(key) {
		return false
	}
	folded := foldKey(key)
	s[folded] = append(s[folded], key)
	return true
}

// renameEntry handles filesystems where a direct case-only rename succeeds
// without changing the stored spelling. The temporary directory reserves a
// private sibling name for the move and for recovery if rollback also fails.
func renameEntry(source, target string) error {
	oldName, newName := filepath.Base(source), filepath.Base(target)
	if oldName == newName || !strings.EqualFold(oldName, newName) {
		return os.Rename(source, target)
	}
	same, err := sameFileEntries(source, target)
	if os.IsNotExist(err) || (err == nil && !same) {
		return os.Rename(source, target)
	}
	if err != nil {
		return err
	}
	// Callers have ruled out distinct hard links. Stage the case-only move
	// before changing the name, so a preparation error leaves history valid.
	dir, err := os.MkdirTemp(filepath.Dir(target), ".brename-case-*")
	if err != nil {
		return err
	}
	defer os.Remove(dir) // An occupied recovery directory is never removed.
	staged := filepath.Join(dir, "entry")
	if err := os.Rename(source, staged); err != nil {
		return err
	}
	if err := os.Rename(staged, target); err != nil {
		if restoreErr := os.Rename(staged, source); restoreErr != nil {
			return fmt.Errorf("%w; restoring %s: %v; entry retained at %s", err, source, restoreErr, staged)
		}
		return err
	}
	return nil
}

// orderOperations keeps moves before any renamed directory or symlink used to
// reach their source or target, including when explicit search roots overlap.
// Moves touching the same entry retain their original order.
func orderOperations(ops []operation) ([]operation, error) {
	if len(ops) == 0 {
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
	// Cross-directory replacements can require the same new path as both a
	// file and a directory. Check this before executing either move.
	for _, op := range ops {
		if filepath.Dir(op.source) != filepath.Dir(op.target) {
			if err := orderTargetParents(ops, keys, sources, addDependency); err != nil {
				return nil, err
			}
			break
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
	// Endpoint keys hide links followed inside another link's target. Inspect
	// those paths separately; ordinary paths need no additional filesystem walk.
	for i, op := range ops {
		for j, path := range []string{op.source, op.target} {
			if keys[i][2*j] == keys[i][2*j+1] {
				continue
			}
			parent, _ := filepath.Split(path)
			if parent == "" {
				parent = "."
			}
			visitDirectoryEntries(parent, func(key string) bool {
				if ancestor, ok := sources[key]; ok {
					if ancestor == i {
						// This move invalidates its own directory alias. Keep stable
						// entry paths for execution and the subsequent undo record.
						ops[i].source, ops[i].target = keys[i][0], keys[i][2]
					} else {
						addDependency(ancestor, i)
					}
				}
				return false
			})
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

// orderTargetParents places a new directory before moves that target its contents
// and rejects using a file as a parent. A move chain can change an entry's type.
func orderTargetParents(ops []operation, keys [][4]string, sources map[string]int, addDependency func(int, int)) error {
	writers := make(map[string]int, len(ops))
	for i, paths := range keys {
		key := paths[2]
		if _, ok := writers[key]; ok {
			writers[key] = -1
		} else {
			writers[key] = i
		}
	}
	for childIndex, paths := range keys {
		for child := paths[2]; ; {
			parent := filepath.Dir(child)
			if parent == child {
				break
			}
			child = parent
			i, ok := writers[parent]
			if !ok || i < 0 {
				continue
			}
			if _, movesAway := sources[parent]; movesAway {
				continue
			}
			if _, replaced := writers[keys[i][0]]; replaced {
				continue
			}
			info, err := os.Lstat(ops[i].source)
			if err != nil {
				return err
			}
			if !info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
				return fmt.Errorf("target %s is also needed as a directory for %s", parent, paths[2])
			}
			if ops[i].code == codeOK {
				addDependency(childIndex, i)
			}
		}
	}
	return nil
}

// sameFileEntries checks whether two existing names refer to the same inode.
func sameFileEntries(source, target string) (bool, error) {
	s, err := os.Lstat(source)
	if err != nil {
		return false, err
	}
	t, err := os.Lstat(target)
	return err == nil && os.SameFile(s, t), err
}

// validateHardlinkOverwrites rejects known no-op overwrites before any move.
// Earlier operations may replace either link or a directory used to reach it.
func validateHardlinkOverwrites(ops []operation) error {
	var modified entrySet
	checked := 0
	for i, op := range ops {
		if op.code != codeExisted {
			continue
		}
		same, err := sameFileEntries(op.source, op.target)
		if err != nil {
			return err
		}
		if !same {
			continue
		}
		if modified == nil {
			modified = make(entrySet)
		}
		for ; checked < i; checked++ {
			for _, path := range []string{ops[checked].source, ops[checked].target} {
				key, err := pathKey(path)
				if err != nil {
					return err
				}
				modified.add(key)
			}
		}
		if !visitOperationPaths(op, modified.contains, modified.contains) {
			return fmt.Errorf("source and target are distinct hard links to the same file: %s -> %s", op.source, op.target)
		}
	}
	return nil
}

// executeOperations performs the checked plan and records each successful move.
// Returning errors lets the journal close normally after a partial failure.
func executeOperations(opt *Options, ops []operation) (n int, err error) {
	var journal *os.File               // Unbuffered writes preserve completed operations on errors.
	var pending string                 // Keep the previous log until the first move succeeds.
	quoted := historyNeedsQuoting(ops) // Ordinary logs remain readable by older builds.
	if !opt.DisableUndo {
		if err := validateHistoryPath(opt.LastOpDetailFile, ops, opt.PathCaseInsensitive); err != nil {
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
		// A fresh log has no inode to compare with a differently cased target.
		// Its staged sibling lets execution check the actual volume even without -w.
		if !opt.PathCaseInsensitive {
			alias := filepath.Join(filepath.Dir(pending), strings.ToUpper(filepath.Base(pending)))
			same, probeErr := sameFileEntries(pending, alias)
			if probeErr != nil && !os.IsNotExist(probeErr) {
				return 0, probeErr
			}
			if same {
				if err := validateHistoryPath(opt.LastOpDetailFile, ops, true); err != nil {
					return 0, err
				}
			}
		}
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
			same, err := sameFileEntries(op.source, op.target)
			if err != nil {
				return n, err
			}
			if same {
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
		if err := renameEntry(op.source, op.target); err != nil {
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
