package views

import (
	"testing"
)

func TestGetSectionName(t *testing.T) {
	tests := []struct {
		name        string
		defaultType string
		customName  string
		expected    string
	}{
		{
			name:        "Custom name overrides default",
			defaultType: "mcq_single_answer",
			customName:  "My Custom Section",
			expected:    "My Custom Section",
		},
		{
			name:        "MCQ single answer default",
			defaultType: "mcq_single_answer",
			customName:  "",
			expected:    "MCQ Single Answer",
		},
		{
			name:        "Numerical answer default",
			defaultType: "numerical_answer",
			customName:  "",
			expected:    "Numerical Answer",
		},
		{
			name:        "Integer type default",
			defaultType: "integer_type",
			customName:  "",
			expected:    "Integer Type",
		},
		{
			name:        "Unknown type fallback",
			defaultType: "some_random_type",
			customName:  "",
			expected:    "Unknown",
		},
		{
			name:        "Empty default type",
			defaultType: "",
			customName:  "",
			expected:    "Unknown",
		},
		{
			name:        "Custom name with unknown type",
			defaultType: "unknown_type",
			customName:  "Custom Name",
			expected:    "Custom Name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetSectionName(tt.defaultType, tt.customName)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
