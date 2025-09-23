package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	secretsv1alpha1 "github.com/yaso/yet-another-secrets-operator/api/v1alpha1"
)

func TestValidateGeneratorSpec(t *testing.T) {
	tests := []struct {
		name    string
		spec    secretsv1alpha1.AGeneratorSpec
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid spec with all character types",
			spec: secretsv1alpha1.AGeneratorSpec{
				Length:              16,
				IncludeUppercase:    true,
				IncludeLowercase:    true,
				IncludeNumbers:      true,
				IncludeSpecialChars: true,
			},
			wantErr: false,
		},
		{
			name: "valid spec with only uppercase",
			spec: secretsv1alpha1.AGeneratorSpec{
				Length:              8,
				IncludeUppercase:    true,
				IncludeLowercase:    false,
				IncludeNumbers:      false,
				IncludeSpecialChars: false,
			},
			wantErr: false,
		},
		{
			name: "valid spec with only lowercase",
			spec: secretsv1alpha1.AGeneratorSpec{
				Length:              12,
				IncludeUppercase:    false,
				IncludeLowercase:    true,
				IncludeNumbers:      false,
				IncludeSpecialChars: false,
			},
			wantErr: false,
		},
		{
			name: "valid spec with only numbers",
			spec: secretsv1alpha1.AGeneratorSpec{
				Length:              6,
				IncludeUppercase:    false,
				IncludeLowercase:    false,
				IncludeNumbers:      true,
				IncludeSpecialChars: false,
			},
			wantErr: false,
		},
		{
			name: "valid spec with only special characters",
			spec: secretsv1alpha1.AGeneratorSpec{
				Length:              10,
				IncludeUppercase:    false,
				IncludeLowercase:    false,
				IncludeNumbers:      false,
				IncludeSpecialChars: true,
			},
			wantErr: false,
		},
		{
			name: "valid spec with mixed characters",
			spec: secretsv1alpha1.AGeneratorSpec{
				Length:              24,
				IncludeUppercase:    true,
				IncludeLowercase:    false,
				IncludeNumbers:      true,
				IncludeSpecialChars: false,
			},
			wantErr: false,
		},
		{
			name: "invalid spec - no character types enabled",
			spec: secretsv1alpha1.AGeneratorSpec{
				Length:              16,
				IncludeUppercase:    false,
				IncludeLowercase:    false,
				IncludeNumbers:      false,
				IncludeSpecialChars: false,
			},
			wantErr: true,
			errMsg:  "at least one character type",
		},
		{
			name: "invalid spec - zero length",
			spec: secretsv1alpha1.AGeneratorSpec{
				Length:              0,
				IncludeUppercase:    true,
				IncludeLowercase:    true,
				IncludeNumbers:      true,
				IncludeSpecialChars: false,
			},
			wantErr: true,
			errMsg:  "length must be greater than 0",
		},
		{
			name: "invalid spec - negative length",
			spec: secretsv1alpha1.AGeneratorSpec{
				Length:              -5,
				IncludeUppercase:    true,
				IncludeLowercase:    true,
				IncludeNumbers:      true,
				IncludeSpecialChars: false,
			},
			wantErr: true,
			errMsg:  "length must be greater than 0",
		},
		{
			name: "edge case - length 1",
			spec: secretsv1alpha1.AGeneratorSpec{
				Length:              1,
				IncludeUppercase:    true,
				IncludeLowercase:    false,
				IncludeNumbers:      false,
				IncludeSpecialChars: false,
			},
			wantErr: false,
		},
		{
			name: "edge case - very large length",
			spec: secretsv1alpha1.AGeneratorSpec{
				Length:              1000,
				IncludeUppercase:    true,
				IncludeLowercase:    true,
				IncludeNumbers:      true,
				IncludeSpecialChars: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGeneratorSpec(tt.spec)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
