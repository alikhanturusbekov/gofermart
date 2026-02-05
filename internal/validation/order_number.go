package validation

import "regexp"

// ValidateOrderNumber checks order number
func ValidateOrderNumber(number string) bool {
	return validateOnlyDigits(number) && validateLuhn(number)
}

// validateOnlyDigits string consists only of digits
func validateOnlyDigits(number string) bool {
	return regexp.MustCompile(`^\d+$`).MatchString(number)
}

// validateLuhn validates the string using Luhn Algorithm
func validateLuhn(number string) bool {
	sum := 0
	alt := false

	// iterates string from right to left
	for i := len(number) - 1; i >= 0; i-- {
		c := number[i]

		if c < '0' || c > '9' {
			return false
		}

		n := int(c - '0')

		if alt {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}

		sum += n
		alt = !alt
	}

	return sum%10 == 0
}
