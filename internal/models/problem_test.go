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

func TestProblemCopyToKeepsTopicsAndMovesToTheChapter(t *testing.T) {
	original := &Problem{
		ID: 7, Code: "P0000007", ChapterID: 9, TopicIDs: []int16{5, 90},
		Paragraph: &ProblemParagraph{ID: 3, Body: "passage"},
	}

	copied := original.CopyTo(&Chapter{ID: 12}, 1, 10, 2)

	if copied.ID != 0 || copied.Code != "" {
		t.Errorf("a copy is a new problem: id and code must be cleared, got %d %q", copied.ID, copied.Code)
	}
	if copied.ChapterID != 12 || copied.CurriculumID != 1 || copied.GradeID != 10 || copied.SubjectID != 2 {
		t.Errorf("the copy should carry the destination, got chapter %d curriculum %d grade %d subject %d",
			copied.ChapterID, copied.CurriculumID, copied.GradeID, copied.SubjectID)
	}
	// topics need not belong to the chapter, so the copy keeps all of them
	if !reflect.DeepEqual(copied.TopicIDs, []int16{5, 90}) {
		t.Errorf("the copy should keep the original's topics, got %v", copied.TopicIDs)
	}
	// but it must not share them: editing the copy's topics must leave the original alone
	copied.TopicIDs[0] = 1
	if original.TopicIDs[0] != 5 {
		t.Errorf("the copy shares its topics list with the original")
	}
	// the passage is copied as a new one, and the original keeps its own
	if copied.Paragraph == nil || copied.Paragraph.ID != 0 || original.Paragraph.ID != 3 {
		t.Errorf("the copy should get a new passage while the original keeps its own")
	}
	if original.ChapterID != 9 {
		t.Errorf("the original must not change")
	}
}

// The single legacy topic is gone: a problem sent to db-service (a copy, for one) carries topic_ids only.
func TestProblemJSONCarriesNoLegacyTopicID(t *testing.T) {
	body, err := json.Marshal(Problem{ChapterID: 9, TopicIDs: []int16{5}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	keys := map[string]json.RawMessage{}
	if err := json.Unmarshal(body, &keys); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, found := keys["topic_id"]; found {
		t.Errorf("a problem must not send the legacy topic_id")
	}
	if string(keys["topic_ids"]) != "[5]" {
		t.Errorf("topic_ids = %s, want [5]", keys["topic_ids"])
	}
}
