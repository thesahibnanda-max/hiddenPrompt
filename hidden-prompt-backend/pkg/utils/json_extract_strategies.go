package utils

import (
	"regexp"
	"strings"
)

// jsonCandidateFinder extracts one or more JSON-object candidate substrings
// from a raw LLM response, in the order they were found. ExtractJSON runs a
// sequence of finders and tries every candidate each one returns before
// falling through to the next, less-confident finder.
type jsonCandidateFinder interface {
	find(raw string) []string
}

var defaultJSONCandidateFinders = []jsonCandidateFinder{
	directJSONFinder{},
	fencedCodeBlockFinder{},
	braceMatchFinder{},
}

// directJSONFinder tries the raw response as-is - the cheapest and most
// common case when the model actually followed instructions.
type directJSONFinder struct{}

func (directJSONFinder) find(raw string) []string {
	return []string{raw}
}

var (
	tripleBacktickJSONFenceRE = regexp.MustCompile("(?is)```json\\s*\\n?(.*?)\\n?\\s*```")
	tripleBacktickFenceRE     = regexp.MustCompile("(?s)```\\s*\\n?(.*?)\\n?\\s*```")
	singleBacktickFenceRE     = regexp.MustCompile("(?s)`([^`]*?)`")
)

// fencedCodeBlockFinder looks for JSON wrapped in a markdown code fence.
// It tries, in order, every ```json fence, then every plain ``` fence, then
// every single-backtick `fence` - stopping at the first tier that finds any
// blocks at all, so a real triple-backtick block is never diluted by a
// spurious single-backtick match inside it (three consecutive backticks can
// otherwise look like an empty single-backtick pair to a naive regex).
type fencedCodeBlockFinder struct{}

func (f fencedCodeBlockFinder) find(raw string) []string {
	if candidates := f.findAllFenced(tripleBacktickJSONFenceRE, raw); len(candidates) > 0 {
		return candidates
	}
	if candidates := f.findAllFenced(tripleBacktickFenceRE, raw); len(candidates) > 0 {
		return candidates
	}
	return f.findAllFenced(singleBacktickFenceRE, raw)
}

func (fencedCodeBlockFinder) findAllFenced(re *regexp.Regexp, raw string) []string {
	matches := re.FindAllStringSubmatch(raw, -1)
	candidates := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		candidate := strings.TrimSpace(m[1])
		if candidate == "" {
			continue
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

// braceMatchFinder is the last-resort strategy: it scans the entire string
// for every top-level {...} object, tracking string-literal/escape state so
// braces inside quoted strings are ignored, and keeps scanning after each
// closed object instead of stopping at the first one. This is what lets
// ExtractJSON recover when a model prefixes the real JSON with prose that
// itself happens to contain a decoy '{' (e.g. "the format looks like
// {example}") - the decoy is tried and rejected, and the scan continues to
// the real object further down instead of giving up.
type braceMatchFinder struct{}

func (braceMatchFinder) find(raw string) []string {
	var candidates []string

	scanner := braceScanner{start: -1}
	for i := 0; i < len(raw); i++ {
		if candidate, closed := scanner.step(raw, i); closed {
			candidates = append(candidates, candidate)
		}
	}

	return candidates
}

// braceScanner tracks brace depth and string-literal/escape state one byte
// at a time, so braceMatchFinder can find every top-level {...} object in a
// string while correctly ignoring braces that appear inside quoted string
// values (respecting \" escapes).
type braceScanner struct {
	depth    int
	start    int
	inString bool
	escaped  bool
}

// step consumes raw[i] and reports a candidate whenever it just closed a
// top-level JSON object.
func (b *braceScanner) step(raw string, i int) (candidate string, closed bool) {
	c := raw[i]

	if b.inString {
		b.stepInString(c)
		return "", false
	}

	switch c {
	case '"':
		b.inString = true
	case '{':
		if b.depth == 0 {
			b.start = i
		}
		b.depth++
	case '}':
		return b.closeBrace(raw, i)
	}

	return "", false
}

func (b *braceScanner) stepInString(c byte) {
	switch {
	case b.escaped:
		b.escaped = false
	case c == '\\':
		b.escaped = true
	case c == '"':
		b.inString = false
	}
}

func (b *braceScanner) closeBrace(raw string, i int) (candidate string, closed bool) {
	if b.depth == 0 {
		return "", false
	}

	b.depth--
	if b.depth != 0 || b.start < 0 {
		return "", false
	}

	candidate = raw[b.start : i+1]
	b.start = -1
	return candidate, true
}
