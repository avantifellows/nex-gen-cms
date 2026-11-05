package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// A resource edited in the CMS is PATCHed with whatever BuildMap returns, so
// any type_params key the form does not carry has to survive the round trip.
func TestBuildMapPreservesUnknownTypeParams(t *testing.T) {
	stored := `{"id":276,"name":[{"lang_code":"en","resource":"Mathematical Tools"}],
		"type":"quiz","subtype":"Assessment",
		"type_params":{"src_link":"https://quiz.avantifellows.org/quiz/old",
		               "sashakt_link":"https://sashakt.example/old"}}`

	var resource Resource
	assert.NoError(t, json.Unmarshal([]byte(stored), &resource))
	assert.Equal(t, "https://quiz.avantifellows.org/quiz/old", resource.TypeParams.SrcLink)

	patch := resource.BuildMap("", "Mathematical Tools", "quiz", "Assessment",
		"https://quiz.avantifellows.org/quiz/new")

	encoded, err := json.Marshal(patch["type_params"])
	assert.NoError(t, err)

	var sent map[string]any
	assert.NoError(t, json.Unmarshal(encoded, &sent))
	assert.Equal(t, "https://quiz.avantifellows.org/quiz/new", sent["src_link"])
	assert.Equal(t, "https://sashakt.example/old", sent["sashakt_link"])
}

func TestBuildMapWithoutUnknownTypeParams(t *testing.T) {
	var resource Resource
	assert.NoError(t, json.Unmarshal(
		[]byte(`{"id":2,"type":"document","type_params":{"src_link":"https://drive.example/a"}}`),
		&resource))

	patch := resource.BuildMap("", "Mathematical Tools", "document", "Module",
		"https://drive.example/b")

	encoded, err := json.Marshal(patch["type_params"])
	assert.NoError(t, err)
	assert.JSONEq(t, `{"src_link":"https://drive.example/b"}`, string(encoded))
}

// NewResource builds a fresh row, so it carries only what the form supplied.
func TestNewResourceTypeParams(t *testing.T) {
	resource := NewResource("11P1", "Mathematical Tools", "quiz", "Assessment",
		"https://quiz.avantifellows.org/quiz/new", 1, 1, 3, 4)

	encoded, err := json.Marshal(resource.TypeParams)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"src_link":"https://quiz.avantifellows.org/quiz/new"}`, string(encoded))
}
