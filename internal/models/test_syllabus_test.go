package models

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// The syllabus lives in type_params, which the CMS reads, changes (e.g. a cached PDF url) and writes
// back as a typed Test, so every field must survive a decode and encode.
func TestSyllabusSurvivesTestRoundTrip(t *testing.T) {
	in := `{"id":7,"type_params":{"duration":"180","syllabus":{"grade_ids":[11,12],` +
		`"subjects":[{"subject_id":2,"chapter_ids":[101,102]},{"subject_id":3,"chapter_ids":[205]}]}}}`

	var test Test
	if err := json.Unmarshal([]byte(in), &test); err != nil {
		t.Fatal(err)
	}
	want := &TestSyllabus{GradeIDs: []int8{11, 12}, Subjects: []SyllabusSubject{
		{SubjectID: 2, ChapterIDs: []int16{101, 102}}, {SubjectID: 3, ChapterIDs: []int16{205}},
	}}
	if !reflect.DeepEqual(test.TypeParams.Syllabus, want) {
		t.Fatalf("decoded %+v, want %+v", test.TypeParams.Syllabus, want)
	}

	// simulate cachePdf: change another type_params field, then write the whole test back
	test.TypeParams.PdfUrls = map[string]PdfCacheEntry{"questions": {Key: "k", Hash: "h"}}
	out, err := json.Marshal(test)
	if err != nil {
		t.Fatal(err)
	}
	var back Test
	if err := json.Unmarshal(out, &back); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(back.TypeParams.Syllabus, want) {
		t.Fatalf("syllabus lost after write-back: %s", out)
	}
}

func TestTestWithoutSyllabusStaysWithoutIt(t *testing.T) {
	var test Test
	if err := json.Unmarshal([]byte(`{"id":7,"type_params":{"duration":"180"}}`), &test); err != nil {
		t.Fatal(err)
	}
	if test.TypeParams.Syllabus != nil {
		t.Fatalf("expected no syllabus, got %+v", test.TypeParams.Syllabus)
	}
	out, _ := json.Marshal(test)
	if strings.Contains(string(out), "syllabus") {
		t.Fatalf("a test without a syllabus must not write the key: %s", out)
	}
}
