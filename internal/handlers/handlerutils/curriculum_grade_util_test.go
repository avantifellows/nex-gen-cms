package handlerutils

import (
	"net/url"
	"testing"
)

func TestParseCurriculumGradesFromForm(t *testing.T) {
	form := url.Values{}
	form.Add("curriculum[]", "1")
	form.Add("grade[]", "10")
	form.Add("curriculum[]", "2")
	form.Add("grade[]", "11")

	got, err := ParseCurriculumGradesFromForm(form)
	if err != nil {
		t.Fatalf("ParseCurriculumGradesFromForm: %v", err)
	}
	if len(got) != 2 || got[0].CurriculumID != 1 || got[0].GradeID != 10 || got[1].CurriculumID != 2 {
		t.Fatalf("got %+v", got)
	}
}
