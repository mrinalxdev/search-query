package minio

import (
	"context"
	"fmt"
	"io"
	"mulit-minIO/internal/config"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Client struct {
	*minio.Client
	config *config.MinIOConfig
}

type UploadResult struct {
	ObjectName string
	Size       int64
	ETag       string
	Location   string
}

type PresignedURLOptions struct {
	Expiry          time.Duration
	RequestMethod   string
}

func NewClient(cfg *config.MinIOConfig) (*Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	return &Client{
		Client: client,
		config: cfg,
	}, nil
}

func (c *Client) EnsureBucket(ctx context.Context) error {
	exists, err := c.BucketExists(ctx, c.config.Bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket: %w", err)
	}

	if !exists {
		err = c.MakeBucket(ctx, c.config.Bucket, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return nil
}

func (c *Client) UploadFile(ctx context.Context, objectName string, reader io.Reader, 
	objectSize int64, contentType string) (*UploadResult, error) {
	
	info, err := c.PutObject(ctx, c.config.Bucket, objectName, reader, objectSize, 
		minio.PutObjectOptions{
			ContentType: contentType,
		})
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	return &UploadResult{
		ObjectName: info.Key,
		Size:       info.Size,
		ETag:       info.ETag,
		Location:   info.Location,
	}, nil
}

func (c *Client) MultipartUpload(ctx context.Context, objectName string, 
	reader io.Reader, objectSize int64, contentType string, 
	partSize int64) (*UploadResult, error) {
	

	var partSizeUint uint64
	if partSize > 0 {
		partSizeUint = uint64(partSize)
	} else {
		partSizeUint = 5 * 1024 * 1024 
	}

	info, err := c.PutObject(ctx, c.config.Bucket, objectName, reader, objectSize,
		minio.PutObjectOptions{
			ContentType: contentType,
			NumThreads:  4,
			PartSize:    partSizeUint, 
		})
	if err != nil {
		return nil, fmt.Errorf("failed to multipart upload: %w", err)
	}

	return &UploadResult{
		ObjectName: info.Key,
		Size:       info.Size,
		ETag:       info.ETag,
		Location:   info.Location,
	}, nil
}

func (c *Client) GeneratePresignedURL(ctx context.Context, objectName string, 
	opts PresignedURLOptions) (string, error) {
	
	if opts.Expiry == 0 {
		opts.Expiry = 15 * time.Minute
	}
	
	if opts.RequestMethod == "" {
		opts.RequestMethod = "GET"
	}

	presignedURL, err := c.PresignedGetObject(ctx, c.config.Bucket, objectName, 
		opts.Expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignedURL.String(), nil
}

func (c *Client) GeneratePresignedPutURL(ctx context.Context, objectName string, 
	expiry time.Duration) (string, error) {
	
	if expiry == 0 {
		expiry = 15 * time.Minute
	}

	presignedURL, err := c.PresignedPutObject(ctx, c.config.Bucket, objectName, expiry)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned PUT URL: %w", err)
	}

	return presignedURL.String(), nil
}

func (c *Client) DownloadFile(ctx context.Context, objectName string) (io.ReadCloser, error) {
	object, err := c.GetObject(ctx, c.config.Bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	return object, nil
}

func (c *Client) DeleteFile(ctx context.Context, objectName string) error {
	err := c.RemoveObject(ctx, c.config.Bucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

func (c *Client) ListFiles(ctx context.Context, prefix string, recursive bool) ([]minio.ObjectInfo, error) {
	var objects []minio.ObjectInfo
	
	for object := range c.ListObjects(ctx, c.config.Bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: recursive,
	}) {
		if object.Err != nil {
			return nil, object.Err
		}
		objects = append(objects, object)
	}

	return objects, nil
}

func (c *Client) GetFileInfo(ctx context.Context, objectName string) (*minio.ObjectInfo, error) {
	info, err := c.StatObject(ctx, c.config.Bucket, objectName, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return &info, nil
}