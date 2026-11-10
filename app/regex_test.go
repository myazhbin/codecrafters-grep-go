package main

import (
	"bytes"
	"testing"
)

type matchCase struct {
	pattern string
	input   string
	want    bool
}

func runCases(t *testing.T, cases []matchCase) {
	t.Helper()
	for _, c := range cases {
		got, err := matchLine([]byte(c.input), c.pattern)
		if err != nil {
			t.Errorf("matchLine(%q, %q) unexpected error: %v", c.input, c.pattern, err)
			continue
		}
		if got != c.want {
			t.Errorf("matchLine(%q, %q) = %v, want %v", c.input, c.pattern, got, c.want)
		}
	}
}

func TestLiteral(t *testing.T) {
	runCases(t, []matchCase{
		{"d", "dog", true},
		{"f", "dog", false},
		{"og", "dog", true},
	})
}

func TestDigits(t *testing.T) {
	runCases(t, []matchCase{
		{`\d`, "3", true},
		{`\d`, "foo101", true},
		{`\d`, "apple123", true},
		{`\d`, "c", false},
		{`\d`, "Art", false},
		{`\d`, "", false},
	})
}

func TestWordCharacters(t *testing.T) {
	runCases(t, []matchCase{
		{`\w`, "foo101", true},
		{`\w`, "word", true},
		{`\w`, "_", true},
		{`\w`, "×#÷_%÷×", true},
		{`\w`, "$!?", false},
		{`\w`, "%÷×=+#", false},
	})
}

func TestPositiveCharacterGroups(t *testing.T) {
	runCases(t, []matchCase{
		{"[abc]", "apple", true},
		{"[abcd]", "a", true},
		{"[abc]", "dog", false},
		{"[efgh]", "[]", false},
		{"[a-f]", "xxdxx", true},
		{"[a-f]", "xyz", false},
		{`[\d_]`, "ab_", true},
	})
}

func TestNegativeCharacterGroups(t *testing.T) {
	runCases(t, []matchCase{
		{"[^abc]", "dog", true},
		{"[^abc]", "cab", false},
		{"[^xyz]", "apple", true},
		{"[^anb]", "banana", false},
		{"[^abc]", "", false},
	})
}

func TestCombiningCharacterClasses(t *testing.T) {
	runCases(t, []matchCase{
		{`\d apple`, "1 apple", true},
		{`\d apple`, "sally has 3 apples", true},
		{`\d apple`, "1 orange", false},
		{`\d\d\d apples`, "sally has 124 apples", true},
		{`\d\d\d apples`, "sally has 12 apples", false},
		{`\d \w\w\ws`, "sally has 3 dogs", true},
		{`\d \w\w\ws`, "sally has 4 dogs", true},
		{`\d \w\w\ws`, "sally has 1 dog", false},
		{`\d\\d\\d apples`, "sally has 12 apples", false},
	})
}

func TestStartAnchor(t *testing.T) {
	runCases(t, []matchCase{
		{"^log", "log", true},
		{"^log", "logs", true},
		{"^log", "slog", false},
		{`^\d\d`, "12ab", true},
		{`^\d\d`, "a12", false},
	})
}

func TestEndAnchor(t *testing.T) {
	runCases(t, []matchCase{
		{"dog$", "dog", true},
		{"dog$", "hotdog", true},
		{"dog$", "dogs", false},
		{"^dog$", "dog", true},
		{"^dog$", "dogdog", false},
		{"^$", "", true},
	})
}

func TestOneOrMore(t *testing.T) {
	runCases(t, []matchCase{
		{"a+", "apple", true},
		{"a+", "SaaS", true},
		{"a+", "dog", false},
		{"ca+ts", "cats", true},
		{"ca+ts", "caaats", true},
		{"ca+ts", "cts", false},
		{"ca+at", "caaat", true}, // needs backtracking
		{"ca+at", "cat", false},
		{`^\d+$`, "12345", true},
		{`^\d+$`, "123a5", false},
		{"[abc]+d", "xxabcabd", true},
	})
}

func TestZeroOrOne(t *testing.T) {
	runCases(t, []matchCase{
		{"dogs?", "dog", true},
		{"dogs?", "dogs", true},
		{"dogs?", "cat", false},
		{"ca?t", "ct", true},
		{"ca?t", "cat", true},
		{"ca?t", "caat", false},
		{"ca?at", "cat", true}, // needs backtracking
		{"^colou?r$", "color", true},
		{"^colou?r$", "colour", true},
		{"^colou?r$", "colouur", false},
	})
}

func TestZeroOrMoreAndCounts(t *testing.T) {
	runCases(t, []matchCase{
		{"ca*t", "ct", true},
		{"ca*t", "caaat", true},
		{"ca*t", "cbt", false},
		{"^a{3}$", "aaa", true},
		{"^a{3}$", "aa", false},
		{"^a{2,}$", "aaaa", true},
		{"^a{2,3}$", "aaaa", false},
		{"^(a*)*$", "aaab", false}, // must terminate
	})
}

func TestWildcard(t *testing.T) {
	runCases(t, []matchCase{
		{"d.g", "dog", true},
		{"d.g", "dig", true},
		{"d.g", "cog", false},
		{"d.g", "dg", false},
		{"c.t", "cat", true},
		{"c.+t", "ct", false},
		{"g.+gol", "goøö0Ogol", true},
		{`\.`, "abc", false},
		{`\.`, "a.c", true},
	})
}

func TestAlternation(t *testing.T) {
	runCases(t, []matchCase{
		{"(cat|dog)", "cat", true},
		{"(cat|dog)", "dog", true},
		{"(cat|dog)", "apple", false},
		{"a (cat|dog)$", "a dog", true},
		{"a (cat|dog)$", "a cow", false},
		{"^I see (\\d (cat|dog|cow)s?(, | and )?)+$", "I see 1 cat, 2 dogs and 3 cows", true},
		{"^I see (\\d (cat|dog|cow)s?(, | and )?)+$", "I see 1 cat, 2 dogs and 3 cow!", false},
		{"cat|dog", "hotdog", true},
		{"^(cat|dog)s$", "dogs", true},
		{"^(a|ab)c$", "abc", true}, // backtrack into the group
	})
}

func TestSingleBackreference(t *testing.T) {
	runCases(t, []matchCase{
		{`(cat) and \1`, "cat and cat", true},
		{`(cat) and \1`, "cat and dog", false},
		{`(\w+) and \1`, "cat and cat", true},
		{`(\w+) and \1`, "dog and dog", true},
		{`(\w+) and \1`, "cat and dog", false},
		{`(\w\w\w\w \d\d\d) is doing \1 times`, "grep 101 is doing grep 101 times", true},
		{`(\w\w\w \d\d\d) is doing \1 times`, "$?! 101 is doing $?! 101 times", false},
		{`(\w\w\w\w \d\d\d) is doing \1 times`, "grep yes is doing grep yes times", false},
		{`([abcd]+) is \1, not [^xyz]+`, "abcd is abcd, not efg", true},
		{`([abcd]+) is \1, not [^xyz]+`, "efgh is efgh, not efg", false},
		{`([abcd]+) is \1, not [^xyz]+`, "abcd is abcd, not xyz", false},
		{`^(\w+) starts and ends with \1$`, "this starts and ends with this", true},
		{`^(this) starts and ends with \1$`, "that starts and ends with this", false},
		{`^(this) starts and ends with \1$`, "this starts and ends with this?", false},
		{`once a (drea+mer), alwaysz? a \1`, "once a dreaaamer, always a dreaaamer", true},
		{`once a (drea+mer), alwaysz? a \1`, "once a dremer, always a dreaaamer", false},
		{`once a (drea+mer), alwaysz? a \1`, "once a dreaaamer, alwayszzz a dreaaamer", false},
		{`(b..s|c..e) here and \1 there`, "bugs here and bugs there", true},
		{`(b..s|c..e) here and \1 there`, "bugz here and bugs there", false},
		{`(a+)\1$`, "aaaa", true}, // group must give characters back
		{`^(a+)\1$`, "aaa", false},
	})
}

func TestMultipleBackreferences(t *testing.T) {
	runCases(t, []matchCase{
		{`(\d+) (\w+) squares and \1 \2 circles`, "3 red squares and 3 red circles", true},
		{`(\d+) (\w+) squares and \1 \2 circles`, "3 red squares and 4 red circles", false},
		{`(\w\w\w\w) (\d\d\d) is doing \1 \2 times`, "grep 101 is doing grep 101 times", true},
		{`(\w\w\w) (\d\d\d) is doing \1 \2 times`, "$?! 101 is doing $?! 101 times", false},
		{`(\w\w\w\w) (\d\d\d) is doing \1 \2 times`, "grep yes is doing grep yes times", false},
		{`([abc]+)-([def]+) is \1-\2, not [^xyz]+`, "abc-def is abc-def, not efg", true},
		{`([abc]+)-([def]+) is \1-\2, not [^xyz]+`, "efg-hij is efg-hij, not efg", false},
		{`([abc]+)-([def]+) is \1-\2, not [^xyz]+`, "abc-def is abc-def, not xyz", false},
		{`^(\w+) (\w+), \1 and \2$`, "apple pie, apple and pie", true},
		{`^(apple) (\w+), \1 and \2$`, "pineapple pie, pineapple and pie", false},
		{`^(\w+) (pie), \1 and \2$`, "apple pie, apple and pies", false},
		{`(how+dy) (he?y) there, \1 \2`, "howwdy hey there, howwdy hey", true},
		{`(how+dy) (he?y) there, \1 \2`, "hody hey there, howwdy hey", false},
		{`(how+dy) (he?y) there, \1 \2`, "howwdy heeey there, howwdy heeey", false},
		{`(c.t|d.g) and (f..h|b..d), \1 with \2`, "cat and fish, cat with fish", true},
		{`(c.t|d.g) and (f..h|b..d), \1 with \2`, "bat and fish, cat with fish", false},
		{`(a)(b)(c)(d)(e)(f)(g)(h)(i)(j)(k)\11\1`, "abcdefghijkka", true},
	})
}

func TestNestedBackreferences(t *testing.T) {
	runCases(t, []matchCase{
		{`('(cat) and \2') is the same as \1`, "'cat and cat' is the same as 'cat and cat'", true},
		{`('(cat) and \2') is the same as \1`, "'cat and cat' is the same as 'cat and dog'", false},
		{`((\w\w\w\w) (\d\d\d)) is doing \2 \3 times, and again \1 times`, "grep 101 is doing grep 101 times, and again grep 101 times", true},
		{`((\w\w\w) (\d\d\d)) is doing \2 \3 times, and again \1 times`, "$?! 101 is doing $?! 101 times, and again $?! 101 times", false},
		{`((\w\w\w\w) (\d\d\d)) is doing \2 \3 times, and again \1 times`, "grep yes is doing grep yes times, and again grep yes times", false},
		{`(([abc]+)-([def]+)) is \1, not ([^xyz]+), \2, or \3`, "abc-def is abc-def, not efg, abc, or def", true},
		{`(([abc]+)-([def]+)) is \1, not ([^xyz]+), \2, or \3`, "efg-hij is efg-hij, not klm, efg, or hij", false},
		{`(([abc]+)-([def]+)) is \1, not ([^xyz]+), \2, or \3`, "abc-def is abc-def, not xyz, abc, or def", false},
		{`^((\w+) (\w+)) is made of \2 and \3. love \1$`, "apple pie is made of apple and pie. love apple pie", true},
		{`^((apple) (\w+)) is made of \2 and \3. love \1$`, "pineapple pie is made of apple and pie. love apple pie", false},
		{`^((\w+) (pie)) is made of \2 and \3. love \1$`, "apple pie is made of apple and pie. love apple pies", false},
		{`'((how+dy) (he?y) there)' is made up of '\2' and '\3'. \1`, "'howwdy hey there' is made up of 'howwdy' and 'hey'. howwdy hey there", true},
		{`'((how+dy) (he?y) there)' is made up of '\2' and '\3'. \1`, "'hody hey there' is made up of 'hody' and 'hey'. hody hey there", false},
		{`'((how+dy) (he?y) there)' is made up of '\2' and '\3'. \1`, "'howwdy heeey there' is made up of 'howwdy' and 'heeey'. howwdy heeey there", false},
		{`((c.t|d.g) and (f..h|b..d)), \2 with \3, \1`, "cat and fish, cat with fish, cat and fish", true},
		{`((c.t|d.g) and (f..h|b..d)), \2 with \3, \1`, "bat and fish, bat with fish, bat and fish", false},
	})
}

func TestInvalidPatterns(t *testing.T) {
	for _, p := range []string{"(abc", "abc)", "[abc", `abc\`, "[z-a]"} {
		if _, err := Compile(p); err == nil {
			t.Errorf("Compile(%q) expected an error", p)
		}
	}
}

func TestGrepPrintsMatchingLines(t *testing.T) {
	var out bytes.Buffer
	matched, err := grep("apple1\nbanana\ncherry22\n", `\d`, &out)
	if err != nil {
		t.Fatal(err)
	}
	if !matched {
		t.Fatal("expected a match")
	}
	if got, want := out.String(), "apple1\ncherry22\n"; got != want {
		t.Errorf("output = %q, want %q", got, want)
	}

	out.Reset()
	matched, err = grep("banana", `\d`, &out)
	if err != nil {
		t.Fatal(err)
	}
	if matched || out.Len() != 0 {
		t.Errorf("expected no match and no output, got matched=%v output=%q", matched, out.String())
	}
}
