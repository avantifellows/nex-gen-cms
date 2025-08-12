package dto

import "github.com/avantifellows/nex-gen-cms/internal/models"

type PaperData struct {
	TestPtr     *models.Test
	ProblemsMap map[int]*models.Problem
	TestRule    *models.TestRule
}
