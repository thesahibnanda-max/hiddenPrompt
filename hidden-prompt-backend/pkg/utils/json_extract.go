package utils

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	goxJsonUtils "github.com/devlibx/gox-base/v2/serialization/utils/json"
)

var trailingCommaRE = regexp.MustCompile(`,(\s*[}\]])`)

// ExtractJSON parses raw as T, trying progressively more lenient strategies
// since LLMs don't reliably follow "JSON only" instructions:
//  1. raw parses as-is.
//  2. a fenced code block (```json, ```, or single-backtick `...`) contains
//     valid JSON - every fenced block found is tried, not just the first.
//  3. brace-matching (string-literal aware) finds a balanced {...} object
//     anywhere in the text - every top-level object found is tried, not
//     just the first.
//
// Each candidate is also retried with trailing commas stripped before
// giving up on it, since that's a common small weak-model malformation
// (e.g. {"intent_score": 80,}) that would otherwise be rejected outright.
func ExtractJSON[T any](raw string) (T, error) {
	var zero T

	raw = strings.TrimSpace(raw)
	if raw == "" {
		return zero, errors.New("empty response")
	}

	var errs []error
	for _, finder := range defaultJSONCandidateFinders {
		for _, candidate := range finder.find(raw) {
			candidate = strings.TrimSpace(candidate)
			if candidate == "" {
				continue
			}

			result, err := unmarshalJSONCandidate[T](candidate)
			if err == nil {
				return result, nil
			}
			errs = append(errs, err)
		}
	}

	return zero, fmt.Errorf("could not extract valid JSON from response: %w", errors.Join(errs...))
}

func unmarshalJSONCandidate[T any](candidate string) (T, error) {
	if result, err := goxJsonUtils.StringToObject[T](candidate); err == nil {
		return result, nil
	}

	if sanitized := trailingCommaRE.ReplaceAllString(candidate, "$1"); sanitized != candidate {
		if result, err := goxJsonUtils.StringToObject[T](sanitized); err == nil {
			return result, nil
		}
	}

	var zero T
	return zero, fmt.Errorf("invalid json candidate: %q", candidate)
}
