package pdfimport

import "testing"

func TestResolveSubtype(t *testing.T) {
	tests := []struct {
		name string
		q    ExtractedQuestion
		want string
	}{
		{"pass through", ExtractedQuestion{Type: "mcq_single_answer", Options: []string{"a"}}, "mcq_single_answer"},
		{"matrix_match", ExtractedQuestion{Type: "matrix_match", Options: []string{"a"}}, "matrix_match"},
		{"invalid with options", ExtractedQuestion{Type: "garbage", Options: []string{"x"}}, "mcq_single_answer"},
		{"invalid no options", ExtractedQuestion{Type: "garbage"}, "numerical_answer"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveSubtype(tt.q); got != tt.want {
				t.Errorf("resolveSubtype() = %q, want %q", got, tt.want)
			}
		})
	}
}
