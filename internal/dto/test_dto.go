package dto

import "github.com/avantifellows/nex-gen-cms/internal/models"

type TestData struct {
	HomeData
	TestPtr  *models.Test
	Problems map[int]*models.Problem // key = Problem ID
	TestRule *models.TestRule
	// True when the test's exam is JEE Advanced (matrix match gets its own section on add-test).
	IsJeeAdvancedExam bool `json:"IsJeeAdvancedExam"`
}
