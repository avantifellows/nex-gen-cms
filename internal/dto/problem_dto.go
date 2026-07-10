package dto

import "github.com/avantifellows/nex-gen-cms/internal/models"

type ProblemData struct {
	HomeData
	ProblemPtr *models.Problem
	TopicPtr   *models.Topic
	ChapterPtr *models.Chapter
}

type CopyProblemModalData struct {
	ProblemID          string
	SourceCurriculumID string
}
