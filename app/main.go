package main

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ANSI sequences GNU grep uses to highlight matched text (bold red).
const (
	colorStart = "\x1b[01;31m\x1b[K"
	colorEnd   = "\x1b[m\x1b[K"
)

// options holds the parsed command line.
type options struct {
	pattern      string
	files        []string
	onlyMatching bool   // -o
	recursive    bool   // -r
	color        string // "always", "never" or "auto"
}

// Usage: echo <input_text> | your_program.sh [-o] [-r] [--color=WHEN] -E <pattern> [file...]
func main() {
	opts, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "usage: mygrep [-o] [-r] [--color=WHEN] -E <pattern> [file...]\nerror: %v\n", err)
		os.Exit(2) // 1 means no lines were selected, >1 means error
	}

	out := bufio.NewWriter(os.Stdout)
	matched, err := run(opts, os.Stdin, out, isTerminal(os.Stdout))
	out.Flush()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	if !matched {
		os.Exit(1)
	}

	// default exit code is 0 which means success
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func parseArgs(args []string) (options, error) {
	opts := options{color: "never"}
	havePattern := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-E" || arg == "-e":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("%s requires a pattern", arg)
			}
			i++
			opts.pattern, havePattern = args[i], true
		case arg == "--color" || arg == "--colour":
			opts.color = "auto"
		case strings.HasPrefix(arg, "--color=") || strings.HasPrefix(arg, "--colour="):
			when := arg[strings.Index(arg, "=")+1:]
			switch when {
			case "always", "never", "auto":
				opts.color = when
			default:
				return opts, fmt.Errorf("invalid --color value %q", when)
			}
		case arg == "--":
			opts.files = append(opts.files, args[i+1:]...)
			i = len(args)
		case len(arg) > 1 && arg[0] == '-':
			// Possibly combined short flags such as -or.
			for _, f := range arg[1:] {
				switch f {
				case 'o':
					opts.onlyMatching = true
				case 'r', 'R':
					opts.recursive = true
				case 'E':
					// extended syntax is the only syntax we support
				default:
					return opts, fmt.Errorf("unknown option %q", arg)
				}
			}
		default:
			opts.files = append(opts.files, arg)
		}
	}
	if !havePattern {
		if len(opts.files) == 0 {
			return opts, fmt.Errorf("missing pattern")
		}
		opts.pattern, opts.files = opts.files[0], opts.files[1:]
	}
	return opts, nil
}

// run executes the search described by opts, reading stdin when no files are
// given, and reports whether anything matched.
func run(opts options, stdin io.Reader, out io.Writer, stdoutIsTerminal bool) (bool, error) {
	re, err := Compile(opts.pattern)
	if err != nil {
		return false, fmt.Errorf("invalid pattern %q: %v", opts.pattern, err)
	}
	p := &printer{
		re:           re,
		out:          out,
		onlyMatching: opts.onlyMatching,
		color:        opts.color == "always" || (opts.color == "auto" && stdoutIsTerminal),
	}

	if len(opts.files) == 0 {
		if opts.recursive {
			opts.files = []string{"."}
		} else {
			input, err := io.ReadAll(stdin)
			if err != nil {
				return false, fmt.Errorf("read input text: %v", err)
			}
			return p.search(string(input), ""), nil
		}
	}

	// Collect the files to search and decide whether to prefix lines with names.
	var paths []string
	var firstErr error
	withNames := len(opts.files) > 1
	for _, name := range opts.files {
		info, err := os.Stat(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "mygrep: %s: %v\n", name, err)
			firstErr = err
			continue
		}
		if !info.IsDir() {
			paths = append(paths, name)
			continue
		}
		if !opts.recursive {
			fmt.Fprintf(os.Stderr, "mygrep: %s: Is a directory\n", name)
			continue
		}
		withNames = true
		err = filepath.WalkDir(name, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.Type().IsRegular() {
				if strings.HasSuffix(name, "/") && !strings.HasPrefix(path, name) {
					// WalkDir cleans the root; keep the path as the user typed it.
					path = name + strings.TrimPrefix(path, filepath.Clean(name)+"/")
				}
				paths = append(paths, path)
			}
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "mygrep: %s: %v\n", name, err)
			firstErr = err
		}
	}

	matched := false
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "mygrep: %s: %v\n", path, err)
			firstErr = err
			continue
		}
		prefix := ""
		if withNames {
			prefix = path + ":"
		}
		if p.search(string(data), prefix) {
			matched = true
		}
	}
	if !matched && firstErr != nil {
		return false, firstErr
	}
	return matched, nil
}

// printer writes matching lines (or matches) in grep's output format.
type printer struct {
	re           *Regex
	out          io.Writer
	onlyMatching bool
	color        bool
}

// search prints the matches found in input, each line prefixed with prefix,
// and reports whether any line matched.
func (p *printer) search(input, prefix string) bool {
	lines := strings.Split(input, "\n")
	if len(lines) > 1 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1] // trailing newline, not an extra empty line
	}

	matched := false
	for _, line := range lines {
		text := []rune(line)
		if _, _, ok := p.re.Find(text); !ok {
			continue
		}
		matched = true

		if !p.onlyMatching && !p.color {
			fmt.Fprintf(p.out, "%s%s\n", prefix, line)
			continue
		}

		matches := p.re.FindAll(text)
		if p.onlyMatching {
			for _, m := range matches {
				fmt.Fprintf(p.out, "%s%s\n", prefix, p.paint(string(text[m[0]:m[1]])))
			}
			continue
		}

		var b strings.Builder
		last := 0
		for _, m := range matches {
			b.WriteString(string(text[last:m[0]]))
			b.WriteString(p.paint(string(text[m[0]:m[1]])))
			last = m[1]
		}
		b.WriteString(string(text[last:]))
		fmt.Fprintf(p.out, "%s%s\n", prefix, b.String())
	}
	return matched
}

func (p *printer) paint(s string) string {
	if !p.color {
		return s
	}
	return colorStart + s + colorEnd
}

// grep writes every line of input that matches pattern to out and reports
// whether at least one line matched.
func grep(input, pattern string, out io.Writer) (bool, error) {
	return run(options{pattern: pattern, color: "never"}, strings.NewReader(input), out, false)
}

// matchLine reports whether line matches pattern.
func matchLine(line []byte, pattern string) (bool, error) {
	re, err := Compile(pattern)
	if err != nil {
		return false, err
	}
	return re.Match(string(line)), nil
}
