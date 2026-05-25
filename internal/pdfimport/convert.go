package pdfimport

import (
	"strings"

	htmltemplate "html/template"

	"github.com/avantifellows/nex-gen-cms/internal/models"
)

const resourceTypeProblem = "problem"

// validProblemSubtypes matches web/html/problem_type_options.html.
var validProblemSubtypes = map[string]struct{}{
	"mcq_single_answer":   {},
	"mcq_multiple_answer": {},
	"integer_type":        {},
	"numerical_answer":    {},
	"matrix_match":        {},
}

// ProblemsFromExtracted maps parsed PDF rows into CMS Problem objects.
func ProblemsFromExtracted(questions []ExtractedQuestion) []models.Problem {
	out := make([]models.Problem, 0, len(questions))
	for _, q := range questions {
		out = append(out, extractedToProblem(q))
	}
	return out
}

func extractedToProblem(q ExtractedQuestion) models.Problem {
	questionHTML := q.ProcessedText
	if questionHTML == "" {
		questionHTML = htmltemplate.HTML(q.Text)
	}

	options := make([]htmltemplate.HTML, len(q.Options))
	for i, opt := range q.Options {
		options[i] = htmltemplate.HTML(opt)
	}

	return models.Problem{
		Type:    resourceTypeProblem,
		Subtype: resolveSubtype(q),
		MetaData: models.ProbMetaData{
			Question: questionHTML,
			Options:  options,
		},
	}
}

// resolveSubtype passes through question_type when it is a valid CMS subtype (as prompted).
// postProcessQuestions may already have set matrix_match from layout detection.
// If the model returns an unknown value, fall back from options presence only.
func resolveSubtype(q ExtractedQuestion) string {
	t := strings.ToLower(strings.TrimSpace(q.Type))
	if _, ok := validProblemSubtypes[t]; ok {
		return t
	}
	if len(q.Options) > 0 {
		return "mcq_single_answer"
	}
	return "numerical_answer"
}
