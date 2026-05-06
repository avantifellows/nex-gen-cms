package views

import "github.com/avantifellows/nex-gen-cms/internal/models"

// JeeAdvancedExamName is the exam name returned by the exams API for the JEE Advanced exam.
const JeeAdvancedExamName = "JEE Advanced"

func GetSectionName(defaultType string, customName string) string {
	if customName != "" {
		return customName
	}

	switch defaultType {
	case "mcq_single_answer":
		return "MCQ Single Answer"
	case "mcq_multiple_answer":
		return "MCQ Multiple Answer"
	case "numerical_answer":
		return "Numerical Answer"
	case "integer_type":
		return "Integer Type"
	case "matrix_match":
		return "Matrix Match"
	default:
		return "Unknown"
	}
}

// ExamIDFromTest returns the first exam id on the test, or 0 if none.
func ExamIDFromTest(t *models.Test) int8 {
	if t == nil || len(t.ExamIDs) == 0 {
		return 0
	}
	return t.ExamIDs[0]
}

// SectionSubtypeForProblem returns the section key used on the add-test screen for grouping rows.
// Matrix match shares the MCQ single-answer section unless the test's exam id equals jeeAdvancedExamID
// (resolved from the exams API by name JeeAdvancedExamName). If jeeAdvancedExamID is 0 (not found),
// matrix_match is grouped under mcq_single_answer.
func SectionSubtypeForProblem(problemSubtype string, testExamID int8, jeeAdvancedExamID int16) string {
	if problemSubtype != "matrix_match" {
		return problemSubtype
	}
	if jeeAdvancedExamID == 0 {
		return "mcq_single_answer"
	}
	if int16(testExamID) == jeeAdvancedExamID {
		return "matrix_match"
	}
	return "mcq_single_answer"
}
