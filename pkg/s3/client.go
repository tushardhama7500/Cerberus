package s3

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"cerberus/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Client struct {
	s3Client *s3.Client
	bucket   string
}

// New creates an S3 client using explicit credentials from config.
func New(cfg *config.Config) (*Client, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.AWS.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AWS.AccessKeyID,
			cfg.AWS.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &Client{
		s3Client: s3.NewFromConfig(awsCfg),
		bucket:   cfg.AWS.S3Bucket,
	}, nil
}

// UploadBase64 decodes a base64 string and uploads it to S3.
// Returns the public URL of the uploaded object.
//
// Why base64? The frontend can encode the file as base64 and send it through
// the GraphQL mutation body — no multipart form needed. For large files you'd
// use presigned URLs instead, but for screenshots this is acceptable.
func (c *Client) UploadBase64(ctx context.Context, requestID, fileName, fileBase64 string) (string, error) {
	// Validate file extension — only allow images
	ext := strings.ToLower(filepath.Ext(fileName))
	allowedExts := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
	}
	contentType, ok := allowedExts[ext]
	if !ok {
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}

	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(fileBase64)
	if err != nil {
		// Try without padding
		decoded, err = base64.RawStdEncoding.DecodeString(fileBase64)
		if err != nil {
			return "", fmt.Errorf("invalid base64 data: %w", err)
		}
	}

	// Max size: 5MB for screenshots
	const maxSize = 5 * 1024 * 1024
	if len(decoded) > maxSize {
		return "", fmt.Errorf("file size exceeds 5MB limit")
	}

	// Build a structured, collision-free S3 key
	// Format: screenshots/{requestID}/{timestamp}_{fileName}
	key := fmt.Sprintf("screenshots/%s/%d_%s", requestID, time.Now().UnixMilli(), fileName)

	_, err = c.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(decoded),
		ContentType: aws.String(contentType),
		// Objects are private by default — use presigned URLs for access if needed
	})
	if err != nil {
		return "", fmt.Errorf("s3 upload failed: %w", err)
	}

	// Return the S3 URL
	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", c.bucket, "ap-south-1", key)
	return url, nil
}
