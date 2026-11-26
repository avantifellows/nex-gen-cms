package models

import (
	"reflect"
	"testing"
)

func TestSyllabusSubjectNames(t *testing.T) {
	cases := map[string][]string{
		"JEE Mains": {"Maths", "Physics", "Chemistry"},
		"NEET":      {"Physics", "Chemistry", "Biology"},
		"CLAT":      nil,
	}
	for name, want := range cases {
		if got := (Curriculum{Name: name}).SyllabusSubjectNames(); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got %v, want %v", name, got, want)
		}
	}
}
