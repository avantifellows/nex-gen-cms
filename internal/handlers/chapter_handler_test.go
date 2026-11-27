package handlers

import (
	"reflect"
	"testing"
)

func TestParseSubjectIds(t *testing.T) {
	got, err := parseSubjectIds("2, 3,2,4")
	if err != nil || !reflect.DeepEqual(got, []int8{2, 3, 4}) {
		t.Fatalf("got %v, %v; want [2 3 4] without repeats", got, err)
	}

	for _, bad := range []string{"", "x", "2,,3", "0", "-1", "2,a", "300"} {
		if _, err := parseSubjectIds(bad); err == nil {
			t.Errorf("parseSubjectIds(%q) should fail", bad)
		}
	}
}

func TestParseChapterIdSetSkipsInvalidIds(t *testing.T) {
	got := parseChapterIdSet("5, 6,x,,0,-3,5,99999")
	if !reflect.DeepEqual(got, map[int16]bool{5: true, 6: true}) {
		t.Fatalf("got %v", got)
	}
	if got := parseChapterIdSet(""); len(got) != 0 {
		t.Fatalf("empty input should give an empty set, got %v", got)
	}
}
