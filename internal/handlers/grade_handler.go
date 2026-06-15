package handlers

import (
	"fmt"
	"net/http"

	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/internal/views"
	"github.com/avantifellows/nex-gen-cms/utils"
)

const GRADE_DROPDOWN_NAME = "grade-dropdown"
const GRADE_COMMON_VALUE = "common"

const getGradesEndPoint = "grade"
const gradesKey = "grades"
const gradesTemplate = "grades.html"

type GradesHandler struct {
	service *services.Service[models.Grade]
}

func NewGradesHandler(service *services.Service[models.Grade]) *GradesHandler {
	return &GradesHandler{
		service: service,
	}
}

func parseGradeFilter(gradeParam string) (gradeId int8, isCommon bool, ok bool) {
	if gradeParam == "" {
		return 0, false, false
	}
	if gradeParam == GRADE_COMMON_VALUE {
		return 0, true, true
	}
	id, err := utils.StringToIntType[int8](gradeParam)
	if err != nil || id == 0 {
		return 0, false, false
	}
	return id, false, true
}

func appendGradeIDQueryParam(queryParams string, gradeParam string) (string, bool) {
	gradeId, isCommon, ok := parseGradeFilter(gradeParam)
	if !ok {
		return "", false
	}
	if isCommon {
		return queryParams, true
	}
	return queryParams + fmt.Sprintf("&grade_id=%d", gradeId), true
}

func (h *GradesHandler) GetGrades(responseWriter http.ResponseWriter, request *http.Request) {
	grades, err := h.service.GetList(getGradesEndPoint, gradesKey, false, false)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching grades: %v", err), http.StatusInternalServerError)
		return
	}

	// Load grades.html
	views.ExecuteTemplate(gradesTemplate, responseWriter, grades, nil)
}
