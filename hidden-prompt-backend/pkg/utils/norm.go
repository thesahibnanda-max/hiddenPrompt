package utils

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

func NormalizeString(dataString string) string {
	s := strings.ToLower(strings.TrimSpace(dataString))
	s = norm.NFC.String(s)
	s = removePunctuation(s)
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func removePunctuation(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsPunct(r) {
			return -1
		}
		return r
	}, s)
}
