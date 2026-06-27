package handlerutils

import (
	"fmt"
	"net/http"

	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/utils"
)

const SubjectsEndPoint = "subject"
const SubjectsKey = "subjects"

func FetchSelectedSubject(
	subIDStr string,
	subjectsService *services.Service[models.Subject],
) (*models.Subject, int, error) {
	subjectID, err := utils.StringToIntType[int8](subIDStr)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid subject ID: %w", err)
	}

	selectedSubPtr, err := subjectsService.GetObject(subIDStr, func(subject *models.Subject) bool {
		return subject.ID == subjectID
	}, SubjectsKey, SubjectsEndPoint)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error fetching subject: %w", err)
	}

	// Fetch and assign parent subject name if ParentID is non-zero
	if selectedSubPtr.ParentID != 0 {
		parentIDStr := utils.IntToString[int8](selectedSubPtr.ParentID)
		parentSubPtr, err := subjectsService.GetObject(parentIDStr, func(subject *models.Subject) bool {
			return subject.ID == selectedSubPtr.ParentID
		}, SubjectsKey, SubjectsEndPoint)

		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("error fetching parent subject: %w", err)
		}
		selectedSubPtr.ParentName = parentSubPtr.Name
	}

	return selectedSubPtr, http.StatusOK, nil
}
