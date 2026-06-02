package curriculumconfig

import (
	"context"
	"errors"
	"time"
)

var ErrNotImplemented = errors.New("curriculum config endpoint is not available in this slice")

type Repository interface {
	SchemaReadiness(ctx context.Context) (Readiness, error)
	List(ctx context.Context, query ListQuery) (ListResult, error)
	FilterOptions(ctx context.Context) (FilterOptions, error)
	ChapterOptions(ctx context.Context, query ChapterOptionsQuery) ([]ChapterOption, error)
	Impact(ctx context.Context, query ImpactQuery) (ImpactResult, error)
	Create(ctx context.Context, input CreateInput) (MutationResult, error)
	Edit(ctx context.Context, input EditInput) (MutationResult, error)
	RemoveFromSyllabus(ctx context.Context, input RemoveInput) (MutationResult, error)
	ExportRows(ctx context.Context, query ListQuery) ([]ExportRow, error)
}

type Readiness struct {
	Ready           bool
	MutationReady   bool
	Reasons         []string
	MutationReasons []string
}

type ListQuery struct {
	ExamTrack      string
	Grade          string
	Subject        string
	Search         string
	ChapterID      string
	SyllabusStatus string
	Page           int
	Limit          int
	Sort           string
	Direction      string
}

type ListResult struct {
	Rows       []ListRow
	TotalRows  int
	Page       int
	Limit      int
	TotalPages int
}

type ListRow struct {
	ID                int64
	ChapterID         int64
	ChapterCode       string
	ChapterName       string
	Grade             string
	Subject           string
	ExamTrack         string
	IsInSyllabus      bool
	PrescribedMinutes int
	PrescribedHours   string
	CoverageSequence  int
	UpdatedByEmail    string
	UpdatedAt         time.Time
	LockToken         string
}

type FilterOptions struct {
	ExamTracks []Option
	Grades     []Option
	Subjects   []Option
}

type Option struct {
	Value string
	Label string
}

type ChapterOptionsQuery struct {
	ExamTrack string
	Grade     string
	Subject   string
	Search    string
}

type ChapterOption struct {
	ChapterID           int64
	ChapterCode         string
	ChapterName         string
	Grade               string
	Subject             string
	TopicCount          int
	ExistingConfigID    *int64
	ExistingInSyllabus  *bool
	ExistingExamTrack   string
	HasZeroTopicWarning bool
	HasDuplicateConfig  bool
}

type ImpactQuery struct {
	ConfigID          int64
	ChapterID         int64
	ExamTrack         string
	IsInSyllabus      bool
	PrescribedMinutes int
	CoverageSequence  int
}

type ImpactResult struct {
	SummaryRows        int
	ActiveLogs         int
	ChapterCompletions int
	Unavailable        bool
	Warnings           []Warning
}

type CreateInput struct {
	ChapterID         int64
	ExamTrack         string
	IsInSyllabus      bool
	PrescribedMinutes int
	CoverageSequence  int
	AdminEmail        string
}

type EditInput struct {
	ID                int64
	IsInSyllabus      bool
	PrescribedMinutes int
	CoverageSequence  int
	LockToken         string
	AdminEmail        string
}

type RemoveInput struct {
	ID         int64
	LockToken  string
	AdminEmail string
}

type MutationResult struct {
	Row      *ListRow
	Warnings []Warning
	Impact   ImpactResult
}

type Warning struct {
	Code    string
	Message string
}

type ExportRow struct {
	ChapterCode       string
	ChapterName       string
	Grade             string
	Subject           string
	ExamTrack         string
	IsInSyllabus      bool
	PrescribedMinutes int
	PrescribedHours   string
	CoverageSequence  int
	UpdatedByEmail    string
	UpdatedAt         time.Time
}
