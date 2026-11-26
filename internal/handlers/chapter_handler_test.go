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
