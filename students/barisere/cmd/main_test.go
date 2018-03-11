package main

import (
	"testing"
)

func TestNormalizePhoneNumber(t *testing.T) {
	testData := []struct {
		message  string
		data     string
		expected string
	}{
		{"A phone number in correct form should not be changed", "1234567890", "1234567890"},
		{"Spaces should be removed from phone numbers", "123 456 7891", "1234567891"},
		{"Parentheses should be removed from phone numbers", "(123) 456 7892", "1234567892"},
		{"Hyphens should be removed from phone numbers", "(123) 456-7893", "1234567893"},
	}
	for _, data := range testData {
		t.Run(data.message, func(t *testing.T) {
			if got := normalizePhoneNumber(data.data); got != data.expected {
				t.Errorf("Expected %s, got %s\n", data.expected, got)
			}
		})
	}
}
