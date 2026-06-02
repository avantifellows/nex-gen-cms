-- Test fixture for Curriculum Config readiness checks.
-- Mirrors db-service migration 20260530120000_create_lms_curriculum_tracking_tables.exs
-- and should be verified against deployed db-service migrations during QA.

CREATE TABLE public.grade (
    id bigserial PRIMARY KEY,
    number integer NOT NULL
);

CREATE TABLE public.subject (
    id bigserial PRIMARY KEY,
    code character varying(255) NOT NULL,
    name jsonb NOT NULL
);

CREATE TABLE public.chapter (
    id bigserial PRIMARY KEY,
    code character varying(255) NOT NULL,
    name jsonb NOT NULL,
    grade_id bigint NOT NULL,
    subject_id bigint NOT NULL
);

CREATE TABLE public.topic (
    id bigserial PRIMARY KEY,
    chapter_id bigint NOT NULL
);

CREATE TABLE public.program (
    id bigserial PRIMARY KEY,
    name character varying(255) NOT NULL
);

CREATE TABLE public.school (
    id bigserial PRIMARY KEY,
    code character varying(255) NOT NULL,
    program_ids integer[] NOT NULL
);

CREATE TABLE public.lms_chapter_exam_configs (
    id bigserial PRIMARY KEY,
    chapter_id bigint NOT NULL,
    exam_track character varying(32) NOT NULL,
    is_in_syllabus boolean NOT NULL DEFAULT true,
    prescribed_minutes integer NOT NULL DEFAULT 0,
    coverage_sequence integer NOT NULL,
    inserted_by_email character varying(255),
    updated_by_email character varying(255),
    inserted_at timestamp without time zone NOT NULL DEFAULT now(),
    updated_at timestamp without time zone NOT NULL DEFAULT now(),
    CONSTRAINT lms_chapter_exam_configs_exam_track_check
        CHECK (exam_track IN ('jee_main', 'jee_advanced', 'neet')),
    CONSTRAINT lms_chapter_exam_configs_prescribed_minutes_check
        CHECK (prescribed_minutes >= 0),
    CONSTRAINT lms_chapter_exam_configs_coverage_sequence_check
        CHECK (coverage_sequence > 0),
    CONSTRAINT lms_chapter_exam_configs_out_of_syllabus_minutes_check
        CHECK (is_in_syllabus OR prescribed_minutes = 0),
    CONSTRAINT lms_chapter_exam_configs_chapter_track_unique
        UNIQUE (chapter_id, exam_track)
);

CREATE INDEX lms_chapter_exam_configs_exam_track_chapter_id_index
    ON public.lms_chapter_exam_configs (exam_track, chapter_id);

CREATE TABLE public.lms_curriculum_logs (
    id bigserial PRIMARY KEY,
    school_code character varying(255) NOT NULL,
    program_id bigint NOT NULL,
    grade_id bigint NOT NULL,
    subject_id bigint NOT NULL,
    exam_track character varying(32) NOT NULL,
    log_date date NOT NULL,
    duration_minutes integer NOT NULL,
    deleted_at timestamp without time zone,
    CONSTRAINT lms_curriculum_logs_exam_track_check
        CHECK (exam_track IN ('jee_main', 'jee_advanced', 'neet')),
    CONSTRAINT lms_curriculum_logs_duration_minutes_check
        CHECK (duration_minutes > 0 AND duration_minutes <= 720)
);

CREATE INDEX lms_curriculum_logs_active_scope_index
    ON public.lms_curriculum_logs (school_code, program_id, grade_id, subject_id, exam_track)
    WHERE deleted_at IS NULL;
CREATE INDEX lms_curriculum_logs_active_scope_date_index
    ON public.lms_curriculum_logs (school_code, program_id, grade_id, subject_id, exam_track, log_date)
    WHERE deleted_at IS NULL;
CREATE INDEX lms_curriculum_logs_log_date_index
    ON public.lms_curriculum_logs (log_date);

CREATE TABLE public.lms_curriculum_log_topics (
    id bigserial PRIMARY KEY,
    curriculum_log_id bigint NOT NULL,
    topic_id bigint NOT NULL,
    CONSTRAINT lms_curriculum_log_topics_log_topic_unique
        UNIQUE (curriculum_log_id, topic_id)
);

CREATE INDEX lms_curriculum_log_topics_log_id_index
    ON public.lms_curriculum_log_topics (curriculum_log_id);
CREATE INDEX lms_curriculum_log_topics_topic_id_index
    ON public.lms_curriculum_log_topics (topic_id);

CREATE TABLE public.lms_curriculum_chapter_completions (
    id bigserial PRIMARY KEY,
    school_code character varying(255) NOT NULL,
    program_id bigint NOT NULL,
    chapter_id bigint NOT NULL,
    exam_track character varying(32) NOT NULL,
    deleted_at timestamp without time zone,
    CONSTRAINT lms_curriculum_chapter_completions_exam_track_check
        CHECK (exam_track IN ('jee_main', 'jee_advanced', 'neet'))
);

CREATE UNIQUE INDEX lms_curriculum_chapter_completions_active_unique
    ON public.lms_curriculum_chapter_completions (school_code, program_id, chapter_id, exam_track)
    WHERE deleted_at IS NULL;
CREATE INDEX lms_curriculum_chapter_completions_scope_index
    ON public.lms_curriculum_chapter_completions (school_code, program_id, chapter_id, exam_track);
