// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package handler

import (
	"strings"
	"unicode"
)

// normalizeLocation trims and collapses whitespace so "  Saillans  " and
// "Saillans" store identically. Case is preserved (SQL comparisons use LOWER).
func normalizeLocation(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

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
