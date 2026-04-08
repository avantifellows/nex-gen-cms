package models

import "strings"

type Resource struct {
	ID         int                `json:"id,omitempty"`
	Name       []ResName          `json:"name"`
	Code       string             `json:"code"`
	Type       string             `json:"type"`
	Subtype    string             `json:"subtype,omitempty"`
	URL        string             `json:"url,omitempty"`
	TypeParams ResourceTypeParams `json:"type_params,omitempty"`
}

type ResourceTypeParams struct {
	SrcLink string `json:"src_link,omitempty"`
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
