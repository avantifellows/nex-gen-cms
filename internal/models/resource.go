package models

import (
	"encoding/json"
	"strings"
)

type Resource struct {
	ID               int                `json:"id,omitempty"`
	Name             []ResName          `json:"name"`
	Code             string             `json:"code"`
	Type             string             `json:"type"`
	Subtype          string             `json:"subtype,omitempty"`
	StatusID         int8               `json:"cms_status_id,omitempty"`
	ChapterID        int16              `json:"chapter_id,omitempty"`
	TopicID          int16              `json:"topic_id,omitempty"`
	SubjectID        int8               `json:"subject_id,omitempty"`
	CurriculumGrades []CurriculumGrade  `json:"curriculum_grades,omitempty"`
	TypeParams       ResourceTypeParams `json:"type_params,omitempty"`
}

type ResourceTypeParams struct {
	SrcLink string `json:"src_link,omitempty"`
	// Keys the CMS does not model, such as sashakt_link. They are carried
	// through a round trip so that editing a resource here does not drop
	// whatever else is stored in type_params.
	Extra map[string]any `json:"-"`
}

func (p *ResourceTypeParams) UnmarshalJSON(data []byte) error {
	var all map[string]any
	if err := json.Unmarshal(data, &all); err != nil {
		return err
	}

	if srcLink, ok := all["src_link"].(string); ok {
		p.SrcLink = srcLink
	}
	delete(all, "src_link")

	if len(all) > 0 {
		p.Extra = all
	}
	return nil
}

func (p ResourceTypeParams) MarshalJSON() ([]byte, error) {
	all := make(map[string]any, len(p.Extra)+1)
	for key, value := range p.Extra {
		all[key] = value
	}
	if strings.TrimSpace(p.SrcLink) != "" {
		all["src_link"] = p.SrcLink
	}
	return json.Marshal(all)
}

func NewResource(code string, name string, resourceType string, subtype string, srcLink string, chapterID int16, curriculumID int16, gradeID int8, subjectID int8) *Resource {
	resource := &Resource{
		Code:      code,
		Name:      []ResName{{LangCode: "en", Resource: name}},
		Type:      resourceType,
		ChapterID: chapterID,
		SubjectID: subjectID,
		CurriculumGrades: []CurriculumGrade{
			{
				CurriculumID: curriculumID,
				GradeID:      gradeID,
			},
		},
	}

	if strings.TrimSpace(subtype) != "" {
		resource.Subtype = subtype
	}

	if strings.TrimSpace(srcLink) != "" {
		resource.TypeParams = ResourceTypeParams{
			SrcLink: srcLink,
		}
	}

	return resource
}

func (resourcePtr *Resource) BuildMap(code string, name string, resourceType string, subtype string, srcLink string) map[string]any {
	resourceMap := map[string]any{
		"code": code,
		"name": []ResName{{Resource: name, LangCode: "en"}},
		"type": resourceType,
	}

	if strings.TrimSpace(subtype) != "" {
		resourceMap["subtype"] = subtype
	}

	if strings.TrimSpace(srcLink) != "" {
		typeParams := resourcePtr.TypeParams
		typeParams.SrcLink = srcLink
		resourceMap["type_params"] = typeParams
	}

	return resourceMap
}

func (r *Resource) GetNameByLang(langCode string) string {
	for _, resourceLang := range r.Name {
		if resourceLang.LangCode == langCode {
			return resourceLang.Resource
		}
	}
	return ""
}
