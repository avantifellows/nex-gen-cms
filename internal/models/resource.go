package models

type Resource struct {
	ID         int                `json:"id,omitempty"`
	Name       []ResName          `json:"name"`
	Code       string             `json:"code"`
	Type       string             `json:"type"`
	Subtype    string             `json:"subtype"`
	URL        string             `json:"url,omitempty"`
	TypeParams ResourceTypeParams `json:"type_params,omitempty"`
}

type ResourceTypeParams struct {
	SrcLink string `json:"src_link,omitempty"`
}

func (r *Resource) GetNameByLang(langCode string) string {
	for _, resourceLang := range r.Name {
		if resourceLang.LangCode == langCode {
			return resourceLang.Resource
		}
	}
	return ""
}
