package dto

import "github.com/avantifellows/nex-gen-cms/internal/models"

type ProblemData struct {
	HomeData
	ProblemPtr *models.Problem
	TopicPtrs  []*models.Topic // the problem's topics, in the order of its TopicIDs
	ChapterPtr *models.Chapter
}
