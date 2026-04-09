package models

import "strings"

type Resource struct {
	ID               int                `json:"id,omitempty"`
	Name             []ResName          `json:"name"`
	Code             string             `json:"code"`
	Type             string             `json:"type"`
	Subtype          string             `json:"subtype,omitempty"`
	StatusID         int8               `json:"cms_status_id,omitempty"`
	ChapterID        int16              `json:"chapter_id,omitempty"`
	TopicID          int16              `json:"topic_id,omitempty"`
	CurriculumGrades []CurriculumGrade  `json:"curriculum_grades,omitempty"`
	TypeParams       ResourceTypeParams `json:"type_params,omitempty"`
}

type ResourceTypeParams struct {
	SrcLink string `json:"src_link,omitempty"`
}

func NewResource(code string, name string, resourceType string, subtype string, srcLink string, chapterID int16, curriculumID int16, gradeID int8) *Resource {
	resource := &Resource{
		Code:      code,
		Name:      []ResName{{LangCode: "en", Resource: name}},
		Type:      resourceType,
		ChapterID: chapterID,
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
		resourceMap["type_params"] = ResourceTypeParams{
			SrcLink: srcLink,
		}
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
