// Copyright © 2013-2024 Wei Shen <shenwei356@gmail.com>
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
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/mattn/go-colorable"
	"github.com/shenwei356/go-logging"
	"github.com/shenwei356/natsort"
	"github.com/shenwei356/util/pathutil"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

var log *logging.Logger

var VERSION = "2.14.1"
var app = "brename"
var LastOpDetailFile = ".brename_detail.txt"

// for detecting one case where two or more files are renamed to same new path
var pathTree map[string]struct{}

// Options is the struct containing all global options
type Options struct {
	Quiet   bool
	Verbose int
	Version bool
	DryRun  bool

	Pattern      string
	PatternRe    *regexp.Regexp
	Replacement  string
	Recursive    bool
	IncludingDir bool
	OnlyDir      bool
	MaxDepth     int
	IgnoreCase   bool
	IgnoreExt    bool
	IgnoreErr    bool

	IncludeFilters   []string
	SkipFilters      []string
	ExcludeFilters   []string
	IncludeFilterRes []*regexp.Regexp
	SkipFilterRes    []*regexp.Regexp
	ExcludeFilterRes []*regexp.Regexp

	ListPath    bool
	ListPathSep string
	ListAbsPath bool
	NatureSort  bool

	ReplaceWithNR bool
	StartNum      int
	NRFormat      string

	ReplaceWithKV bool
	KVs           map[string]string
	KVFile        string
	KeepKey       bool
	KeyCaptIdx    int
	KeyMissRepl   string

	OverwriteMode int

	PathCaseInsensitive bool

	Undo             bool
	ForceUndo        bool
	LastOpDetailFile string

	DisableUndo        bool
	ClearOpDetailFiles bool
}

var reNR = regexp.MustCompile(`\{(NR|nr)\}`)
var reKV = regexp.MustCompile(`\{(KV|kv)\}`)

func getOptions(cmd *cobra.Command) *Options {
	dryrun := getFlagBool(cmd, "dry-run")
	quiet := getFlagBool(cmd, "quiet")
	if dryrun && quiet {
		quiet = false
	}

	undo := getFlagBool(cmd, "undo")
	forceUndo := getFlagBool(cmd, "force-undo")
	if undo || forceUndo {
		return &Options{
			Undo:             true, // set it true even only force-undo given
			DryRun:           dryrun,
			Quiet:            quiet,
			ForceUndo:        forceUndo,
			LastOpDetailFile: LastOpDetailFile,
		}
	}

	clearLastOpDetailFiles := getFlagBool(cmd, "clear")
	if clearLastOpDetailFiles {
		return &Options{
			ClearOpDetailFiles: true,
			DryRun:             dryrun,
			MaxDepth:           getFlagNonNegativeInt(cmd, "max-depth"),
			Quiet:              quiet,
			LastOpDetailFile:   LastOpDetailFile,
			Recursive:          getFlagBool(cmd, "recursive"),
		}
	}

	disableUndo := getFlagBool(cmd, "disable-undo")

	version := getFlagBool(cmd, "version")
	if version {
		checkVersion()
		return &Options{Version: version}
	}

	pattern := getFlagString(cmd, "pattern")
	if pattern == "" {
		log.Errorf(`flag -p/--pattern needed. type "brename -h" for usage and examples.`)
		os.Exit(1)
	}
	p := pattern
	ignoreCase := getFlagBool(cmd, "ignore-case")
	if ignoreCase {
		p = "(?i)" + p
	}
	re, err := regexp.Compile(p)
	if err != nil {
		log.Errorf("illegal regular expression for search pattern: %s", pattern)
		os.Exit(1)
	}

	rewildcard := regexp.MustCompile(`^\*`)

	infilters := getFlagStringSlice(cmd, "include-filters")
	infilterRes := make([]*regexp.Regexp, 0, 10)
	for _, infilter := range infilters {
		if infilter == "" {
			log.Errorf("value of flag -f/--include-filters missing")
			os.Exit(1)
		}
		if rewildcard.MatchString(infilter) {
			log.Warningf("Are you using wildcard for -f/--include-filters? It should be regular expression: %s", infilter)
		}
		if !(infilter == "./" || infilter == "." || infilter == "..") {
			existed, err := pathutil.Exists(infilter)
			if err != nil {
				log.Warningf("something wrong when trying to check whether %s is a existed file", infilter)
			}
			if existed {
				log.Warning("Seems you are using wildcard for -f/--include-filters? Make sure using regular expression: %s", infilter)
			}
		}

		var infilterRe *regexp.Regexp
		if ignoreCase {
			infilterRe, err = regexp.Compile("(?i)" + infilter)
		} else {
			infilterRe, err = regexp.Compile(infilter)
		}
		if err != nil {
			log.Errorf("illegal regular expression for include filter: %s", infilter)
			os.Exit(1)
		}
		infilterRes = append(infilterRes, infilterRe)
	}

	skipfilters := getFlagStringSlice(cmd, "skip-filters")
	skipRes := make([]*regexp.Regexp, 0, 10)
	for _, skipfilter := range skipfilters {
		if skipfilter == "" {
			log.Errorf("value of flag -S/--skip-filters missing")
			os.Exit(1)
		}
		if rewildcard.MatchString(skipfilter) {
			log.Warningf("Are you using wildcard for -S/--skip-filters? It should be regular expression: %s", skipfilter)
		}
		var exfilterRe *regexp.Regexp
		if ignoreCase {
			exfilterRe, err = regexp.Compile("(?i)" + skipfilter)
		} else {
			exfilterRe, err = regexp.Compile(skipfilter)
		}
		if err != nil {
			log.Errorf("illegal regular expression for skip filter: %s", skipfilter)
			os.Exit(1)
		}
		skipRes = append(skipRes, exfilterRe)
	}

	exfilters := getFlagStringSlice(cmd, "exclude-filters")
	exfilterRes := make([]*regexp.Regexp, 0, 10)
	for _, exfilter := range exfilters {
		if exfilter == "" {
			log.Errorf("value of flag -F/--exclude-filters missing")
			os.Exit(1)
		}
		if rewildcard.MatchString(exfilter) {
			log.Warningf("Are you using wildcard for -F/--exclude-filters? It should be regular expression: %s", exfilter)
		}
		if !(exfilter == "./" || exfilter == "." || exfilter == "..") {
			existed, err := pathutil.Exists(exfilter)
			if err != nil {
				log.Warningf("something wrong when trying to check whether %s is a existed file", exfilter)
			}
			if existed {
				log.Warningf("Seems you are using wildcard for -F/--exclude-filters? Make sure using regular expression: %s", exfilter)
			}
		}

		var exfilterRe *regexp.Regexp
		if ignoreCase {
			exfilterRe, err = regexp.Compile("(?i)" + exfilter)
		} else {
			exfilterRe, err = regexp.Compile(exfilter)
		}
		if err != nil {
			log.Errorf("illegal regular expression for exclude filter: %s", exfilter)
			os.Exit(1)
		}
		exfilterRes = append(exfilterRes, exfilterRe)
	}

	replacement := getFlagString(cmd, "replacement")
	kvFile := getFlagString(cmd, "kv-file")

	if kvFile != "" {
		if len(replacement) == 0 {
			checkError(fmt.Errorf("flag -r/--replacement needed when given flag -k/--kv-file"))
		}
		if !reKV.MatchString(replacement) {
			checkError(fmt.Errorf(`replacement symbol "{kv}"/"{KV}" not found in value of flag -r/--replacement when flag -k/--kv-file given`))
		}
	}

	var replaceWithNR bool
	if reNR.MatchString(replacement) {
		replaceWithNR = true
	}

	var replaceWithKV bool
	var kvs map[string]string
	keepKey := getFlagBool(cmd, "keep-key")
	keyMissRepl := getFlagString(cmd, "key-miss-repl")
	if reKV.MatchString(replacement) {
		replaceWithKV = true
		if !regexp.MustCompile(`\(.+\)`).MatchString(pattern) {
			checkError(fmt.Errorf(`value of -p/--pattern must contains "(" and ")" to capture data which is used specify the KEY`))
		}
		if kvFile == "" {
			checkError(fmt.Errorf(`since replacement symbol "{kv}"/"{KV}" found in value of flag -r/--replacement, tab-delimited key-value file should be given by flag -k/--kv-file`))
		}

		if keepKey && keyMissRepl != "" && !quiet {
			log.Warning("flag -m/--key-miss-repl ignored when flag -K/--keep-key given")
		}
		if !quiet {
			log.Infof("read key-value file: %s", kvFile)
		}
		kvs, err = readKVs(kvFile, ignoreCase)
		if err != nil {
			checkError(fmt.Errorf("read key-value file: %s", err))
		}
		if len(kvs) == 0 {
			checkError(fmt.Errorf("no valid data in key-value file: %s", kvFile))
		}

		if !quiet {
			log.Infof("%d pairs of key-value loaded", len(kvs))
		}
	}

	verbose := getFlagNonNegativeInt(cmd, "verbose")
	if verbose > 2 {
		log.Errorf("illegal value of flag --verbose: %d, only 0/1/2 allowed", verbose)
		os.Exit(1)
	}

	overwriteMode := getFlagNonNegativeInt(cmd, "overwrite-mode")
	overwriteModes := []string{"reporting error", "overwrite", "do not rename"}
	if overwriteMode > 2 {
		log.Errorf("illegal value of flag -o/--overwrite-mode: %d, only 0/1/2 allowed", overwriteMode)
		os.Exit(1)
	}

	if !quiet && verbose == 0 {
		log.Infof("brename v%s", VERSION)
		log.Infof("https://github.com/shenwei356/brename")

		log.Info()
	}

	pathCaseInsensitive := getFlagBool(cmd, "case-insensitive-path")
	pathCaseSensitive := getFlagBool(cmd, "case-sensitive-path")
	if !pathCaseInsensitive {
		if runtime.GOOS == "windows" {
			if !pathCaseSensitive {
				if !quiet && verbose <= 1 {
					log.Warning()
					log.Warning("The flag -w/--case-insensitive-path is switched on Windows by default, ")
					log.Warning("where the path is case-insensitive in file systems like FAT32 and NTFS.")
					log.Warning("If you are using a file system in which paths are case-sensitive,")
					log.Warning("please use -W/--case-sensitive-path.")
					log.Warning()
				}
				pathCaseInsensitive = true
			} else {
				if !quiet && verbose <= 1 {
					log.Info()
					log.Info("You've switched on the flag -W/--case-sensitive-path, which means")
					log.Info("you believe that the paths are case-sensitive.")
					log.Info()
				}
			}
		} else {
			if !pathCaseSensitive && !quiet && verbose <= 1 {
				log.Warning()
				log.Warning("If the file system where the search path locates is FAT32 or NTFS (most on Windows),")
				log.Warning("please use -w/--case-insensitive-path to correctly check file overwrites!")
				log.Warning()
			}
		}
	} else if pathCaseSensitive {
		checkError(fmt.Errorf("the flag -w/--case-insensitive-path and -W/--case-sensitive-path are incompatible"))
	}

	recursive := getFlagBool(cmd, "recursive")
	includingDir := getFlagBool(cmd, "including-dir")
	onlyDir := getFlagBool(cmd, "only-dir")
	maxDepth := getFlagNonNegativeInt(cmd, "max-depth")
	onlyList := getFlagBool(cmd, "list")

	if !quiet && verbose == 0 {
		log.Info("---------------- main options ------------------------")
		log.Info()

		log.Info("search mode:")
		log.Infof(" recursively rename: %v", recursive)
		log.Infof("      maximum depth: %d (0 for no limit)", maxDepth)
		log.Infof("  include directory: %v", includingDir)
		log.Infof("     only directory: %v", onlyDir)
		log.Info()

		log.Info("path filters and search pattern:")
		log.Infof("   search pattern: %s", p)
		log.Infof("      replacement: %s", replacement)
		log.Infof("      ignore case: %v", ignoreCase)
		log.Info()

		if len(infilters) > 0 {
			log.Infof("     skip filters: %s", strings.Join(skipfilters, ", "))
		}
		if len(exfilters) > 0 {
			log.Infof("  exclude filters: %s", strings.Join(exfilters, ", "))
		}
		if len(infilters) > 0 {
			log.Infof("  include filters: %s", strings.Join(infilters, ", "))
		}
		log.Info()

		log.Info("path overwrite checking:")
		log.Infof("  case-insensitive path: %v", pathCaseInsensitive)
		log.Infof("         overwrite mode: %d (%s)", overwriteMode, overwriteModes[overwriteMode])
		log.Info()

		log.Info("miscellaneous:")
		log.Infof("     disable undo: %v", disableUndo)
		log.Infof("  only list paths: %v", onlyList)
		log.Infof("          dry run: %v", dryrun)
		log.Info()
	}

	return &Options{
		Quiet:   quiet,
		Verbose: verbose,
		Version: version,
		DryRun:  dryrun,

		Pattern:      pattern,
		PatternRe:    re,
		Replacement:  replacement,
		Recursive:    recursive,
		IncludingDir: includingDir,
		OnlyDir:      onlyDir,
		MaxDepth:     maxDepth,
		IgnoreCase:   ignoreCase,
		IgnoreExt:    getFlagBool(cmd, "ignore-ext"),
		IgnoreErr:    getFlagBool(cmd, "ignore-err"),

		IncludeFilters:   infilters,
		IncludeFilterRes: infilterRes,
		SkipFilters:      skipfilters,
		SkipFilterRes:    skipRes,
		ExcludeFilters:   exfilters,
		ExcludeFilterRes: exfilterRes,

		ListPath:    onlyList,
		ListPathSep: getFlagString(cmd, "list-sep"),
		ListAbsPath: getFlagBool(cmd, "list-abs"),
		NatureSort:  getFlagBool(cmd, "nature-sort"),

		ReplaceWithNR: replaceWithNR,
		StartNum:      getFlagNonNegativeInt(cmd, "start-num"),
		NRFormat:      fmt.Sprintf("%%0%dd", getFlagPositiveInt(cmd, "nr-width")),
		ReplaceWithKV: replaceWithKV,

		KVs:         kvs,
		KVFile:      kvFile,
		KeepKey:     keepKey,
		KeyCaptIdx:  getFlagPositiveInt(cmd, "key-capt-idx"),
		KeyMissRepl: keyMissRepl,

		OverwriteMode: overwriteMode,

		PathCaseInsensitive: pathCaseInsensitive,

		Undo:             false,
		LastOpDetailFile: LastOpDetailFile,
		DisableUndo:      disableUndo,
	}
}

func init() {
	// logFormat := logging.MustStringFormatter(`%{color}[%{level:.4s}]%{color:reset} %{message}`)
	logFormat := logging.MustStringFormatter(`%{message}`)
	var stderr io.Writer = os.Stderr
	if runtime.GOOS == "windows" {
		stderr = colorable.NewColorableStderr()
	}
	backend := logging.NewLogBackend(stderr, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, logFormat)
	logging.SetBackend(backendFormatter)
	log = logging.MustGetLogger(app)

	RootCmd.Flags().BoolP("quiet", "q", false, "be quiet, do not show any information and warning")
	RootCmd.Flags().IntP("verbose", "v", 2, "verbose level (0 for all, 1 for warning, error and renamed files, 2 for only error and renamed files)")
	RootCmd.Flags().BoolP("version", "V", false, "print version information and check for update")
	RootCmd.Flags().BoolP("dry-run", "d", false, "preview rename, undo, or history removal without changing files")

	RootCmd.Flags().StringP("pattern", "p", "", "search pattern (regular expression)")
	RootCmd.Flags().StringP("replacement", "r", "", `replacement. capture variables supported.  e.g. $1 or ${1} (prefered) represents the first submatch. ATTENTION: for *nix OS, use SINGLE quote NOT double quotes or use the \ escape character. Ascending integer is also supported by "{nr}"`)
	RootCmd.Flags().BoolP("recursive", "R", false, "rename recursively")
	RootCmd.Flags().BoolP("including-dir", "D", false, "rename directories")
	RootCmd.Flags().BoolP("only-dir", "", false, "only rename directories")
	RootCmd.Flags().IntP("max-depth", "", 0, "maximum depth for recursive search or --clear (0 for no limit)")
	RootCmd.Flags().BoolP("ignore-case", "i", false, "ignore case of -p/--pattern, -f/--include-filters and -F/--exclude-filters")
	RootCmd.Flags().BoolP("ignore-ext", "e", false, "ignore file extension. i.e., replacement does not change file extension")
	RootCmd.Flags().BoolP("ignore-err", "E", false, "ignore directory reading errors")

	RootCmd.Flags().StringSliceP("include-filters", "f", []string{"."}, `include file filter(s) (regular expression, NOT wildcard). multiple values supported, e.g., -f ".html" -f ".htm", but ATTENTION: each comma in the filter is treated as the separator of multiple filters, please use double quotation marks for patterns containing comma, e.g., -p '"A{2,}"'`)
	RootCmd.Flags().StringSliceP("skip-filters", "S", []string{`^\.`}, `skip file filter(s) (regular expression, NOT wildcard). multiple values supported, e.g., -S "^\." for skipping files starting with a dot, but ATTENTION: each comma in the filter is treated as the separator of multiple filters, please use double quotation marks for patterns containing comma, e.g., -p '"A{2,}"'`)
	RootCmd.Flags().StringSliceP("exclude-filters", "F", []string{}, `exclude file filter(s) (regular expression, NOT wildcard). multiple values supported, e.g., -F ".html" -F ".htm", but ATTENTION: each comma in the filter is treated as the separator of multiple filters, please use double quotation marks for patterns containing comma, e.g., -p '"A{2,}"'`)

	RootCmd.Flags().BoolP("list", "l", false, `only list paths that match pattern`)
	RootCmd.Flags().StringP("list-sep", "s", "\n", `separator for list of found paths`)
	RootCmd.Flags().BoolP("list-abs", "a", false, `list absolute path, using along with -l/--list`)
	RootCmd.Flags().BoolP("nature-sort", "N", false, `sort paths in nature order for renaming or listing`)

	RootCmd.Flags().StringP("kv-file", "k", "",
		`tab-delimited key-value file for replacing key with value when using "{kv}" in -r (--replacement)`)
	RootCmd.Flags().BoolP("keep-key", "K", false, "keep the key as value when no value found for the key")
	RootCmd.Flags().IntP("key-capt-idx", "I", 1, "capture variable index of key (1-based)")
	RootCmd.Flags().StringP("key-miss-repl", "m", "", "replacement for key with no corresponding value")
	RootCmd.Flags().IntP("start-num", "n", 1, `starting number when using {nr} in replacement`)
	RootCmd.Flags().IntP("nr-width", "", 1, `minimum width for {nr} in flag -r/--replacement. e.g., formating "1" to "001" by --nr-width 3`)

	RootCmd.Flags().IntP("overwrite-mode", "o", 0, "overwrite mode (0 for reporting error, 1 for overwrite, 2 for not renaming) (default 0)")

	RootCmd.Flags().BoolP("case-insensitive-path", "w", false, "the file system (e.g., FAT32 or NTFS) is case-insensitive. It's automatically swiched on on Windows")
	RootCmd.Flags().BoolP("case-sensitive-path", "W", false, "believing that the file system is case-sensitive. Please use this to disable the flag -w/--case-insensitive-path, which is switched on by default on Windows")

	RootCmd.Flags().BoolP("undo", "u", false, "undo the last rename operation; retain failed entries for retry")
	RootCmd.Flags().BoolP("force-undo", "U", false, "continue undo after failures without overwriting existing paths")
	RootCmd.Flags().BoolP("disable-undo", "x", false, "do not create .brename_detail.txt file for undo")
	RootCmd.Flags().BoolP("clear", "", false, `remove .brename_detail.txt files; use -R/--recursive to include subdirectories`)

	RootCmd.Example = `  1. dry run and showing potential dangerous operations (-d)
      brename -p "abc" -d
  2. dry run and only show operations that will cause error (-v)
      brename -p "abc" -d -v 2
  3. only renaming specific paths via include filters (-f)
      brename -p ":" -r "-" -f ".htm$" -f ".html$"
  4. renaming all .jpeg files to .jpg in all subdirectories (-R)
      brename -p "\.jpeg" -r ".jpg" -R   dir
  5. using capture variables, e.g., $1, $2 ...
      brename -p "(a)" -r "\$1\$1"
      or brename -p "(a)" -r '$1$1' in Linux/Mac OS X
  6. renaming directory too (-D)
      brename -p ":" -r "-" -R -D   pdf-dirs
  7. using key-value file (-k)
      brename -p "(.+)" -r "{kv}" -k kv.tsv
  8. do not touch file extension (-e)
      brename -p ".+" -r "{nr}" -f .mkv -f .mp4 -e
  9. only list paths that match pattern (-l)
      brename -i -f '.docx?$' -p . -R -l
  10. undo the last rename operation; retain failed entries for retry (-u)
      brename -u
  11. disable undo if you do not want to create .brename_detail.txt (-x)
      brename -p xxx -r yyy -x
  12. clear/remove all .brename_detail.txt files (--clear)
      brename --clear -R
  13. also operate on hidden files: empty -S (default: ^\.)
      brename -p xxx -r yyy -S ""

  More examples: https://github.com/shenwei356/brename`

	RootCmd.SetUsageTemplate(`Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsagesWrapped 110 | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsagesWrapped 110 | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`)

	pathTree = make(map[string]struct{}, 1024)
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

// getFileList preserves explicit search roots and removes trailing separators.
func getFileList(args []string) []string {
	if len(args) == 0 {
		return []string{"."}
	}
	files := make([]string, 0, len(args))
	for _, path := range args {
		files = append(files, filepath.Clean(path))
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

func getFlagStringSlice(cmd *cobra.Command, flag string) []string {
	value, err := cmd.Flags().GetStringSlice(flag)
	checkError(err)
	return value
}

func getFlagPositiveInt(cmd *cobra.Command, flag string) int {
	value, err := cmd.Flags().GetInt(flag)
	checkError(err)
	if value <= 0 {
		checkError(fmt.Errorf("value of flag --%s should be greater than 0", flag))
	}
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
	fmt.Printf("%s v%s\n", app, VERSION)
	fmt.Println("\nChecking new version...")

	resp, err := http.Get(fmt.Sprintf("https://github.com/shenwei356/%s/releases/latest", app))
	if err != nil {
		checkError(fmt.Errorf("Network error"))
	}
	items := strings.Split(resp.Request.URL.String(), "/")
	var v string
	if items[len(items)-1] == "" {
		v = items[len(items)-2]
	} else {
		v = items[len(items)-1]
	}
	if v == "v"+VERSION {
		fmt.Printf("You are using the latest version of %s\n", app)
	} else {
		fmt.Printf("New version available: %s %s at %s\n", app, v, resp.Request.URL.String())
	}
}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   app,
	Short: "a cross-platform command-line tool for safely batch renaming files/directories via regular expression",
	Long: fmt.Sprintf(`
brename: a practical cross-platform command-line tool for safely batch renaming files/directories via regular expression

Version: %s

Author: Wei Shen <shenwei356@gmail.com>

Homepage: https://github.com/shenwei356/brename

Warnings:
  1. The path in file systems like FAT32 or NTFS is case-insensitive, so you should switch on the flag
     -w/--case-insensitive-path to correctly check file overwrites.
  2. The flag -w/--case-insensitive-path is switched on by default on Windows, please use
     -W/--case-sensitive-path to disable it if the file system is indeed case-sensitive.
  3. New paths ending with periods or spaces, being error-prone, are not allowed.
  4. Undo refuses to overwrite existing paths. Contents overwritten with -o 1 cannot be recovered.
  5. Renaming the working directory or its ancestors is not allowed.

Three path filters:

  1. -S/--skip-filters       black list     default value: ^\. (skipping paths starting with ".")
  2. -F/--exclude-filters    black list     no default value
  3. -f/--include-filters    white list     default value: .   (anything)

  Notes:
  1. Paths starting with "." are ignored by default, disable this with -S "".
  2. These options support multiple values, e.g., -f ".html" -f ".htm".
     But ATTENTION: each comma in filters is treated as a separator of multiple filters.
     Please use double quotation marks for patterns containing comma, e.g., -p '"A{2,}"'
  3. The three filters are performed in order of -S, -F, -f.
  4. -F/--exclude-filters is prefered for excluding path, cause it has no default value.
     Setting -S/--skip-filters will overwrite its default value.
  5. Skip and exclude filters also skip matching subdirectories during recursive search.
     Include filters select entries to rename without stopping traversal. Explicit search roots are searched.

Undo and history:
  1. Successful moves are recorded even if a later move fails. Undo failures remain in the log for retry.
  2. -U continues independent undo operations after failures and returns a nonzero status if any fail.
  3. -d previews renaming, undo, and --clear without changing files. Undo preview lists recorded operations.
  4. History stays in .brename_detail.txt. Existing logs remain readable, and ordinary filenames
     retain the old record format. Logs with filenames requiring escaping need an updated build.


Special replacement symbols:

  {nr}    Ascending integer
  {kv}    Corresponding value of the key (captured variable $n) by key-value file,
          n can be specified by flag -I/--key-capt-idx (default: 1)

Special cases of replacement string:
 *1. Capture variables should be in the format of '${1}' to reduce errors.
    a). If the capture variable is followed with space or other simple, it's OK:
            -r '$1 abc'
    b). If followed by numbers, characters, or underscore. That is ambiguous:
            -r '$1abc' actually refers to the variable '1abc', please use '${1}abc'.
            -r '$2_$1' actually refers to the variable '2_', please use '${2}_${1}'.
  2. Want to replace with a charactor '$',
    a). If using '{kv}', you need use '$$$$' instead of a single '$':
            -r '{kv}' -k <(sed 's/\$/$$$$/' kv.txt)
    b). If not, use '$$'. e.g., adding '$' to all numbers:
            -p '(\d+)' -d -r '$$${1}'

`, VERSION),
	Run: func(cmd *cobra.Command, args []string) {
		timeStart := time.Now()
		// var err error
		opt := getOptions(cmd)

		if opt.Version {
			return
		}

		// ------------------------------------------------
		// clear
		if opt.ClearOpDetailFiles {
			paths := getFileList(args)

			for _, path := range paths {
				checkError(clear(opt, path, 1))
			}

			return
		}

		// ------------------------------------------------
		// undo
		if opt.Undo {
			checkError(undoOperations(opt))
			return
		}

		// ------------------------------------------------
		// rename
		ops := make([]operation, 0, 1024)
		opCH := make(chan operation, 1024)
		done := make(chan int)

		var hasErr bool
		var n, nErr int
		var err error

		go func() {
			var outPath string
			var err error // Output errors belong to this goroutine.
			first := true
			verbose := !opt.Quiet || opt.DryRun
			for op := range opCH {
				if opt.ListPath {
					if opt.ListAbsPath {
						outPath, err = filepath.Abs(op.source)
						checkError(err)
					} else {
						outPath = op.source
					}
					if first {
						fmt.Print(outPath)
						first = false
					} else {
						fmt.Print(opt.ListPathSep + outPath)
					}
					continue
				}
				if int(op.code) >= opt.Verbose || opt.DryRun {
					switch op.code {
					case codeOK:
						if verbose {
							log.Infof("  %s\n", op)
						}
					case codeUnchanged:
						if verbose {
							log.Warningf("  %s\n", op)
						}
					case codeExisted, codeOverwriteNewPath:
						switch opt.OverwriteMode {
						case 0: // report error
							log.Errorf("  %s\n", op)
						case 1: // overwrite
							if verbose {
								log.Warningf("  %s (will be overwrited)\n", op)
							}
						case 2: // no renaming
							if verbose {
								log.Warningf("  %s (will NOT be overwrited)\n", op)
							}
						}
					case codeEndingWithPeriod, codeEndingWithSpace:
						if verbose {
							log.Errorf("  %s\n", op)
						}
					case codeMissingTarget:
						log.Errorf("  %s\n", op)
					}
				}

				switch op.code {
				case codeOK:
					ops = append(ops, op)
					n++
				case codeUnchanged:
				case codeExisted, codeOverwriteNewPath:
					switch opt.OverwriteMode {
					case 0: // report error
						hasErr = true
						nErr++
						continue
					case 1: // overwrite
						ops = append(ops, op)
						n++
					case 2: // no renaming

					}
				default:
					hasErr = true
					nErr++
					continue
				}
			}
			if opt.ListPath {
				fmt.Println()
			}
			done <- 1
		}()

		paths := getFileList(args)

		if !opt.Quiet && opt.Verbose == 0 {
			log.Info("------------------------------------------------------")
			log.Info()
			log.Infof("search paths: %s", strings.Join(paths, ", "))
			log.Info()
		}

		if (!opt.Quiet || opt.DryRun) && !opt.ListPath {
			log.Info(bold("Searching for paths to rename..."))
			log.Info()
		}
		for _, path := range paths {
			err = walk(opt, opCH, path, 1)
			if err != nil {
				close(opCH)
				checkError(err)
			}
		}
		if !opt.Quiet && !opt.DryRun && !opt.ListPath {
			fmt.Fprintf(os.Stderr, "\r  %-78s\n", green("Done searching."))
		}
		close(opCH)
		<-done

		if hasErr {
			log.Info()
			log.Errorf("%d potential error(s) detected, please check", nErr)
			os.Exit(1)
		}

		if opt.ListPath {
			return
		}
		if opt.DryRun || (!opt.Quiet && opt.Verbose == 0) || n == 0 {
			log.Info()
			log.Infof("%d path(s) to be renamed", n)
		}
		if n == 0 {
			return
		}

		if opt.DryRun {
			return
		}

		n2, err := executeOperations(opt, ops)
		checkError(err)

		if !opt.Quiet {
			log.Info()
			log.Infof("%d path(s) renamed in %.3f seconds", n2, time.Since(timeStart).Seconds())
		}
	},
}

type code int

const (
	codeOK code = iota
	codeUnchanged
	codeExisted
	codeOverwriteNewPath
	codeMissingTarget
	codeEndingWithSpace
	codeEndingWithPeriod
)

var yellow = color.New(color.FgYellow).SprintFunc()
var red = color.New(color.FgRed).SprintFunc()
var green = color.New(color.FgGreen).SprintFunc()
var bold = color.New(color.Bold).SprintFunc()

func (c code) String() string {
	switch c {
	case codeOK:
		return green("OK")
	case codeUnchanged:
		return yellow("unchanged")
	case codeExisted:
		return red("new path existed")
	case codeOverwriteNewPath:
		return red("overwriting newly renamed path")
	case codeMissingTarget:
		return red("missing target")
	case codeEndingWithSpace:
		return red("new path ending with a space")
	case codeEndingWithPeriod:
		return red("new path ending with a period")
	}

	return "undefined code"
}

type operation struct {
	source string
	target string
	code   code
}

func (op operation) String() string {
	return fmt.Sprintf(`[%s] %s -> %s`, op.code, op.source, op.target)
}

// checkOperation checks an renaming operation
func checkOperation(opt *Options, path string) (bool, operation) {
	path = filepath.Clean(path)
	dir, filename := filepath.Split(path)
	var ext string
	if opt.IgnoreExt {
		ext = filepath.Ext(path)
		filename = filename[0 : len(filename)-len(ext)]
	}

	if !opt.PatternRe.MatchString(filename) {
		return false, operation{}
	}

	r := opt.Replacement

	if opt.ReplaceWithNR {
		r = reNR.ReplaceAllString(r, fmt.Sprintf(opt.NRFormat, opt.StartNum))
		opt.StartNum++
	}

	if opt.ReplaceWithKV {
		founds := opt.PatternRe.FindAllStringSubmatch(filename, -1)
		if len(founds) > 0 {
			found := founds[0]
			if opt.KeyCaptIdx > len(found)-1 {
				checkError(fmt.Errorf("value of flag -I/--key-capt-idx overflows"))
			}
			k := found[opt.KeyCaptIdx]
			if opt.IgnoreCase {
				k = strings.ToLower(k)
			}
			if _, ok := opt.KVs[k]; ok {
				r = reKV.ReplaceAllString(r, opt.KVs[k])
			} else if opt.KeepKey {
				r = reKV.ReplaceAllString(r, found[opt.KeyCaptIdx])
			} else if opt.KeyMissRepl != "" {
				r = reKV.ReplaceAllString(r, opt.KeyMissRepl)
			} else {
				return false, operation{path, path, codeUnchanged}
			}
		}
	}

	filename2 := opt.PatternRe.ReplaceAllString(filename, r) + ext

	target := filepath.Join(dir, filename2)

	if filename2 == "" {
		return true, operation{path, target, codeMissingTarget}
	}

	if filename2[len(filename2)-1] == '.' {
		return true, operation{path, target, codeEndingWithPeriod}
	}

	if filename2[len(filename2)-1] == ' ' {
		return true, operation{path, target, codeEndingWithSpace}
	}

	if filename2 == filename+ext {
		return true, operation{path, target, codeUnchanged}
	}

	checkError(protectWorkingDirectory(path, target))
	exists, err := targetExists(path, target)
	checkError(err)
	if exists {
		return true, operation{path, target, codeExisted}
	}

	// Resolve directory aliases, but keep the final directory entry itself.
	target2, err := pathKey(target)
	checkError(err)
	if opt.PathCaseInsensitive {
		target2 = strings.ToLower(target2)
	}
	if pathTree == nil {
		pathTree = make(map[string]struct{}, 1024)
	}
	if _, ok := pathTree[target2]; ok {
		return true, operation{path, target, codeOverwriteNewPath}
	}
	pathTree[target2] = struct{}{}

	return true, operation{path, target, codeOK}
}

// excluded controls both path exclusion and recursive directory pruning.
func excluded(opt *Options, path string) bool {
	for _, filters := range [][]*regexp.Regexp{opt.SkipFilterRes, opt.ExcludeFilterRes} {
		for _, re := range filters {
			if re.MatchString(path) {
				return true
			}
		}
	}
	return false
}

// ignore applies include filters after the two exclusion filters.
func ignore(opt *Options, path string) bool {
	if excluded(opt, path) {
		return true
	}
	for _, re := range opt.IncludeFilterRes { // white list
		if re.MatchString(path) {
			return false
		}
	}
	return true
}

// clear removes history files without reading file contents or following links.
func clear(opt *Options, path string, depth int) error {
	if opt.MaxDepth > 0 && depth > opt.MaxDepth {
		return nil
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		if filepath.Base(path) != opt.LastOpDetailFile {
			return nil
		}
		if opt.DryRun {
			log.Infof("would remove: %s", path)
			return nil
		}
		if err := os.Remove(path); err != nil {
			return err
		}
		if !opt.Quiet {
			log.Infof("removed: %s", path)
		}
		return nil
	}
	files, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, file := range files {
		// Never treat a directory named like the history file as a log.
		if (!file.IsDir() && file.Name() == opt.LastOpDetailFile) || (file.IsDir() && opt.Recursive) {
			nextDepth := depth
			if file.IsDir() {
				nextDepth++
			}
			if err := clear(opt, filepath.Join(path, file.Name()), nextDepth); err != nil {
				return err
			}
		}
	}
	return nil
}

// walk recursively search file to rename
func walk(opt *Options, opCh chan<- operation, path string, depth int) error {
	path = filepath.Clean(path)
	if opt.MaxDepth > 0 && depth > opt.MaxDepth {
		return nil
	}
	if !opt.Quiet && !opt.DryRun && !opt.ListPath {
		_path := path
		n := len(_path)
		if n > 78 {
			_path = path[:40] + "..." + path[n-35:]
		}
		fmt.Fprintf(os.Stderr, "\r  %-78s", _path)
	}

	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	// Preserve explicit directory-link search roots. Links encountered inside a
	// directory remain file entries and are never followed by recursive search.
	if depth == 1 && info.Mode()&os.ModeSymlink != 0 {
		if target, err := os.Stat(path); err == nil && target.IsDir() {
			info = target
		}
	}
	// Directory entries can be renamed without opening their contents.
	if !info.IsDir() {
		if opt.OnlyDir || ignore(opt, filepath.Base(path)) {
			return nil
		}
		if ok, op := checkOperation(opt, path); ok {
			opCh <- op
		}
		return nil
	}

	// it's a directory
	files, err := os.ReadDir(path)
	if err != nil {
		if !opt.Quiet && !opt.DryRun && !opt.ListPath {
			fmt.Fprintf(os.Stderr, "\n")
		}
		if !opt.IgnoreErr {
			return fmt.Errorf("  %s when reading dir: %s, you can use -E/--ignore-err to skip this", red("error detected"), path)
		}
		log.Warningf("  %s when reading dir: %s", yellow("err ignored"), path)
		return nil
	}

	var filename string
	_files := make([]string, 0, len(files))
	_dirs := make([]string, 0, len(files))
	for _, file := range files {
		filename = file.Name()

		if filename == "." || filename == ".." { // TODO
			continue
		}

		if file.IsDir() {
			_dirs = append(_dirs, filename)
		} else {
			_files = append(_files, filename)
		}
	}

	if !opt.OnlyDir {
		if opt.NatureSort {
			natsort.Sort(_files)
		}
		for _, filename := range _files {
			if ignore(opt, filename) {
				continue
			}
			fileFullPath := filepath.Join(path, filename)
			if ok, op := checkOperation(opt, fileFullPath); ok {
				opCh <- op
			}
		}
	}

	// sub directory
	if opt.NatureSort {
		natsort.Sort(_dirs)
	}
	for _, filename := range _dirs {
		if excluded(opt, filename) {
			continue
		}

		fileFullPath := filepath.Join(path, filename)
		if opt.Recursive {
			err := walk(opt, opCh, fileFullPath, depth+1)
			if err != nil {
				return err
			}
		}
		// rename directories
		if (opt.OnlyDir || opt.IncludingDir) && !ignore(opt, filename) {
			if ok, op := checkOperation(opt, fileFullPath); ok {
				opCh <- op
			}
		}
	}

	if depth > 1 {
		return nil
	}

	// rename the given root directory
	if (opt.OnlyDir || opt.IncludingDir) && !ignore(opt, filepath.Base(path)) {
		if ok, op := checkOperation(opt, path); ok {
			opCh <- op
		}
	}

	return nil
}

// readKVs reads a tab-delimited file containing key-value pairs
func readKVs(file string, ignoreCase bool) (kvs map[string]string, err error) {
	kvs = make(map[string]string)
	reader, closeInput, err := openKVInput(file)
	if err == xopen.ErrNoContent {
		return kvs, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := closeInput(); err == nil {
			err = closeErr
		}
	}()
	for {
		line, readErr := reader.ReadString('\n')
		// A truncated or corrupt compressed stream must never supply a partial map.
		if readErr != nil && readErr != io.EOF {
			return nil, readErr
		}
		items := strings.SplitN(strings.TrimRight(line, "\r\n"), "\t", 3)
		if len(items) >= 2 {
			key := items[0]
			if ignoreCase {
				key = strings.ToLower(key)
			}
			kvs[key] = items[1]
		}
		if readErr == io.EOF {
			return kvs, nil
		}
	}
}

// openKVInput retains compressed files, stdin, URLs and command inputs. Gzip
// uses the standard reader because the old pgzip dependency accepts a missing
// trailer as EOF. Keep the existing Go version requirement for this fix.
func openKVInput(file string) (*bufio.Reader, func() error, error) {
	var raw io.Reader
	var closeSource func() error
	switch {
	case file == "-":
		raw = os.Stdin
		closeSource = func() error { return nil }
	case strings.HasPrefix(file, "|"):
		args := strings.Fields(file[1:])
		if len(args) == 0 {
			return nil, nil, fmt.Errorf("empty key-value input command")
		}
		cmd := exec.Command(args[0], args[1:]...)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil, nil, err
		}
		if err := cmd.Start(); err != nil {
			stdout.Close()
			return nil, nil, err
		}
		raw = stdout
		closeSource = func() error {
			closeErr := stdout.Close()
			if err := cmd.Wait(); err != nil {
				return err
			}
			return closeErr
		}
	default:
		var err error
		raw, err = xopen.XReader(file)
		if err != nil {
			return nil, nil, err
		}
		closeSource = raw.(io.Closer).Close
	}
	buffer := bufio.NewReader(raw)
	magic, _ := buffer.Peek(2)
	isGzip := len(magic) == 2 && magic[0] == 0x1f && magic[1] == 0x8b
	var decoded io.ReadCloser
	var err error
	if isGzip {
		decoded, err = gzip.NewReader(buffer)
	} else {
		decoded, err = xopen.Buf(buffer)
	}
	if err != nil {
		if closeErr := closeSource(); err == xopen.ErrNoContent && closeErr != nil {
			err = closeErr
		}
		return nil, nil, err
	}
	closeInput := func() error {
		decodeErr := decoded.Close()
		sourceErr := closeSource()
		if decodeErr != nil {
			return decodeErr
		}
		return sourceErr
	}
	reader := bufio.NewReader(decoded)
	// xopen handles a BOM itself; match that behavior for the standard gzip reader.
	if isGzip {
		if bom, _ := reader.Peek(3); string(bom) == "\xef\xbb\xbf" {
			reader.Discard(3)
		}
	}
	return reader, closeInput, nil
}
