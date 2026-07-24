package models

import "html/template"

type Problem struct {
	ID              int               `json:"id,omitempty"`
	Code            string            `json:"code,omitempty"`
	Type            string            `json:"type"`
	Subtype         string            `json:"subtype"`
	Paragraph       *ProblemParagraph `json:"paragraph,omitempty"`
	TypeParams      ProbTypeParams    `json:"type_params"`
	MetaData        ProbMetaData      `json:"meta_data"`
	LangVersions    []LangVersion     `json:"lang_versions"`
	SkillIDs        []int16           `json:"skill_ids"`
	Skills          []Skill
	CurriculumID    int16 `json:"curriculum_id"` // used with only get call
	GradeID         int8  `json:"grade_id"`      // used with only get call
	SubjectID       int8  `json:"subject_id"`
	Subject         Subject
	TopicID         int16         `json:"topic_id"`
	ChapterID       int16         `json:"chapter_id"`
	ChapterName     []ChapterLang `json:"chapter_name"` // used with only get call
	Concepts        []Concept     `json:"concepts"`     // used with only get call
	DifficultyLevel string        `json:"difficulty_level"`
	TagIDs          []int         `json:"tag_ids"`
	TagNames        []string
	StatusID        int8 `json:"cms_status_id"`
}

type LangVersion struct {
	LangCode string       `json:"lang_code"`
	MetaData ProbMetaData `json:"meta_data"`
}

type ProbTypeParams struct {
	TestIds []int `json:"test_ids"`
}

type ProbMetaData struct {
	Question  template.HTML   `json:"text"`
	Options   []template.HTML `json:"options"`
	Answers   []string        `json:"answer"`
	Solutions []Solution      `json:"solutions"`
}

type Solution struct {
	Type  string        `json:"type"`
	Value template.HTML `json:"value"`
}

type ProblemParagraph struct {
	ID   int           `json:"id"`
	Body template.HTML `json:"body"`
}

func (p *Problem) GetLangVersion(langCode string) *LangVersion {
	for i := range p.LangVersions {
		if p.LangVersions[i].LangCode == langCode {
			return &p.LangVersions[i]
		}
	}
	return nil
}

func (p Problem) DisplayDifficulty() int8 {
	switch p.DifficultyLevel {
	case "hard":
		return 3
	case "medium":
		return 2
	default:
		return 1
	}
}

func (p *Problem) GetChapterNameByLang(langCode string) string {
	for _, chapterLang := range p.ChapterName {
		if chapterLang.LangCode == langCode {
			return chapterLang.ChapterName
		}
	}
	return ""
}

// CopyTo returns a clone of p cleared for create in the destination topic.
func (p *Problem) CopyTo(topic *Topic, curriculumID int16, gradeID int8, subjectID int8) Problem {
	copied := *p
	copied.ID = 0
	copied.Code = ""
	copied.TopicID = topic.ID
	copied.ChapterID = topic.ChapterID
	copied.CurriculumID = curriculumID
	copied.GradeID = gradeID
	copied.SubjectID = subjectID

	if copied.Paragraph != nil {
		paragraph := *copied.Paragraph
		paragraph.ID = 0
		copied.Paragraph = &paragraph
	}

	return copied
}
