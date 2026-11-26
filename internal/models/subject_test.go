package models

import (
	"reflect"
	"testing"
)

func subjectNamed(id int8, english string) *Subject {
	return &Subject{ID: id, Name: []SubjectLang{
		{LangCode: "hi", SubName: "हिंदी " + english},
		{LangCode: "en", SubName: english},
	}}
}

func TestMatchSubjectsByNameKeepsNameOrderAndSkipsMissing(t *testing.T) {
	biology, physics, chemistry := subjectNamed(7, "Biology"), subjectNamed(3, "Physics"), subjectNamed(5, "chemistry")

	got := MatchSubjectsByName([]*Subject{biology, physics, chemistry}, []string{"Maths", "Physics", "Chemistry"})

	want := []*Subject{physics, chemistry}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
