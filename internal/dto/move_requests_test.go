package dto

import (
	"encoding/json"
	"testing"

	"github.com/avantifellows/nex-gen-cms/internal/models"
)

func jsonKeys(t *testing.T, value any) map[string]json.RawMessage {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	keys := map[string]json.RawMessage{}
	if err := json.Unmarshal(body, &keys); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return keys
}

// A problem move changes the chapter only. Sending any topic key would make db-service rewrite the problems'
// topics, so the request must never carry one, not even as null.
func TestMoveProblemsRequestCarriesNoTopic(t *testing.T) {
	keys := jsonKeys(t, MoveProblemsRequest{
		ResourceIDs:      []int{7, 8},
		CurriculumGrades: []models.CurriculumGrade{{CurriculumID: 1, GradeID: 10}},
		SubjectID:        2,
		ChapterID:        12,
		LangCode:         "en",
	})

	for _, unwanted := range []string{"topic_id", "topic_ids"} {
		if _, found := keys[unwanted]; found {
			t.Errorf("a problem move must not send %q", unwanted)
		}
	}
	for _, want := range []string{"resource_ids", "curriculum_grades", "subject_id", "chapter_id", "lang_code"} {
		if _, found := keys[want]; !found {
			t.Errorf("a problem move should send %q", want)
		}
	}
	if string(keys["chapter_id"]) != "12" {
		t.Errorf("chapter_id = %s, want 12", keys["chapter_id"])
	}
}

// A resource move clears the old topic when none is chosen, by sending an explicit null: left out, db-service
// would keep the old topic row. This type must keep doing that.
func TestMoveResourcesRequestStillSendsAnExplicitNullTopic(t *testing.T) {
	keys := jsonKeys(t, MoveResourcesRequest{ResourceIDs: []int{3}, ChapterID: 12})

	topic, found := keys["topic_id"]
	if !found || string(topic) != "null" {
		t.Errorf("a resource move without a topic must send topic_id as null, got %q (present: %v)", topic, found)
	}

	topicID := int16(5)
	keys = jsonKeys(t, MoveResourcesRequest{ResourceIDs: []int{3}, ChapterID: 12, TopicID: &topicID})
	if string(keys["topic_id"]) != "5" {
		t.Errorf("a resource move to a topic should send it, got %s", keys["topic_id"])
	}
}
