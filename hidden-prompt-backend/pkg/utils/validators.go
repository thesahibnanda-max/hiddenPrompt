package utils

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"unicode"
)

func ValidateEmail(email string) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	parsed, err := mail.ParseAddress(email)
	if err != nil {
		return "", err
	}

	if parsed.Address != email {
		return "", fmt.Errorf("email must not contain a display name")
	}

	return parsed.Address, nil
}

func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("Password must be at least 8 characters long.")
	}
	if len(password) > 32 {
		return errors.New("Password must not exceed 32 characters.")
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool

	for _, r := range password {
		if r > unicode.MaxASCII {
			return errors.New("Password may only contain standard English letters, numbers, and symbols.")
		}

		switch {
		case 'A' <= r && r <= 'Z':
			hasUpper = true
		case 'a' <= r && r <= 'z':
			hasLower = true
		case '0' <= r && r <= '9':
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		default:
			return errors.New("Password contains a character that isn't allowed.")
		}
	}

	if !hasUpper {
		return errors.New("Password must contain at least one uppercase letter.")
	}
	if !hasLower {
		return errors.New("Password must contain at least one lowercase letter.")
	}
	if !hasDigit {
		return errors.New("Password must contain at least one digit.")
	}
	if !hasSpecial {
		return errors.New("Password must contain at least one special character.")
	}

	return nil
}
