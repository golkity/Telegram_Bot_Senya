package s3

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"telegram_bot/pkg/metrics"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Client struct {
	s3Client      *s3.Client
	presignClient *s3.PresignClient
	bucket        string
}

func New(ctx context.Context, endpoint, accessKey, secretKey, bucket string) (*Client, error) {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			PartitionID:   "aws",
			URL:           endpoint,
			SigningRegion: "ru-central1",
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("ru-central1"),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load s3 config: %w", err)
	}

	client := s3.NewFromConfig(cfg)
	presignClient := s3.NewPresignClient(client)

	return &Client{
		s3Client:      client,
		presignClient: presignClient,
		bucket:        bucket,
	}, nil
}

func (c *Client) UploadFile(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) error {
	start := time.Now()

	_, err := c.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(c.bucket),
		Key:           aws.String(objectName),
		Body:          reader,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(contentType),
	})

	metrics.S3OperationDuration.WithLabelValues("upload").Observe(time.Since(start).Seconds())

	if err != nil {
		metrics.S3OperationsTotal.WithLabelValues("upload", "error").Inc()
		return fmt.Errorf("upload to s3 failed: %w", err)
	}

	metrics.S3OperationsTotal.WithLabelValues("upload", "success").Inc()
	return nil
}

func (c *Client) GetPresignedURL(ctx context.Context, objectName string, lifetime time.Duration) (string, error) {
	start := time.Now()

	request, err := c.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(objectName),
	}, s3.WithPresignExpires(lifetime))

	metrics.S3OperationDuration.WithLabelValues("presign").Observe(time.Since(start).Seconds())

	if err != nil {
		metrics.S3OperationsTotal.WithLabelValues("presign", "error").Inc()
		return "", fmt.Errorf("failed to sign url: %w", err)
	}

	metrics.S3OperationsTotal.WithLabelValues("presign", "success").Inc()
	return request.URL, nil
}

func (c *Client) DownloadFile(ctx context.Context, objectName string) (io.ReadCloser, error) {
	start := time.Now()

	out, err := c.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(objectName),
	})

	metrics.S3OperationDuration.WithLabelValues("download").Observe(time.Since(start).Seconds())

	if err != nil {
		metrics.S3OperationsTotal.WithLabelValues("download", "error").Inc()
		return nil, fmt.Errorf("failed to download file from S3: %w", err)
	}

	metrics.S3OperationsTotal.WithLabelValues("download", "success").Inc()
	return out.Body, nil
}

func (c *Client) DeleteFile(ctx context.Context, objectName string) error {
	start := time.Now()

	_, err := c.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(objectName),
	})

	metrics.S3OperationDuration.WithLabelValues("delete").Observe(time.Since(start).Seconds())

	if err != nil {
		metrics.S3OperationsTotal.WithLabelValues("delete", "error").Inc()
		return err
	}

	metrics.S3OperationsTotal.WithLabelValues("delete", "success").Inc()
	return nil
}

func (c *Client) MoveFile(ctx context.Context, oldKey, newKey string) error {
	start := time.Now()

	source := fmt.Sprintf("%s/%s", c.bucket, oldKey)
	u := &url.URL{Path: source}
	sourceEncoded := u.String()

	_, err := c.s3Client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(c.bucket),
		CopySource: aws.String(sourceEncoded),
		Key:        aws.String(newKey),
	})
	if err != nil {
		metrics.S3OperationDuration.WithLabelValues("move").Observe(time.Since(start).Seconds())
		metrics.S3OperationsTotal.WithLabelValues("move", "error").Inc()
		return fmt.Errorf("copy failed: %w", err)
	}

	_, err = c.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(oldKey),
	})

	metrics.S3OperationDuration.WithLabelValues("move").Observe(time.Since(start).Seconds())

	if err != nil {
		metrics.S3OperationsTotal.WithLabelValues("move", "error").Inc()
		return err
	}

	metrics.S3OperationsTotal.WithLabelValues("move", "success").Inc()
	return nil
}
