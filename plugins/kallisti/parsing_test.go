package main

import (
	"regexp"
	"testing"
)

func TestParseEntity(t *testing.T) {
	reParens := regexp.MustCompile(`\s*\((\d+)\)$`)
	reBrackets := regexp.MustCompile(`\s*\[(\d+)\]$`)

	tests := []struct {
		input        string
		re           *regexp.Regexp
		expectedName string
		expectedQty  int
	}{
		{
			input:        "A sword",
			re:           reParens,
			expectedName: "A sword",
			expectedQty:  1,
		},
		{
			input:        "A gold coin (50)",
			re:           reParens,
			expectedName: "A gold coin",
			expectedQty:  50,
		},
		{
			input:        "Something [2]",
			re:           reBrackets,
			expectedName: "Something",
			expectedQty:  2,
		},
		{
			input:        "Something [2]",
			re:           reParens,
			expectedName: "Something [2]",
			expectedQty:  1,
		},
		{
			input:        "Something (2)",
			re:           reBrackets,
			expectedName: "Something (2)",
			expectedQty:  1,
		},
		{
			input:        "A friendly receptionist... [ sanc ]",
			re:           reBrackets,
			expectedName: "A friendly receptionist... [ sanc ]",
			expectedQty:  1,
		},
		{
			input:        "A cute rabbit is here. [2]",
			re:           reBrackets,
			expectedName: "A cute rabbit is here.",
			expectedQty:  2,
		},
		{
			input:        "A battered golden harp has been tipped over. (2)",
			re:           reParens,
			expectedName: "A battered golden harp has been tipped over.",
			expectedQty:  2,
		},
		{
			input:        "  Whitespace padding (10)  ",
			re:           reParens,
			expectedName: "Whitespace padding",
			expectedQty:  10,
		},
	}

	for _, tt := range tests {
		got := ParseEntity(tt.input, tt.re)
		if got.Name != tt.expectedName {
			t.Errorf("ParseEntity(%q).Name = %q, want %q", tt.input, got.Name, tt.expectedName)
		}
		if got.Quantity != tt.expectedQty {
			t.Errorf("ParseEntity(%q).Quantity = %d, want %d", tt.input, got.Quantity, tt.expectedQty)
		}
	}
}
