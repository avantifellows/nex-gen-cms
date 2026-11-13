package dto

import "github.com/avantifellows/nex-gen-cms/internal/models"

// MoveProblemsRequest moves problems to another chapter. It has no topic field on purpose: a move leaves each
// problem's topics as they are, and they are edited on the problem's own screen.
type MoveProblemsRequest struct {
	ResourceIDs      []int                    `json:"resource_ids"`
	CurriculumGrades []models.CurriculumGrade `json:"curriculum_grades"`
	SubjectID        int8                     `json:"subject_id"`
	ChapterID        int16                    `json:"chapter_id"`
	LangCode         string                   `json:"lang_code"`
}
