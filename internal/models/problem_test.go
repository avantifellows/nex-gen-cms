package models

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestProblemUnmarshalTopicIDs(t *testing.T) {
	tests := []struct {
		name          string
		body          string
		wantTopicIDs  []int16
		wantChapterID int16
	}{
		{"several topics, one from another chapter", `{"chapter_id": 7, "topic_ids": [11, 12, 90]}`, []int16{11, 12, 90}, 7},
		{"empty list", `{"chapter_id": 7, "topic_ids": []}`, []int16{}, 7},
		{"key absent", `{"chapter_id": 7}`, nil, 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var problem Problem
			if err := json.Unmarshal([]byte(tt.body), &problem); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if !reflect.DeepEqual(problem.TopicIDs, tt.wantTopicIDs) {
				t.Errorf("TopicIDs = %#v, want %#v", problem.TopicIDs, tt.wantTopicIDs)
			}
			if problem.ChapterID != tt.wantChapterID {
				t.Errorf("ChapterID = %d, want %d", problem.ChapterID, tt.wantChapterID)
			}
		})
	}
}
