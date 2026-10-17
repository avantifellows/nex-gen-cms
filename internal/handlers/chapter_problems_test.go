package handlers

import "testing"

func TestChapterProblemsQuery(t *testing.T) {
	tests := []struct {
		name      string
		chapterID string
		topicID   string
		want      string
		wantErr   string
	}{
		{"chapter only (all topics)", "12", "", "?chapter_id=12", ""},
		{"chapter with topic filter", "12", "45", "?chapter_id=12&topic_id=45", ""},
		{"missing chapter", "", "45", "", "Invalid Chapter ID"},
		{"non-numeric chapter", "abc", "", "", "Invalid Chapter ID"},
		{"non-numeric topic", "12", "abc", "", "Invalid Topic ID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := chapterProblemsQuery(tt.chapterID, tt.topicID)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("query = %q, want %q", got, tt.want)
			}
		})
	}
}
