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

Titles are filled in / corrected from `codecrafters task` as each stage is reached.

- [ ] Task 2 — Match word characters (`\w`)
- [ ] Task 3 — Positive character groups (`[abc]`)
- [ ] Task 4 — Negative character groups (`[^abc]`)
- [ ] Task 5 — Combining character classes (`\d apple`, `\d\d\d apples`)
- [ ] Task 6 — Start of string anchor (`^`)
- [ ] Task 7 — End of string anchor (`$`)
- [ ] Task 8 — Match one or more times (`+`)
- [ ] Task 9 — Match zero or one times (`?`)
- [ ] Task 10 — Wildcard (`.`)
- [ ] Task 11 — Alternation (`(cat|dog)`)
- [ ] Task 12 — Single backreference (`\1`)
- [ ] Task 13 — Multiple backreferences (`\1 ... \2`)
- [ ] Task 14 — Nested backreferences (final task)
