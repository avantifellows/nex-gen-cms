package handlerutils

import (
	"fmt"
	"net/url"

	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/utils"
)

// ParseCurriculumGradesFromForm reads curriculum[] and grade[] from form values (works after
// ParseForm or ParseMultipartForm).
func ParseCurriculumGradesFromForm(form url.Values) ([]models.CurriculumGrade, error) {
	curriculums := form["curriculum[]"]
	grades := form["grade[]"]

	var curriculumGrades []models.CurriculumGrade
	for i := range curriculums {
		curriculumId, err := utils.StringToIntType[int16](curriculums[i])
		if err != nil {
			return nil, fmt.Errorf("invalid curriculum id at index %d", i)
		}

		gradeId, err := utils.StringToIntType[int8](grades[i])
		if err != nil {
			return nil, fmt.Errorf("invalid grade id at index %d", i)
		}

		curriculumGrades = append(curriculumGrades, models.CurriculumGrade{
			CurriculumID: curriculumId,
			GradeID:      gradeId,
		})
	}
	return curriculumGrades, nil
}
