package main

import (
	"fmt"
	"strconv"
)

// A small backtracking regex engine written from scratch.
//
// Supported syntax:
//   literals, .            \d \w \s (and \D \W \S)
//   [abc] [^abc] [a-z]     ^ $
//   + ? * {n} {n,} {n,m}   (a|b) groups and alternation
//   \1 .. \N               backreferences (including nested groups)

type nodeKind int

const (
	kindChar    nodeKind = iota // a literal rune
	kindAny                     // .
	kindSet                     // \d, \w, [abc], [^abc]
	kindStart                   // ^
	kindEnd                     // $
	kindGroup                   // ( alt | alt ), capturing when index > 0
	kindBackref                 // \N
	kindRepeat                  // child{min,max}
)

type setItem struct {
	lo, hi rune              // inclusive range, used when class == nil
	class  func(r rune) bool // predefined class such as \d
}

type node struct {
	kind nodeKind

	ch rune // kindChar

	items  []setItem // kindSet
	negate bool      // kindSet

	alts  [][]*node // kindGroup: alternatives, each a sequence
	index int       // kindGroup: capture index (0 = non-capturing); kindBackref: group referenced

	child    *node // kindRepeat
	min, max int   // kindRepeat (max < 0 means unbounded)
}

func isDigit(r rune) bool { return r >= '0' && r <= '9' }

func isWord(r rune) bool {
	return isDigit(r) || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_'
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f' || r == '\v'
}

func (n *node) matchesRune(r rune) bool {
	switch n.kind {
	case kindChar:
		return n.ch == r
	case kindAny:
		return r != '\n'
	case kindSet:
		found := false
		for _, it := range n.items {
			if it.class != nil {
				if it.class(r) {
					found = true
					break
				}
			} else if r >= it.lo && r <= it.hi {
				found = true
				break
			}
		}
		return found != n.negate
	}
	return false
}

// ---------------------------------------------------------------------------
// Parser
// ---------------------------------------------------------------------------

type parser struct {
	src    []rune
	pos    int
	groups int // number of capturing groups seen so far
}

// Regex is a compiled pattern.
type Regex struct {
	root   *node
	groups int
}

// Compile parses pattern into a Regex.
func Compile(pattern string) (*Regex, error) {
	p := &parser{src: []rune(pattern)}
	alts, err := p.parseAlternatives()
	if err != nil {
		return nil, err
	}
	if p.pos < len(p.src) {
		return nil, fmt.Errorf("unmatched ')' at position %d", p.pos)
	}
	return &Regex{root: &node{kind: kindGroup, alts: alts}, groups: p.groups}, nil
}

func (p *parser) more() bool { return p.pos < len(p.src) }

func (p *parser) parseAlternatives() ([][]*node, error) {
	var alts [][]*node
	for {
		seq, err := p.parseSequence()
		if err != nil {
			return nil, err
		}
		alts = append(alts, seq)
		if p.more() && p.src[p.pos] == '|' {
			p.pos++
			continue
		}
		return alts, nil
	}
}

func (p *parser) parseSequence() ([]*node, error) {
	seq := []*node{}
	for p.more() && p.src[p.pos] != '|' && p.src[p.pos] != ')' {
		atom, err := p.parseAtom()
		if err != nil {
			return nil, err
		}
		atom, err = p.parseQuantifiers(atom)
		if err != nil {
			return nil, err
		}
		seq = append(seq, atom)
	}
	return seq, nil
}

func (p *parser) parseAtom() (*node, error) {
	c := p.src[p.pos]
	p.pos++
	switch c {
	case '.':
		return &node{kind: kindAny}, nil
	case '^':
		return &node{kind: kindStart}, nil
	case '$':
		return &node{kind: kindEnd}, nil
	case '(':
		capturing := true
		if p.pos+1 < len(p.src) && p.src[p.pos] == '?' && p.src[p.pos+1] == ':' {
			capturing = false
			p.pos += 2
		}
		index := 0
		if capturing {
			// Groups are numbered by the position of their opening parenthesis.
			p.groups++
			index = p.groups
		}
		alts, err := p.parseAlternatives()
		if err != nil {
			return nil, err
		}
		if !p.more() || p.src[p.pos] != ')' {
			return nil, fmt.Errorf("missing ')'")
		}
		p.pos++
		return &node{kind: kindGroup, alts: alts, index: index}, nil
	case '[':
		return p.parseSet()
	case '\\':
		return p.parseEscape()
	}
	return &node{kind: kindChar, ch: c}, nil
}

func classFor(c rune) (func(rune) bool, bool) {
	switch c {
	case 'd':
		return isDigit, true
	case 'w':
		return isWord, true
	case 's':
		return isSpace, true
	}
	return nil, false
}

func escapedLiteral(c rune) rune {
	switch c {
	case 'n':
		return '\n'
	case 't':
		return '\t'
	case 'r':
		return '\r'
	}
	return c
}

func (p *parser) parseEscape() (*node, error) {
	if !p.more() {
		return nil, fmt.Errorf("trailing backslash")
	}
	c := p.src[p.pos]
	p.pos++
	if class, ok := classFor(c); ok {
		return &node{kind: kindSet, items: []setItem{{class: class}}}, nil
	}
	switch c {
	case 'D', 'W', 'S':
		class, _ := classFor(c + ('a' - 'A'))
		return &node{kind: kindSet, items: []setItem{{class: class}}, negate: true}, nil
	}
	if c >= '1' && c <= '9' {
		start := p.pos - 1
		for p.more() && isDigit(p.src[p.pos]) {
			p.pos++
		}
		n, _ := strconv.Atoi(string(p.src[start:p.pos]))
		return &node{kind: kindBackref, index: n}, nil
	}
	return &node{kind: kindChar, ch: escapedLiteral(c)}, nil
}

func (p *parser) parseSet() (*node, error) {
	n := &node{kind: kindSet}
	if p.more() && p.src[p.pos] == '^' {
		n.negate = true
		p.pos++
	}
	first := true
	for {
		if !p.more() {
			return nil, fmt.Errorf("missing ']'")
		}
		c := p.src[p.pos]
		p.pos++
		if c == ']' && !first {
			return n, nil
		}
		first = false
		if c == '\\' {
			if !p.more() {
				return nil, fmt.Errorf("trailing backslash")
			}
			e := p.src[p.pos]
			p.pos++
			if class, ok := classFor(e); ok {
				n.items = append(n.items, setItem{class: class})
				continue
			}
			c = escapedLiteral(e)
		}
		// Range such as a-z ('-' right before ']' is a literal dash).
		if p.pos+1 < len(p.src) && p.src[p.pos] == '-' && p.src[p.pos+1] != ']' {
			hi := p.src[p.pos+1]
			p.pos += 2
			if hi == '\\' && p.more() {
				hi = escapedLiteral(p.src[p.pos])
				p.pos++
			}
			if hi < c {
				return nil, fmt.Errorf("invalid range %c-%c", c, hi)
			}
			n.items = append(n.items, setItem{lo: c, hi: hi})
			continue
		}
		n.items = append(n.items, setItem{lo: c, hi: c})
	}
}

func (p *parser) parseQuantifiers(atom *node) (*node, error) {
	for p.more() {
		min, max := 0, 0
		switch p.src[p.pos] {
		case '+':
			min, max = 1, -1
			p.pos++
		case '*':
			min, max = 0, -1
			p.pos++
		case '?':
			min, max = 0, 1
			p.pos++
		case '{':
			lo, hi, next, ok := p.parseBraces()
			if !ok {
				return atom, nil // treat '{' as a literal
			}
			min, max = lo, hi
			p.pos = next
		default:
			return atom, nil
		}
		if atom.kind == kindStart || atom.kind == kindEnd {
			return nil, fmt.Errorf("quantifier applied to an anchor")
		}
		atom = &node{kind: kindRepeat, child: atom, min: min, max: max}
	}
	return atom, nil
}

// parseBraces parses {n}, {n,} or {n,m} starting at p.pos without consuming it.
func (p *parser) parseBraces() (lo, hi, next int, ok bool) {
	i := p.pos + 1
	readInt := func() (int, bool) {
		start := i
		for i < len(p.src) && isDigit(p.src[i]) {
			i++
		}
		if i == start {
			return 0, false
		}
		v, err := strconv.Atoi(string(p.src[start:i]))
		return v, err == nil
	}
	lo, ok = readInt()
	if !ok {
		return 0, 0, 0, false
	}
	hi = lo
	if i < len(p.src) && p.src[i] == ',' {
		i++
		if i < len(p.src) && p.src[i] == '}' {
			hi = -1
		} else if hi, ok = readInt(); !ok || hi < lo {
			return 0, 0, 0, false
		}
	}
	if i >= len(p.src) || p.src[i] != '}' {
		return 0, 0, 0, false
	}
	return lo, hi, i + 1, true
}

// ---------------------------------------------------------------------------
// Matcher
// ---------------------------------------------------------------------------

type matcher struct {
	text []rune
	caps [][2]int // caps[i] = [start, end) of group i, start < 0 when unset
}

// matchSeq matches seq[i:] at pos and then calls k with the end position.
func (m *matcher) matchSeq(seq []*node, i, pos int, k func(int) bool) bool {
	if i == len(seq) {
		return k(pos)
	}
	return m.matchNode(seq[i], pos, func(end int) bool {
		return m.matchSeq(seq, i+1, end, k)
	})
}

func (m *matcher) matchNode(n *node, pos int, k func(int) bool) bool {
	switch n.kind {
	case kindChar, kindAny, kindSet:
		return pos < len(m.text) && n.matchesRune(m.text[pos]) && k(pos+1)

	case kindStart:
		return pos == 0 && k(pos)

	case kindEnd:
		return pos == len(m.text) && k(pos)

	case kindGroup:
		for _, alt := range n.alts {
			ok := m.matchSeq(alt, 0, pos, func(end int) bool {
				if n.index == 0 {
					return k(end)
				}
				saved := m.caps[n.index]
				m.caps[n.index] = [2]int{pos, end}
				if k(end) {
					return true
				}
				m.caps[n.index] = saved // backtrack
				return false
			})
			if ok {
				return true
			}
		}
		return false

	case kindBackref:
		if n.index >= len(m.caps) || m.caps[n.index][0] < 0 {
			return false
		}
		captured := m.text[m.caps[n.index][0]:m.caps[n.index][1]]
		if pos+len(captured) > len(m.text) {
			return false
		}
		for j, r := range captured {
			if m.text[pos+j] != r {
				return false
			}
		}
		return k(pos + len(captured))

	case kindRepeat:
		return m.matchRepeat(n, 0, pos, k)
	}
	return false
}

// matchRepeat greedily matches n.child as many times as allowed, backtracking
// to fewer repetitions when the rest of the pattern fails.
func (m *matcher) matchRepeat(n *node, count, pos int, k func(int) bool) bool {
	if n.max < 0 || count < n.max {
		ok := m.matchNode(n.child, pos, func(end int) bool {
			if end == pos && count >= n.min {
				return false // empty iteration: avoid looping forever
			}
			return m.matchRepeat(n, count+1, end, k)
		})
		if ok {
			return true
		}
	}
	return count >= n.min && k(pos)
}

// Find returns the [start, end) rune offsets of the leftmost match in text.
func (re *Regex) Find(text []rune) (start, end int, ok bool) {
	m := &matcher{text: text, caps: make([][2]int, re.groups+1)}
	for start = 0; start <= len(text); start++ {
		for i := range m.caps {
			m.caps[i] = [2]int{-1, -1}
		}
		if m.matchNode(re.root, start, func(e int) bool { end = e; return true }) {
			return start, end, true
		}
	}
	return 0, 0, false
}

// Match reports whether the pattern matches anywhere in text.
func (re *Regex) Match(text string) bool {
	_, _, ok := re.Find([]rune(text))
	return ok
}
