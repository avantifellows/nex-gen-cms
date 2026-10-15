package handlers

import (
	"encoding/json"
	"html/template"
	"reflect"
	"testing"

	"github.com/avantifellows/nex-gen-cms/internal/dto"
	"github.com/avantifellows/nex-gen-cms/internal/models"
)

func TestExtractSimilarityLanguagesIncludesOptions(t *testing.T) {
	problem := models.Problem{
		LangVersions: []models.LangVersion{
			{
				LangCode: "en",
				MetaData: models.ProbMetaData{
					Question: template.HTML("Which is correct?"),
					Options:  []template.HTML{"<p>A</p>", "<p>B</p>", "<p>C</p>", "<p>D</p>"},
				},
			},
			{
				LangCode: "hi",
				MetaData: models.ProbMetaData{
					Question: template.HTML("सही कौन सा है?"),
					Options:  []template.HTML{"क", "ख"},
				},
			},
		},
	}

	got := extractSimilarityLanguages(problem)

	want := []dto.SimilarSearchLanguage{
		{LangCode: "en", Text: "Which is correct?", Options: []string{"<p>A</p>", "<p>B</p>", "<p>C</p>", "<p>D</p>"}},
		{LangCode: "hi", Text: "सही कौन सा है?", Options: []string{"क", "ख"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("languages = %+v, want %+v", got, want)
	}
}

func TestExtractSimilarityLanguagesSkipsBlankQuestionEvenWithOptions(t *testing.T) {
	problem := models.Problem{
		LangVersions: []models.LangVersion{
			{LangCode: "en", MetaData: models.ProbMetaData{Options: []template.HTML{"A", "B"}}},
		},
	}

	if got := extractSimilarityLanguages(problem); len(got) != 0 {
		t.Fatalf("languages = %+v, want none", got)
	}
}

func TestSimilarSearchLanguageOmitsOptionsWhenEmpty(t *testing.T) {
	problem := models.Problem{
		LangVersions: []models.LangVersion{
			{LangCode: "en", MetaData: models.ProbMetaData{Question: template.HTML("Integer answer question")}},
		},
	}

	body, err := json.Marshal(dto.SimilarSearchRequest{Languages: extractSimilarityLanguages(problem)})
	if err != nil {
		t.Fatal(err)
	}

	want := `{"languages":[{"lang_code":"en","text":"Integer answer question"}]}`
	if string(body) != want {
		t.Fatalf("body = %s, want %s", body, want)
	}
}

func TestSimilarSearchRequestSendsOptionsWhenPresent(t *testing.T) {
	problem := models.Problem{
		LangVersions: []models.LangVersion{
			{LangCode: "en", MetaData: models.ProbMetaData{
				Question: template.HTML("Q"),
				Options:  []template.HTML{"A", "B"},
			}},
		},
	}

	body, err := json.Marshal(dto.SimilarSearchRequest{Languages: extractSimilarityLanguages(problem)})
	if err != nil {
		t.Fatal(err)
	}

	want := `{"languages":[{"lang_code":"en","text":"Q","options":["A","B"]}]}`
	if string(body) != want {
		t.Fatalf("body = %s, want %s", body, want)
	}
}
