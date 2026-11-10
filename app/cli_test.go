package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// runCLI parses args and runs the search with the given stdin.
func runCLI(t *testing.T, stdin string, tty bool, args ...string) (string, bool) {
	t.Helper()
	opts, err := parseArgs(args)
	if err != nil {
		t.Fatalf("parseArgs(%q): %v", args, err)
	}
	var out bytes.Buffer
	matched, err := run(opts, strings.NewReader(stdin), &out, tty)
	if err != nil {
		t.Fatalf("run(%q): %v", args, err)
	}
	return out.String(), matched
}

func expect(t *testing.T, stdin string, args []string, wantOut string, wantMatch bool) {
	t.Helper()
	got, matched := runCLI(t, stdin, false, args...)
	if got != wantOut || matched != wantMatch {
		t.Errorf("args %q stdin %q:\n got  (%v) %q\n want (%v) %q", args, stdin, matched, got, wantMatch, wantOut)
	}
}

func TestParseArgs(t *testing.T) {
	opts, err := parseArgs([]string{"-o", "--color=always", "-r", "-E", `\d`, "a.txt", "b.txt"})
	if err != nil {
		t.Fatal(err)
	}
	want := options{pattern: `\d`, files: []string{"a.txt", "b.txt"}, onlyMatching: true, recursive: true, color: "always"}
	if !reflect.DeepEqual(opts, want) {
		t.Errorf("got %+v, want %+v", opts, want)
	}
	for _, bad := range [][]string{{}, {"-E"}, {"--color=sometimes", "-E", "a"}, {"-z", "-E", "a"}} {
		if _, err := parseArgs(bad); err == nil {
			t.Errorf("parseArgs(%q) expected an error", bad)
		}
	}
}

func TestPrintMatchingLines(t *testing.T) {
	expect(t, "banana", []string{"-E", "na"}, "banana\n", true)
	expect(t, "banana", []string{"-E", "x"}, "", false)
	expect(t, "apple1\nbanana\ncherry22\n", []string{"-E", `\d`}, "apple1\ncherry22\n", true)
}

func TestOnlyMatching(t *testing.T) {
	expect(t, "The king had 7 daughters", []string{"-o", "-E", `\d`}, "7\n", true)
	expect(t, "I have one cow", []string{"-o", "-E", "cow"}, "cow\n", true)
	expect(t, "I have one cow", []string{"-o", "-E", "dog"}, "", false)
	expect(t, "jekyll and hyde", []string{"-o", "-E", "(jekyll|hyde)"}, "jekyll\nhyde\n", true)
	expect(t, "a1b22c333", []string{"-o", "-E", `\d+`}, "1\n22\n333\n", true)
	expect(t, "cat 1\ndog\nbird 23 4\n", []string{"-o", "-E", `\d+`}, "1\n23\n4\n", true)
	expect(t, "abc", []string{"-o", "-E", "x*"}, "", true) // only empty matches: nothing printed
}

func TestHighlighting(t *testing.T) {
	red := func(s string) string { return "\x1b[01;31m\x1b[K" + s + "\x1b[m\x1b[K" }
	expect(t, "I have 3 cows", []string{"--color=always", "-E", `\d`}, "I have "+red("3")+" cows\n", true)
	expect(t, "a1b22c", []string{"--color=always", "-E", `\d+`}, "a"+red("1")+"b"+red("22")+"c\n", true)
	expect(t, "x1\nnone\ny2z\n", []string{"--color=always", "-E", `\d`}, "x"+red("1")+"\ny"+red("2")+"z\n", true)
	expect(t, "I have 3 cows", []string{"--color=always", "-E", "x"}, "", false)
	expect(t, "I have 3 cows", []string{"--color=never", "-E", `\d`}, "I have 3 cows\n", true)
	expect(t, "a1b2", []string{"--color=always", "-o", "-E", `\d`}, red("1")+"\n"+red("2")+"\n", true)

	// auto: colour only when stdout is a terminal.
	expect(t, "I have 3 cows", []string{"--color=auto", "-E", `\d`}, "I have 3 cows\n", true)
	got, _ := runCLI(t, "I have 3 cows", true, "--color=auto", "-E", `\d`)
	if want := "I have " + red("3") + " cows\n"; got != want {
		t.Errorf("auto on a terminal: got %q, want %q", got, want)
	}
	got, _ = runCLI(t, "I have 3 cows", true, "-E", `\d`)
	if want := "I have 3 cows\n"; got != want {
		t.Errorf("no --color on a terminal: got %q, want %q", got, want)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFileSearch(t *testing.T) {
	dir := t.TempDir()
	fruits := filepath.Join(dir, "fruits.txt")
	veg := filepath.Join(dir, "vegetables.txt")
	writeFile(t, fruits, "apple\nbanana\nblueberry\n")
	writeFile(t, veg, "broccoli\ncarrot")

	// single file: no filename prefix
	expect(t, "", []string{"-E", "appl.*", fruits}, "apple\n", true)
	expect(t, "", []string{"-E", "b.*y$", fruits}, "blueberry\n", true)
	expect(t, "", []string{"-E", "missing", fruits}, "", false)
	expect(t, "", []string{"-E", "^b", fruits}, "banana\nblueberry\n", true)

	// multiple files: filename prefix
	expect(t, "", []string{"-E", "b.*", fruits, veg},
		fruits+":banana\n"+fruits+":blueberry\n"+veg+":broccoli\n", true)
	expect(t, "", []string{"-E", "carrot", fruits, veg}, veg+":carrot\n", true)
	expect(t, "", []string{"-E", "zzz", fruits, veg}, "", false)
}

func TestRecursiveSearch(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "fruits.txt"), "pear\nstrawberry\n")
	writeFile(t, filepath.Join(dir, "subdir", "vegetables.txt"), "celery\ncarrot\n")
	writeFile(t, filepath.Join(dir, "subdir", "deep", "dairy.txt"), "cheese\nyogurt\n")

	got, matched := runCLI(t, "", false, "-r", "-E", ".*er", dir)
	want := filepath.Join(dir, "fruits.txt") + ":strawberry\n" +
		filepath.Join(dir, "subdir", "vegetables.txt") + ":celery\n"
	if !matched || got != want {
		t.Errorf("recursive: got (%v) %q, want %q", matched, got, want)
	}

	// trailing slash on the directory is kept as typed
	got, _ = runCLI(t, "", false, "-r", "-E", "yogurt", dir+"/")
	if want := dir + "/subdir/deep/dairy.txt:yogurt\n"; got != want {
		t.Errorf("recursive with trailing slash: got %q, want %q", got, want)
	}

	expect(t, "", []string{"-r", "-E", "missing_fruit", dir}, "", false)
}
