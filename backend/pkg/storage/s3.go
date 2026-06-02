package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	appconfig "github.com/dreamlog/backend/internal/config"
)

// Client is an S3-compatible storage client (R2, MinIO, or AWS S3).
type Client struct {
	s3     *s3.Client
	pre    *s3.PresignClient // signs with internal endpoint (for server-side ops)
	prePub *s3.PresignClient // signs with public endpoint (for client upload URLs)
	bucket string
	expiry time.Duration
}

// New constructs a storage client from app config.
func New(cfg *appconfig.StorageConfig) (*Client, error) {
	internalClient, err := newS3Client(cfg.Endpoint, cfg)
	if err != nil {
		return nil, err
	}

	// If a public base URL is configured, create a separate presign client whose
	// URLs are signed for the public host so mobile clients can reach MinIO directly.
	pubPresign := s3.NewPresignClient(internalClient)
	if cfg.PublicBaseURL != "" {
		pubClient, err := newS3Client(cfg.PublicBaseURL, cfg)
		if err != nil {
			return nil, err
		}
		pubPresign = s3.NewPresignClient(pubClient)
	}

	return &Client{
		s3:     internalClient,
		pre:    s3.NewPresignClient(internalClient),
		prePub: pubPresign,
		bucket: cfg.Bucket,
		expiry: cfg.PresignExpiry,
	}, nil
}

func newS3Client(endpoint string, cfg *appconfig.StorageConfig) (*s3.Client, error) {
	resolver := aws.EndpointResolverWithOptionsFunc(
		func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               endpoint,
				HostnameImmutable: true,
				SigningRegion:     cfg.Region,
			}, nil
		},
	)

	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
		config.WithEndpointResolverWithOptions(resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("storage: load aws config: %w", err)
	}

	return s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.UsePathStyle
	}), nil
}

// PresignUpload generates a PUT pre-signed URL for direct client upload.
// The URL is signed with the public endpoint so mobile clients can reach it.
func (c *Client) PresignUpload(ctx context.Context, key string) (string, error) {
	req, err := c.prePub.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(c.expiry))
	if err != nil {
		return "", fmt.Errorf("storage: presign upload %q: %w", key, err)
	}
	return req.URL, nil
}

// PresignDownload generates a GET pre-signed URL for temporary file access.
func (c *Client) PresignDownload(ctx context.Context, key string, expiry time.Duration) (string, error) {
	req, err := c.pre.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("storage: presign download %q: %w", key, err)
	}
	return req.URL, nil
}

// Download streams an object from storage. Caller must close the returned ReadCloser.
func (c *Client) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("storage: download %q: %w", key, err)
	}
	return out.Body, nil
}

// Upload reads body into memory and puts it to storage.
// The AWS SDK v2 requires a seekable reader to compute the payload hash,
// so we buffer the full body before calling PutObject.
func (c *Client) Upload(ctx context.Context, key, contentType string, body io.Reader) error {
	buf, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("storage: read upload body: %w", err)
	}
	_, err = c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(c.bucket),
		Key:           aws.String(key),
		ContentType:   aws.String(contentType),
		Body:          bytes.NewReader(buf),
		ContentLength: aws.Int64(int64(len(buf))),
	})
	if err != nil {
		return fmt.Errorf("storage: upload %q: %w", key, err)
	}
	return nil
}

// Delete removes an object. Returns nil if the object does not exist.
func (c *Client) Delete(ctx context.Context, key string) error {
	_, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("storage: delete %q: %w", key, err)
	}
	return nil
}

// Exists returns true if the key exists in the bucket.
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	_, err := c.s3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return false, nil
	}
	return true, nil
}
