package dto

import (
	"reflect"
	"testing"

	"github.com/avantifellows/nex-gen-cms/internal/models"
)

func TestSyllabusOptionsWithoutSavedSyllabus(t *testing.T) {
	options := SyllabusOptions{TestGradeID: 11}

	if !options.GradeTicked(11) || options.GradeTicked(12) {
		t.Fatal("only the test's own grade should start ticked")
	}
	if options.SubjectTicked(2) {
		t.Fatal("no subject should start ticked")
	}
	if got := options.SavedChapterIDs(); len(got) != 0 {
		t.Fatalf("expected no chapters, got %v", got)
	}
}

func TestSyllabusOptionsWithSavedSyllabus(t *testing.T) {
	options := SyllabusOptions{TestGradeID: 11, Saved: &models.TestSyllabus{
		GradeIDs: []int8{12},
		Subjects: []models.SyllabusSubject{{SubjectID: 2, ChapterIDs: []int16{101, 102}}, {SubjectID: 3, ChapterIDs: []int16{205}}},
	}}

	if options.GradeTicked(11) || !options.GradeTicked(12) {
		t.Fatal("the saved grades replace the test's own grade")
	}
	if !options.SubjectTicked(2) || !options.SubjectTicked(3) || options.SubjectTicked(4) {
		t.Fatal("exactly the saved subjects should start ticked")
	}
	if got := options.SavedChapterIDs(); !reflect.DeepEqual(got, []int16{101, 102, 205}) {
		t.Fatalf("got %v", got)
	}
}
