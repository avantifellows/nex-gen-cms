package dto

import (
	"github.com/avantifellows/nex-gen-cms/internal/constants"
)

type SortState struct {
	Column string
	Order  constants.SortOrder
}

type TopicsData struct {
	ChapterId       string
	TopicsSortState SortState
}
