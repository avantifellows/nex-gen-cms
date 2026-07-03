package handlers

import (
	"fmt"
	"net/http"

	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/utils"
)

const GRADE_DROPDOWN_NAME = "grade-dropdown"
const GRADE_COMMON_VALUE int8 = -1

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
	id, err := utils.StringToIntType[int8](gradeParam)
	if err != nil || id == 0 {
		return 0, false, false
	}
	return id, id == GRADE_COMMON_VALUE, true
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

func (h *GradesHandler) GetGrades(responseWriter http.ResponseWriter, _ *http.Request) {
	renderEntityList(responseWriter, h.service, getGradesEndPoint, gradesKey, gradesTemplate, "grades")
}
