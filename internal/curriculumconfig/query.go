package curriculumconfig

import (
	"strconv"
	"strings"
)

const (
	DefaultExamTrack      = "jee_main"
	DefaultSyllabusStatus = "in_syllabus"
	DefaultPage           = 1
	DefaultLimit          = 50
	DefaultSort           = "curriculum"
	DefaultDirection      = "asc"
)

var AllowedPageSizes = []int{10, 20, 50, 100}

func NormalizeListQuery(query ListQuery) ListQuery {
	query.ExamTrack = normalizeChoice(query.ExamTrack, DefaultExamTrack, map[string]struct{}{
		"jee_main":     {},
		"jee_advanced": {},
		"neet":         {},
	})
	query.SyllabusStatus = normalizeChoice(query.SyllabusStatus, DefaultSyllabusStatus, map[string]struct{}{
		"in_syllabus":     {},
		"out_of_syllabus": {},
		"all":             {},
	})
	if query.Page < 1 {
		query.Page = DefaultPage
	}
	switch query.Limit {
	case 10, 20, 50, 100:
	default:
		query.Limit = DefaultLimit
	}
	query.Sort = normalizeChoice(query.Sort, DefaultSort, map[string]struct{}{
		"curriculum":        {},
		"exam_track":        {},
		"grade":             {},
		"subject":           {},
		"coverage_sequence": {},
		"chapter_code":      {},
		"chapter_name":      {},
		"updated_at":        {},
	})
	if strings.EqualFold(query.Direction, "desc") {
		query.Direction = "desc"
	} else {
		query.Direction = DefaultDirection
	}
	query.Grade = normalizePositiveIDText(query.Grade)
	query.Subject = strings.TrimSpace(query.Subject)
	if strings.EqualFold(query.Subject, "all") {
		query.Subject = ""
	}
	query.Search = strings.TrimSpace(query.Search)
	query.ChapterID = normalizePositiveIDText(query.ChapterID)
	return query
}

func normalizeChoice(value, fallback string, allowed map[string]struct{}) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if _, ok := allowed[value]; ok {
		return value
	}
	return fallback
}

func normalizePositiveIDText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "all") {
		return ""
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 1 {
		return ""
	}
	return value
}
