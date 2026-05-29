package dto

import "github.com/avantifellows/nex-gen-cms/internal/models"

// ImportTestReviewData is passed to import_test_review.html.
type ImportTestReviewData struct {
	HomeData
	TestPtr  *models.Test
	Problems []*models.Problem
}
