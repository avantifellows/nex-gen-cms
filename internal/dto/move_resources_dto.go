package dto

import "github.com/avantifellows/nex-gen-cms/internal/models"

type MoveResourcesRequest struct {
	ResourceIDs      []int                    `json:"resource_ids"`
	CurriculumGrades []models.CurriculumGrade `json:"curriculum_grades"`
	SubjectID        int8                     `json:"subject_id"`
	TopicID          int16                    `json:"topic_id,omitempty"`
	ChapterID        int16                    `json:"chapter_id"`
	LangCode         string                   `json:"lang_code"`
}
