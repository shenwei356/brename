// Copyright 2014 Wei Shen (shenwei356@gmail.com). All rights reserved.
// Use of this source code is governed by a MIT-license
// that can be found in the LICENSE file.

// Recursively batch rename files and directories by regular expression.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

var (
	path string // path
	src  string // source regular expression
	repl string // replacement
	R    bool   // recursive
	D    bool   // Rename directories
)

func init() {
	flag.StringVar(&src, "s", "", "Regular expression")
	flag.StringVar(&repl, "r", "", "Replacement")
	flag.BoolVar(&R, "R", false, "Recursively rename")
	flag.BoolVar(&D, "D", false, "Rename directories")

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "\nbrename\n  Recursively batch rename files and directories by regular expression.")
		fmt.Fprintf(os.Stderr, "\nUsage: %s -s <regexp> -r <replacement> [-R] [-D] [path...]\n\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
		fmt.Fprint(os.Stderr, `
Example:
  1. a.jpeg -> a.jpg
     brename -s '\.jpeg$' -r '.jpg'
  2. ab.png -> abab.png
     brename -s '([ab]+)' -r '$1$1'
`)
		fmt.Fprintln(os.Stderr, "\n  Site: https://github.com/shenwei356/brename")
		fmt.Fprintln(os.Stderr, "Author: Wei Shen (shenwei356@gmail.com)\n")
	}

	flag.Parse()
	if src == "" {
		fmt.Fprintln(os.Stderr, "option -s should be set")
		os.Exit(1)
	}
}

func main() {
	re, err := regexp.Compile(src)
	if err != nil {
		recover()
		fmt.Fprintln(os.Stderr, "[Error] Illegal regular expression!")
		return
	}

	var paths []string
	if len(flag.Args()) == 0 {
		paths = []string{"./"}
	} else {
		paths = flag.Args()
	}
	for _, path := range paths {
		fmt.Printf("%s:\n", path)
		n, err := BatchRename(path, re, repl, R, D)
		if err != nil {
			recover()
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		fmt.Printf("%d files be renamed.\n\n", n)
	}
}

func BatchRename(path string, re *regexp.Regexp, repl string, recursive bool, D bool) (uint, error) {
	var n uint = 0

	_, err := ioutil.ReadFile(path)
	// it's a file
	if err == nil {
		n, err = Rename(path, re, repl)
		if err != nil {
			recover()
			fmt.Fprintln(os.Stderr, err)
			return 0, err
		}
		return n, nil
	}

	// it's a directory
	files, err := ioutil.ReadDir(path)
	if err != nil {
		recover()
		if os.IsNotExist(err) {
			return 0, errors.New("[Error] Path not exist: " + path)
		}
		return 0, errors.New("[Error] Path read error: " + path)
	}

	var filename string
	for _, file := range files {
		filename = file.Name()
		if filename == "." || filename == ".." {
			continue
		}

		fileFullPath := filepath.Join(path, filename)
		// sub directory
		if file.IsDir() {
			if recursive {
				num, err := BatchRename(fileFullPath, re, repl, recursive, D)
				if err != nil {
					recover()
					fmt.Fprintln(os.Stderr, err)
					continue
				}
				n += num
			}
			// Rename directories
			if D {
				num, err := Rename(fileFullPath, re, repl)
				if err != nil {
					recover()
					fmt.Fprintln(os.Stderr, err)
					continue
				}
				n += num
			}
		} else {
			num, err := Rename(fileFullPath, re, repl)
			if err != nil {
				recover()
				fmt.Fprintln(os.Stderr, err)
				continue
			}
			n += num
		}
	}
	return n, nil
}

func Rename(path string, re *regexp.Regexp, repl string) (uint, error) {
	dir, filename := filepath.Split(path)

	// not matched
	if !re.Match([]byte(filename)) {
		return 0, nil
	}

	filename2 := re.ReplaceAllString(filename, repl)
	// not changed
	if filename2 == filename {
		return 0, nil
	}

	// duplicated files
	// in windows, rename a file to another existed file will cause err.
	// however, it will succeed in Linux.
	if _, err := os.Stat(filename2); err == nil {
		return 0, errors.New("[Error] Rename file error: " + filename + " -> " + filename2 + " (" + filename2 + " already existed!)")
	}

	// rename
	err := os.Rename(path, filepath.Join(dir, filename2))
	if err != nil {
		return 0, errors.New(fmt.Sprintf("[Error] %v", err))
	}

	return 1, nil
}
