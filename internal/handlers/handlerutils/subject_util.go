package handlerutils

import (
	"fmt"
	"net/http"

	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/utils"
)

func FetchSelectedSubject(
	subIdStr string,
	subjectsService *services.Service[models.Subject],
	cacheKey, endpoint string,
) (*models.Subject, int, error) {
	subjectId, err := utils.StringToIntType[int8](subIdStr)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid subject ID: %w", err)
	}

	selectedSubPtr, err := subjectsService.GetObject(subIdStr, func(subject *models.Subject) bool {
		return subject.ID == subjectId
	}, cacheKey, endpoint)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error fetching subject: %w", err)
	}

	return selectedSubPtr, http.StatusOK, nil
}
