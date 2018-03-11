package main

import (
	"regexp"
	"strconv"
	"strings"
)

func main() {
}

var phoneNumberMatcher = regexp.MustCompile("[0-9]+")

func normalizePhoneNumber(phoneNumber string) string {
	normalized := strings.FieldsFunc(phoneNumber, func(r rune) bool {
		return !phoneNumberMatcher.MatchString(strconv.QuoteRune(r))
	})
	return strings.Join(normalized, "")
}
