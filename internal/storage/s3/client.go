package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Client provides S3-compatible storage operations
type Client struct {
	client     *s3.Client
	bucket     string
	prefix     string
}

// Config contains S3 configuration
type Config struct {
	Endpoint   string
	Region     string
	Bucket     string
	Prefix     string
	AccessKey  string
	SecretKey  string
}

// NewClient creates a new S3 client
func NewClient(cfg Config) (*Client, error) {
	var opts []func(*config.LoadOptions) error

	if cfg.Region != "" {
		opts = append(opts, config.WithRegion(cfg.Region))
	} else {
		opts = append(opts, config.WithRegion("us-east-1"))
	}

	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		))
	}

	awsCfg, err := config.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	s3Opts := []func(*s3.Options){}
	if cfg.Endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true
		})
	}

	client := s3.NewFromConfig(awsCfg, s3Opts...)

	return &Client{
		client: client,
		bucket: cfg.Bucket,
		prefix: cfg.Prefix,
	}, nil
}

// Put uploads an object
func (c *Client) Put(ctx context.Context, key string, data []byte) error {
	fullKey := c.fullKey(key)

	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(fullKey),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("putting object: %w", err)
	}

	return nil
}

// Get downloads an object
func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
	fullKey := c.fullKey(key)

	result, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(fullKey),
	})
	if err != nil {
		return nil, fmt.Errorf("getting object: %w", err)
	}
	defer result.Body.Close()

	return io.ReadAll(result.Body)
}

// Delete deletes an object
func (c *Client) Delete(ctx context.Context, key string) error {
	fullKey := c.fullKey(key)

	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(fullKey),
	})
	if err != nil {
		return fmt.Errorf("deleting object: %w", err)
	}

	return nil
}

// Exists checks if an object exists
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := c.fullKey(key)

	_, err := c.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(fullKey),
	})
	if err != nil {
		return false, nil
	}

	return true, nil
}

// List lists objects with a prefix
func (c *Client) List(ctx context.Context, prefix string) ([]string, error) {
	fullPrefix := c.fullKey(prefix)

	result, err := c.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucket),
		Prefix: aws.String(fullPrefix),
	})
	if err != nil {
		return nil, fmt.Errorf("listing objects: %w", err)
	}

	var keys []string
	for _, obj := range result.Contents {
		key := aws.ToString(obj.Key)
		if c.prefix != "" {
			key = key[len(c.prefix)+1:]
		}
		keys = append(keys, key)
	}

	return keys, nil
}

// GetPresignedURL generates a presigned URL for downloading
func (c *Client) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	fullKey := c.fullKey(key)

	presignClient := s3.NewPresignClient(c.client)
	result, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(fullKey),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})
	if err != nil {
		return "", fmt.Errorf("presigning URL: %w", err)
	}

	return result.URL, nil
}

func (c *Client) fullKey(key string) string {
	if c.prefix != "" {
		return filepath.Join(c.prefix, key)
	}
	return key
}
