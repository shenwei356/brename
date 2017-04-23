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
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/op/go-logging"
	"github.com/spf13/cobra"
)

var log *logging.Logger

var version = "2.0"
var app = "brename"

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
	if version {
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
		log.Errorf("illegal regular expression for search pattern: %s", pattern)
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
	log = logging.MustGetLogger(app)

	RootCmd.Flags().IntP("verbose", "v", 0, "verbose level (0 for all, 1 for warning and error, 2 for only error)")
	RootCmd.Flags().BoolP("version", "V", false, "print version information and check for update")
	RootCmd.Flags().BoolP("dry-run", "d", false, "print rename operations but do not run")

	RootCmd.Flags().StringP("pattern", "p", "", "search pattern (regular expression)")
	RootCmd.Flags().StringP("replacement", "r", "", `replacement. capture variables supported.  e.g. $1 represents the first submatch. ATTENTION: for *nix OS, use SINGLE quote NOT double quotes or use the \ escape character.`)
	RootCmd.Flags().BoolP("recursive", "R", false, "rename recursively")
	RootCmd.Flags().BoolP("including-dir", "D", false, "rename directories")
	RootCmd.Flags().BoolP("ignore-case", "i", false, "ignore case")

	RootCmd.Example = `  1. dry run and showing potential dangerous operations
      brename -p "abc" -d
  2. dry run and only show operations that will cause error
      brename -p "abc" -d -v 2
  3. renaming all .jpeg files to .jpg in all subdirectories
      brename -p "\.jpeg" -r ".jpg" -R   dir
  4. using capture variables, e.g., $1, $2 ...
      brename -p "(a)" -r "\$1\$1"
      or brename -p "(a)" -r '$1$1' in Linux/Mac OS X
  5. even renaming directory
      brename -p ":" -r "-" -R -D   pdf-dirs

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

Use "{{.CommandPath}} --help" for more information about a command.{{end}}
`)
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}

func checkError(err error) {
	if err != nil {
		log.Error(err)
		os.Exit(1)
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
	fmt.Printf("%s v%s\n", app, version)
	fmt.Println("\nChecking new version...")

	resp, err := http.Get(fmt.Sprintf("https://github.com/shenwei356/%s/releases/latest", app))
	if err != nil {
		checkError(fmt.Errorf("Network error"))
	}
	items := strings.Split(resp.Request.URL.String(), "/")
	var version string
	if items[len(items)-1] == "" {
		version = items[len(items)-2]
	} else {
		version = items[len(items)-1]
	}
	if version == "v"+version {
		fmt.Printf("You are using the latest version of %s\n", app)
	} else {
		fmt.Printf("New version available: %s %s at %s\n", app, version, resp.Request.URL.String())
	}
}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   app,
	Short: "a cross-platform command-line tool for safely batch renaming files/directories",
	Long: fmt.Sprintf(`
brename -- a cross-platform command-line tool for safely batch renaming files/directories

Version: %s

Author: Wei Shen <shenwei356@gmail.com>

Homepage: https://github.com/shenwei356/brename

Attention:
  1. Paths starting with "." is ignored
  2. Overwriting existed files is not allowed

`, version),
	Run: func(cmd *cobra.Command, args []string) {
		// var err error
		opt := getOptions(cmd)

		if opt.Version {
			return
		}

		ops := make([]operation, 0, 1000)
		opCH := make(chan operation, 100)
		done := make(chan int)

		var hasErr bool
		var n, nErr int

		go func() {
			for op := range opCH {
				if int(op.code) >= opt.Verbose {
					log.Infof("checking: %s\n", op)
				}

				switch int(op.code) {
				case 0:
					ops = append(ops, op)
					n++
				case 1:
				default:
					hasErr = true
					nErr++
					continue
				}
			}
			done <- 1
		}()

		var err error
		for _, path := range getFileList(args) {
			err = walk(opt, opCH, path)
			if err != nil {
				close(opCH)
				checkError(err)
			}
		}
		close(opCH)
		<-done

		if hasErr {
			log.Errorf("%d potential errors detected, please check", nErr)
			os.Exit(1)
		}

		log.Infof("%d paths to be renamed", n)
		if n == 0 {
			return
		}

		if opt.DryRun {
			return
		}

		var n2 int
		for _, op := range ops {
			err := os.Rename(op.source, op.target)
			if err != nil {
				log.Errorf("fail to rename: %s -> %s", op.source, op.target)
				os.Exit(1)
			}
			log.Infof("renamed: %s -> %s", op.source, op.target)
			n2++
		}

		log.Infof("%d paths renamed", n2)
	},
}

type code int

const (
	codeOK code = iota
	codeUnchanged
	codeExisted
	codeMissingTarget
)

var yellow = color.New(color.FgYellow).SprintFunc()
var red = color.New(color.FgRed).SprintFunc()
var green = color.New(color.FgGreen).SprintFunc()

var isWindows = runtime.GOOS == "windows"

func (c code) String() string {
	if isWindows {
		switch c {
		case codeOK:
			return "ok"
		case codeUnchanged:
			return "unchanged"
		case codeExisted:
			return "new path existed"
		case codeMissingTarget:
			return "missing target"
		}
	}

	switch c {
	case codeOK:
		return green("ok")
	case codeUnchanged:
		return yellow("unchanged")
	case codeExisted:
		return red("new path existed")
	case codeMissingTarget:
		return red("missing target")
	}

	return "undefined code"
}

type operation struct {
	source string
	target string
	code   code
}

func (op operation) String() string {
	return fmt.Sprintf("%s -> %s [%s]", op.source, op.target, op.code)
}

func checkOperation(opt *Options, path string) (bool, operation) {
	dir, filename := filepath.Split(path)

	if !opt.PatternRe.MatchString(filename) {
		return false, operation{}
	}

	filename2 := opt.PatternRe.ReplaceAllString(filename, opt.Replacement)
	if filename2 == "" {
		return true, operation{path, filename2, codeMissingTarget}
	}

	if filename2 == filename {
		return true, operation{path, filepath.Join(dir, filename2), codeUnchanged}
	}

	target := filepath.Join(dir, filename2)
	if _, err := os.Stat(target); err == nil {
		return true, operation{path, target, codeExisted}
	}

	return true, operation{path, filepath.Join(dir, filename2), codeOK}
}

func walk(opt *Options, opCh chan<- operation, path string) error {
	_, err := ioutil.ReadFile(path)
	// it's a file
	if err == nil {
		if ok, op := checkOperation(opt, path); ok {
			opCh <- op
		}
		return nil
	}

	// it's a directory
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
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
				err := walk(opt, opCh, fileFullPath)
				if err != nil {
					return err
				}
			}
			// rename directories
			if opt.IncludingDir {
				if ok, op := checkOperation(opt, fileFullPath); ok {
					opCh <- op
				}
			}
		} else {
			if ok, op := checkOperation(opt, fileFullPath); ok {
				opCh <- op
			}
		}
	}

	return nil
}
