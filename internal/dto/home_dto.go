package dto

import "github.com/avantifellows/nex-gen-cms/internal/models"

type HomeData struct {
	CurriculumID     int16
	GradeID          int8
	SubjectID        int8
	ChapterPtr       *models.Chapter
	ChapterSortState SortState
	TestPtr          *models.Test
	ProblemPtr       *models.Problem
}
