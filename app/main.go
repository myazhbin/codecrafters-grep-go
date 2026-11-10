package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Usage: echo <input_text> | your_program.sh -E <pattern>
func main() {
	if len(os.Args) < 3 || os.Args[1] != "-E" {
		fmt.Fprintf(os.Stderr, "usage: mygrep -E <pattern>\n")
		os.Exit(2) // 1 means no lines were selected, >1 means error
	}

	pattern := os.Args[2]

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: read input text: %v\n", err)
		os.Exit(2)
	}

	out := bufio.NewWriter(os.Stdout)
	matched, err := grep(string(input), pattern, out)
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

// grep writes every line of input that matches pattern to out and reports
// whether at least one line matched.
func grep(input, pattern string, out io.Writer) (bool, error) {
	re, err := Compile(pattern)
	if err != nil {
		return false, fmt.Errorf("invalid pattern %q: %v", pattern, err)
	}

	lines := strings.Split(input, "\n")
	if len(lines) > 1 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1] // trailing newline, not an extra empty line
	}

	matched := false
	for _, line := range lines {
		if re.Match(line) {
			matched = true
			fmt.Fprintln(out, line)
		}
	}
	return matched, nil
}

// matchLine reports whether line matches pattern.
func matchLine(line []byte, pattern string) (bool, error) {
	re, err := Compile(pattern)
	if err != nil {
		return false, err
	}
	return re.Match(string(line)), nil
}
