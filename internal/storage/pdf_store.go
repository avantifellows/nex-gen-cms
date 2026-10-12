// Package storage caches generated test PDFs in S3 so they aren't re-rendered
// via headless Chrome on every download once nothing about their content has
// changed. See internal/handlers/test_handler.go (DownloadPdf) for the caller.
package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/avantifellows/nex-gen-cms/config"
)

// PresignExpiry bounds how long a redirect URL we hand out for a cache hit
// stays valid. Long enough to cover slow clients opening the link, short
// enough that a leaked URL doesn't grant standing access to the bucket.
const PresignExpiry = 5 * time.Minute

// PdfStore reads/writes cached PDFs in the "test PDFs" S3 bucket (provisioned
// in terraform/s3.tf). A nil *PdfStore means caching is disabled (no
// AWS_S3_BUCKET configured, e.g. local dev without AWS credentials) — callers
// must treat that as "always regenerate," never as an error.
type PdfStore struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	bucket        string
}

// NewPdfStore builds a PdfStore from AWS_S3_BUCKET/AWS_REGION. It returns a
// nil PdfStore (not an error) when AWS_S3_BUCKET is unset, so the app still
// runs without AWS credentials — PDFs are just always regenerated in that case.
func NewPdfStore(ctx context.Context) (*PdfStore, error) {
	bucket := config.GetEnv("AWS_S3_BUCKET", "")
	if bucket == "" {
		return nil, nil
	}

	region := config.GetEnv("AWS_REGION", "ap-south-1")
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	client := s3.NewFromConfig(cfg)
	return &PdfStore{
		client:        client,
		presignClient: s3.NewPresignClient(client),
		bucket:        bucket,
	}, nil
}

// Exists reports whether key is already cached in the bucket.
func (s *PdfStore) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	if err == nil {
		return true, nil
	}

	var notFound *types.NotFound
	if errors.As(err, &notFound) {
		return false, nil
	}
	return false, err
}

// Put uploads pdfData under key, overwriting any existing object there.
func (s *PdfStore) Put(ctx context.Context, key string, pdfData []byte) error {
	contentType := "application/pdf"
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &s.bucket,
		Key:         &key,
		Body:        bytes.NewReader(pdfData),
		ContentType: &contentType,
	})
	return err
}

// PresignGetURL returns a short-lived signed URL for key that makes S3 send
// the object back with the given download filename, so the browser sees the
// same Content-Disposition it would get from a freshly generated PDF.
func (s *PdfStore) PresignGetURL(ctx context.Context, key, filename string) (string, error) {
	disposition := fmt.Sprintf(`attachment; filename=%q`, filename)
	req, err := s.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     &s.bucket,
		Key:                        &key,
		ResponseContentDisposition: &disposition,
	}, s3.WithPresignExpires(PresignExpiry))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}
