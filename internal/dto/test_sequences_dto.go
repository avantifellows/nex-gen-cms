package dto

// TestSequencesResponse mirrors the db-service response for /resources/test-sequences,
// listing the test sequence numbers already used for a given program/type_code/year.
type TestSequencesResponse struct {
	UsedSequences []int `json:"used_sequences"`
}

// TestSequenceOptionsData is passed to test_sequence_options.html to render the Sequence
// dropdown's <option> list, marking already-used sequence numbers and preserving whichever
// one is currently selected.
type TestSequenceOptionsData struct {
	Sequences     []int
	UsedSequences map[int]bool
	Selected      string
}
