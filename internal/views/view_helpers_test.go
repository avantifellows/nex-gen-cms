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
			name:        "Matrix match default",
			defaultType: "matrix_match",
			customName:  "",
			expected:    "Matrix Match",
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

func TestSectionSubtypeForProblem(t *testing.T) {
	const jeeID int16 = 7

	tests := []struct {
		name              string
		subtype           string
		testExamID        int8
		jeeAdvancedExamID int16
		want              string
	}{
		{"matrix same exam as JEE Advanced id", "matrix_match", 7, jeeID, "matrix_match"},
		{"matrix different exam", "matrix_match", 3, jeeID, "mcq_single_answer"},
		{"matrix JEE id unknown", "matrix_match", 7, 0, "mcq_single_answer"},
		{"mcq unchanged", "mcq_single_answer", 0, jeeID, "mcq_single_answer"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SectionSubtypeForProblem(tt.subtype, tt.testExamID, tt.jeeAdvancedExamID); got != tt.want {
				t.Errorf("SectionSubtypeForProblem(%q, %d, %d) = %q, want %q",
					tt.subtype, tt.testExamID, tt.jeeAdvancedExamID, got, tt.want)
			}
		})
	}
}
