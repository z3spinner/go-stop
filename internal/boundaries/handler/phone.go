package handler

import (
	"strings"
	"unicode"
)

// normalizePhone strips whitespace, dashes, dots and parentheses so that
// "06 12 34 56 78", "06-12-34-56-78", and "0612345678" all become the same
// canonical string. The leading '+' is preserved for international numbers.
// Empty strings are returned as-is (callers validate presence separately).
func normalizePhone(phone string) string {
	var b strings.Builder
	for i, r := range phone {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		} else if r == '+' && i == 0 {
			b.WriteRune(r)
		}
		// drop spaces, dashes, dots, parentheses, etc.
	}
	return b.String()
}
