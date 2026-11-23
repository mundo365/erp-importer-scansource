package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	"github.com/mundo365/erp-importer-scansource/internal/domain"
)

type Store struct {
	client *s3.Client
}

func NewStore(endpoint, region, accessKey, secretKey string) (*Store, error) {
	region = strings.TrimSpace(region)
	if region == "" {
		region = "us-east-1"
	}
	cfg := aws.Config{
		Region: region,
		Credentials: credentials.NewStaticCredentialsProvider(
			strings.TrimSpace(accessKey),
			strings.TrimSpace(secretKey),
			"",
		),
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint = strings.TrimSpace(endpoint); endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		}
	})
	return &Store{client: client}, nil
}

func (s *Store) Download(ctx context.Context, bucket, key string) ([]byte, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFound(err) {
			return nil, domain.Permanent(domain.StageParse, "blob_not_found", "objeto nao encontrado no S3")
		}
		return nil, fmt.Errorf("s3 get %s/%s: %w", bucket, key, err)
	}
	defer out.Body.Close()
	return io.ReadAll(out.Body)
}

func isNotFound(err error) bool {
	var nsk *types.NoSuchKey
	if errors.As(err, &nsk) {
		return true
	}
	var nf *types.NotFound
	if errors.As(err, &nf) {
		return true
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NoSuchKey", "NotFound", "NoSuchBucket":
			return true
		}
	}
	return false
}
