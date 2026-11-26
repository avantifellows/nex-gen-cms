package models

type Curriculum struct {
	ID   int16  `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

// TEMPORARY curriculum -> syllabus subjects mapping (English subject names, in tab order) until
// the real mapping is shared. Keys are curriculum names, since ids differ between environments.
var syllabusSubjectNamesByCurriculum = map[string][]string{
	"JEE Mains": {"Maths", "Physics", "Chemistry"},
	"NEET":      {"Physics", "Chemistry", "Biology"},
}

// SyllabusSubjectNames returns the English names of the subjects a test of this curriculum can
// have in its syllabus, in display order. Empty if the curriculum has no mapping.
func (c Curriculum) SyllabusSubjectNames() []string {
	return syllabusSubjectNamesByCurriculum[c.Name]
}
