package dto

import "github.com/avantifellows/nex-gen-cms/internal/models"

type AddTestSearchData struct {
	TestPtr     *models.Test
	Problems    map[int]*models.Problem
	SelectedIDs map[int]bool
}
