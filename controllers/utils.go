package controllers

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

	secretsv1alpha1 "github.com/example/another-secrets-operator/api/v1alpha1"
)

// validateGeneratorSpec validates that the generator specification is valid
func validateGeneratorSpec(spec secretsv1alpha1.AGeneratorSpec) error {
	// Ensure at least one character type is enabled
	if !spec.IncludeUppercase && !spec.IncludeLowercase && !spec.IncludeNumbers && !spec.IncludeSpecialChars {
		return errors.New("at least one character type (uppercase, lowercase, numbers, or special chars) must be enabled")
	}
	
	// Ensure length is positive
	if spec.Length <= 0 {
		return errors.New("length must be greater than 0")
	}
	
	return nil
}

// generateRandomString generates a random string according to the generator specification
func generateRandomString(spec secretsv1alpha1.AGeneratorSpec) (string, error) {
	var chars string
	
	if spec.IncludeUppercase {
		chars += "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	}
	
	if spec.IncludeLowercase {
		chars += "abcdefghijklmnopqrstuvwxyz"
	}
	
	if spec.IncludeNumbers {
		chars += "0123456789"
	}
	
	if spec.IncludeSpecialChars {
		chars += spec.SpecialChars
	}
	
	if len(chars) == 0 {
		return "", errors.New("no character set defined for password generation")
	}
	
	result := make([]byte, spec.Length)
	maxVal := big.NewInt(int64(len(chars)))
	
	for i := 0; i < spec.Length; i++ {
		randomIndex, err := rand.Int(rand.Reader, maxVal)
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %v", err)
		}
		result[i] = chars[randomIndex.Int64()]
	}
	
	return string(result), nil
}