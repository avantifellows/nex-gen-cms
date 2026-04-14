package dto

import "github.com/avantifellows/nex-gen-cms/internal/models"

type TestData struct {
	HomeData
	TestPtr  *models.Test
	Problems map[int]*models.Problem // key = Problem ID
	TestRule *models.TestRule
}
