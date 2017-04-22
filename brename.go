// Copyright © 2017 Wei Shen <shenwei356@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/op/go-logging"
	"github.com/spf13/cobra"
)

var log *logging.Logger

var VERSION = "0.2.0"
var APP = "brename"

// Options is the struct containing all global options
type Options struct {
	Verbose int
	Version bool
	DryRun  bool

	Pattern      string
	PatternRe    *regexp.Regexp
	Replacement  string
	Recursive    bool
	IncludingDir bool
	IgnoreCase   bool
}

func getOptions(cmd *cobra.Command) *Options {
	version := getFlagBool(cmd, "version")
	if version{
		checkVersion()
		return &Options{Version: version}
	}

	pattern := getFlagString(cmd, "pattern")
	if pattern == "" {
		log.Errorf("flag -p/--pattern needed")
		os.Exit(1)
	}
	p := pattern
	if getFlagBool(cmd, "ignore-case") {
		p = "(?i)" + p
	}
	re, err := regexp.Compile(p)
	if err != nil {
		log.Errorf("Illegal regular expression: %s", pattern)
		os.Exit(1)
	}

	return &Options{
		Verbose: getFlagNonNegativeInt(cmd, "verbose"),
		Version: version,
		DryRun:  getFlagBool(cmd, "dry-run"),

		Pattern:      pattern,
		PatternRe:    re,
		Replacement:  getFlagString(cmd, "replacement"),
		Recursive:    getFlagBool(cmd, "recursive"),
		IncludingDir: getFlagBool(cmd, "including-dir"),
	}
}

func init() {
	logFormat := logging.MustStringFormatter(`%{color}[%{level:.4s}]%{color:reset} %{message}`)
	backend := logging.NewLogBackend(os.Stderr, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, logFormat)
	logging.SetBackend(backendFormatter)
	log = logging.MustGetLogger(APP)

	RootCmd.Flags().IntP("verbose", "v", 0, "verbose level (0 for all, 1 for warning and error, 2 for only error)")
	RootCmd.Flags().BoolP("version", "V", false, "print version information and check for update")
	RootCmd.Flags().BoolP("dry-run", "d", false, "print rename operations but do not run")

	RootCmd.Flags().StringP("pattern", "p", "", "search pattern (regular expression)")
	RootCmd.Flags().StringP("replacement", "r", "", "replacement")
	RootCmd.Flags().BoolP("recursive", "R", false, "rename recursively")
	RootCmd.Flags().BoolP("including-dir", "D", false, "rename directories")
	RootCmd.Flags().BoolP("ignore-case", "i", false, "ignore case")

	RootCmd.Example = `  1. renaming all .jpeg files to .jpg in all subdirs
      brename -p "\.jpeg" -r ".jpg" -R   dir
  2. doubling "a"
      brename -p "(a)" -r "\$1\$1"  or brename -p "(a)" -r '$1$1'
  3. even renaming directory
      brename -p ":" -r "-" -R -D   pdf-dirs
  4. dry run and showing potential dangerous operation
      brename -p "abc" -d -v 2

  More examples: https://github.com/shenwei356/brename`

	RootCmd.SetUsageTemplate(`Usage:{{if .Runnable}}
  {{if .HasAvailableFlags}}{{appendIfNotPresent .UseLine "[flags]"}}{{else}}{{.UseLine}}{{end}}{{end}}{{if .HasAvailableSubCommands}}
  {{ .CommandPath}} [command]{{end}} {{if gt .Aliases 0}}

Aliases:
  {{.NameAndAliases}}
{{end}}{{if .HasExample}}

Examples:
{{ .Example }}{{end}}{{ if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if .IsAvailableCommand}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{ if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimRightSpace}}{{end}}{{ if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimRightSpace}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsHelpCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{ if .HasAvailableSubCommands }}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`)
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		log.Error(err)
		os.Exit(-1)
	}
}

func checkError(err error) {
	if err != nil {
		log.Error(err)
		os.Exit(-1)
	}
}

func getFileList(args []string) []string {
	files := []string{}
	if len(args) == 0 {
		files = append(files, "./")
	} else {
		for _, file := range files {
			if file == "./" {
				continue
			}
			if _, err := os.Stat(file); os.IsNotExist(err) {
				checkError(err)
			}
		}
		files = args
	}
	return files
}

func getFlagBool(cmd *cobra.Command, flag string) bool {
	value, err := cmd.Flags().GetBool(flag)
	checkError(err)
	return value
}

func getFlagString(cmd *cobra.Command, flag string) string {
	value, err := cmd.Flags().GetString(flag)
	checkError(err)
	return value
}

func getFlagNonNegativeInt(cmd *cobra.Command, flag string) int {
	value, err := cmd.Flags().GetInt(flag)
	checkError(err)
	if value < 0 {
		checkError(fmt.Errorf("value of flag --%s should be greater than or equal to 0", flag))
	}
	return value
}

func checkVersion() {
	app := APP
	fmt.Printf("%s v%s\n", app, VERSION)
	fmt.Println("\nChecking new version...")

	resp, err := http.Get(fmt.Sprintf("https://github.com/shenwei356/%s/releases/latest", app))
	if err != nil {
		checkError(fmt.Errorf("Network error"))
	}
	items := strings.Split(resp.Request.URL.String(), "/")
	version := ""
	if items[len(items)-1] == "" {
		version = items[len(items)-2]
	} else {
		version = items[len(items)-1]
	}
	if version == "v"+VERSION {
		fmt.Printf("You are using the latest version of %s\n", app)
	} else {
		fmt.Printf("New version available: %s %s at %s\n", app, version, resp.Request.URL.String())
	}
}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   APP,
	Short: "a cross-platform command-line tool for batch renaming files/directories",
	Long: fmt.Sprintf(`
brename -- a cross-platform command-line tool for batch renaming files/directories

Version: %s

Author: Wei Shen <shenwei356@gmail.com>

Homepage: https://github.com/shenwei356/brename

`, VERSION),
	Run: func(cmd *cobra.Command, args []string) {
		// var err error
		opt := getOptions(cmd)

		if opt.Version {
			return
		}

		ops := make([]operation, 0, 100)
		var err error

		for _, path := range getFileList(args) {
			ops, err = walk(opt, ops, path)
			checkError(err)
		}

		var hasErr bool
		for _, op := range ops {
			if int(op.code) >= opt.Verbose {
				log.Infof("TODO: %s\n", op)
			}

			if int(op.code) > 1 {
				hasErr = true
			}
		}

		if opt.DryRun {
			return
		}

		if hasErr {
			checkError(errors.New("potential errors will occur, please check!!!"))

		}

		for _, op := range ops {
			err := os.Rename(op.source, op.target)
			if err != nil {
				log.Errorf("fail to rename: %s -> %s", op.source, op.target)
				os.Exit(-1)
			}
			if opt.Verbose >= 0 {
				log.Infof("renamed: %s -> %s", op.source, op.target)
			}
		}
	},
}

type code int

const (
	codeOK code = iota
	codeNoChange
	codeExisted
	codeMissingTarget
)

var yellow = color.New(color.FgYellow).SprintFunc()
var red = color.New(color.FgRed).SprintFunc()

func (c code) String() string {
	switch c {
	case codeOK:
		return "ok"
	case codeNoChange:
		return yellow("not changed")
	case codeExisted:
		return red("new path existed")
	case codeMissingTarget:
		return red("missing target")
	}
	return "undefined code"
}

type operation struct {
	source, target string
	code           code
}

func (op operation) String() string {
	return fmt.Sprintf("%s -> %s: %s", op.source, op.target, op.code)
}

func check(opt *Options, path string) (bool, operation) {
	dir, filename := filepath.Split(path)

	if !opt.PatternRe.MatchString(filename) {
		return false, operation{}
	}

	filename2 := opt.PatternRe.ReplaceAllString(filename, opt.Replacement)
	if filename2 == "" {
		return true, operation{path, filename2, codeMissingTarget}
	}

	if filename2 == filename {
		return true, operation{path, filepath.Join(dir, filename2), codeNoChange}
	}

	target := filepath.Join(dir, filename2)
	if _, err := os.Stat(target); err == nil {
		return true, operation{path, target, codeExisted}
	}

	return true, operation{path, filepath.Join(dir, filename2), codeOK}
}

func walk(opt *Options, ops []operation, path string) ([]operation, error) {
	_, err := ioutil.ReadFile(path)
	// it's a file
	if err == nil {
		if ok, op := check(opt, path); ok {
			ops = append(ops, op)
		}
		return ops, nil
	}

	// it's a directory
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return ops, err
	}

	var filename string
	for _, file := range files {
		filename = file.Name()
		if filename[0] == '.' {
			continue
		}

		fileFullPath := filepath.Join(path, filename)
		// sub directory
		if file.IsDir() {
			if opt.Recursive {
				var err error
				ops, err = walk(opt, ops, fileFullPath)
				if err != nil {
					return ops, err
				}
			}
			// Rename directories
			if opt.IncludingDir {
				if ok, op := check(opt, fileFullPath); ok {
					ops = append(ops, op)
				}
			}
		} else {
			if ok, op := check(opt, fileFullPath); ok {
				ops = append(ops, op)
			}
		}
	}

	return ops, nil
}
