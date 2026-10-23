package dto

import "github.com/avantifellows/nex-gen-cms/internal/models"

type TopicsData struct {
	ChapterID string
}

// TopicChecklistData feeds the add / edit / copy problem screen's Topics checklist.
type TopicChecklistData struct {
	Topics   []*models.Topic       // the chapter's own topics
	Other    []TopicChecklistExtra // topics the problem has that the list above does not show
	Selected map[int16]bool        // ids to tick
}

// TopicChecklistExtra is a ticked topic that is not one of the chapter's listed topics: it belongs to
// another chapter, or it is archived. Note says which.
type TopicChecklistExtra struct {
	Topic *models.Topic
	Note  string
}
