package handlerutils

import (
	"fmt"
	"net/http"

	"github.com/avantifellows/nex-gen-cms/internal/services"
)

func GetEntityByID[T any, ID comparable](
	idStr string,
	service *services.Service[T],
	cacheKey string,
	endpoint string,
	idParser func(string) (ID, error),
	idMatcher func(*T, ID) bool,
	entityName string,
) (*T, int, error) {
	id, err := idParser(idStr)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid %s ID: %w", entityName, err)
	}

	entityPtr, err := service.GetObject(idStr, func(e *T) bool {
		return idMatcher(e, id)
	}, cacheKey, endpoint)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error fetching %s: %w", entityName, err)
	}

	return entityPtr, http.StatusOK, nil
}
