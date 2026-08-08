package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

// S3 stores cache artifacts in an S3-compatible bucket (AWS S3, Cloudflare
// R2, MinIO, ...). This is the recommended backend for CI runners: it
// survives ephemeral hosts and scales to multiple server replicas behind a
// load balancer.
type S3 struct {
	client *s3.Client
	bucket string
	prefix string
}

type S3Options struct {
	Bucket       string
	Region       string
	Prefix       string
	Endpoint     string // optional: non-AWS S3-compatible endpoint
	UsePathStyle bool
}

func NewS3(ctx context.Context, opts S3Options) (*S3, error) {
	loadOpts := []func(*awsconfig.LoadOptions) error{}
	if opts.Region != "" {
		loadOpts = append(loadOpts, awsconfig.WithRegion(opts.Region))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if opts.Endpoint != "" {
			o.BaseEndpoint = aws.String(opts.Endpoint)
		}
		o.UsePathStyle = opts.UsePathStyle
	})

	return &S3{client: client, bucket: opts.Bucket, prefix: opts.Prefix}, nil
}

func (s *S3) key(hash string) string {
	if s.prefix == "" {
		return hash
	}
	return path.Join(s.prefix, hash)
}

func (s *S3) Exists(ctx context.Context, hash string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.key(hash)),
	})
	if err == nil {
		return true, nil
	}
	if isNotFound(err) {
		return false, nil
	}
	return false, err
}

func (s *S3) Put(ctx context.Context, hash string, r io.Reader, size int64) error {
	exists, err := s.Exists(ctx, hash)
	if err != nil {
		return err
	}
	if exists {
		return ErrAlreadyExists
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(s.key(hash)),
		Body:          io.LimitReader(r, size),
		ContentLength: aws.Int64(size),
	})
	if err != nil {
		return fmt.Errorf("put object: %w", err)
	}
	return nil
}

func (s *S3) Get(ctx context.Context, hash string) (io.ReadCloser, int64, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.key(hash)),
	})
	if isNotFound(err) {
		return nil, 0, ErrNotFound
	}
	if err != nil {
		return nil, 0, fmt.Errorf("get object: %w", err)
	}
	size := int64(0)
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	return out.Body, size, nil
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NotFound", "NoSuchKey":
			return true
		}
	}
	return false
}
