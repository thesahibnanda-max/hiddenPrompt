package utils

import "strings"

// hintStopwords strips true function words - articles, prepositions,
// basic pronouns, auxiliary/linking verbs, demonstratives, and a couple of
// generic fillers ("type", "kind", "sort") - but deliberately KEEPS
// frequency/comparison/superlative words ("usually", "often", "always",
// "best", "most", "largest", "specific", "main", "typical", ...). Those
// are exactly the words that carry the distinguishing framing in this
// game's questions (e.g. "usually" is the one word that separates "what
// do you usually eat for breakfast" from "what's the best breakfast
// meal") - a generic/aggressive stopword list would strip the single most
// useful signal MissingContentWords exists to surface.
var hintStopwords = map[string]bool{
	"a": true, "an": true, "the": true,
	"is": true, "are": true, "was": true, "were": true, "be": true, "been": true, "being": true,
	"do": true, "does": true, "did": true, "done": true,
	"have": true, "has": true, "had": true,
	"can": true, "could": true, "will": true, "would": true, "shall": true, "should": true, "may": true, "might": true, "must": true,
	"i": true, "me": true, "my": true, "mine": true,
	"you": true, "your": true, "yours": true,
	"it": true, "its": true, "they": true, "them": true, "their": true,
	"we": true, "us": true, "our": true,
	"he": true, "him": true, "his": true, "she": true, "her": true, "hers": true,
	"this": true, "that": true, "these": true, "those": true,
	"what": true, "which": true, "who": true, "whom": true, "whose": true,
	"of": true, "in": true, "on": true, "at": true, "to": true, "for": true, "with": true, "from": true, "by": true, "about": true, "as": true, "into": true, "onto": true,
	"and": true, "or": true, "but": true, "if": true, "so": true,
	"type": true, "kind": true, "sort": true,
}

// MissingContentWords returns up to max words present in target's
// normalized token set but absent from given's - a cheap, deterministic
// proxy for "what specific concept does this guess not address," used to
// ground LLM hint generation without spending any extra tokens/calls on a
// separate reasoning pass. Order follows target's own word order; each
// returned word appears at most once.
func MissingContentWords(target, given string, max int) []string {
	givenWords := contentWords(given)
	givenSet := make(map[string]bool, len(givenWords))
	for _, w := range givenWords {
		givenSet[w] = true
	}

	var missing []string
	seen := make(map[string]bool)
	for _, w := range contentWords(target) {
		if givenSet[w] || seen[w] {
			continue
		}
		seen[w] = true
		missing = append(missing, w)
		if len(missing) >= max {
			break
		}
	}
	return missing
}

func contentWords(s string) []string {
	fields := strings.Fields(NormalizeString(s))
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if hintStopwords[f] {
			continue
		}
		out = append(out, f)
	}
	return out
}
