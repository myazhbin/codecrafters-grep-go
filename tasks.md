# Tasks — CodeCrafters "Build your own grep" (Go)

Workflow for every task:

1. Read the stage instructions (`codecrafters task`).
2. Implement it in `app/` (no use of Go's `regexp` package).
3. Add/extend Go tests in `app/*_test.go` and run `go test ./app/`.
4. Run `codecrafters submit`.
5. Once the remote tests pass, move to the next stage.

Status: `[ ]` todo, `[x]` implemented, Go tests pass, `codecrafters submit` passed.

## Task 1 — Match digits (`oq2`)

https://app.codecrafters.io/courses/grep/stages/oq2?repo=5b165d03-8a78-495a-ad26-a44a86829e57

- [x] Add support for the digit (`\d`) character class.
  - `\d` matches any single digit character (0-9).
  - `\d` should match `3` and `foo101`; should not match `c` or `Art`.
  - Tester runs: `echo -n "apple123" | ./your_program.sh -E "\d"`
  - Exit with code 0 if a digit is found in the input, 1 otherwise.
  - Build the regex implementation from scratch.

## Following tasks

Stage list taken from the course sidebar; details are filled in from
`codecrafters task` as each stage is reached.

### Base stages

- [x] Task 2 — Match word characters
- [x] Task 3 — Positive Character Groups
- [x] Task 4 — Negative Character Groups
- [x] Task 5 — Combining Character Classes
- [x] Task 6 — Start of string anchor
- [x] Task 7 — End of string anchor
- [x] Task 8 — Match one or more times
- [x] Task 9 — Match zero or one times
- [x] Task 10 — Wildcard
- [x] Task 11 — Alternation

### Printing Matches

- [x] Task 12 — Print a single matching line
- [x] Task 13 — Print multiple matching lines

### Multiple Matches

- [x] Task 14 — Print single match
- [x] Task 15 — Print multiple matches
- [x] Task 16 — Process multiple input lines

### Highlighting

- [x] Task 17 — Highlight a single match
- [x] Task 18 — Highlight multiple matches
- [x] Task 19 — Highlight matches in multiple lines
- [x] Task 20 — Disable highlighting
- [x] Task 21 — Implement the auto color option

### File Search

- [x] Task 22 — Search a single-line file
- [x] Task 23 — Search a multi-line file
- [x] Task 24 — Search multiple files
- [x] Task 25 — Recursive search

### Quantifiers

- [x] Task 26 — Match zero or more times
- [x] Task 27 — Match exactly n times
- [x] Task 28 — Match at least n times
- [x] Task 29 — Match between n and m times

### Backreferences

- [x] Task 30 — Single Backreference
- [x] Task 31 — Multiple Backreferences
- [x] Task 32 — Nested Backreferences (final task)
