package validation

import "testing"

func TestValidateOrderNumber(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid", "79927398713", true},      // classic Luhn valid
		{"valid", "4532015112830366", true}, // VISA test number
		{"invalid luhn", "79927398714", false},
		{"contains letters", "79927398A13", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateOrderNumber(tt.input)
			if got != tt.want {
				t.Errorf("ValidateOrderNumber(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
