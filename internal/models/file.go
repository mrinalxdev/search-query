package models

import (
	"time"
)

type File struct {
	ID           int64     `json:"id"`
	ObjectName   string    `json:"object_name"`
	OriginalName string    `json:"original_name"`
	ContentType  string    `json:"content_type"`
	Size         int64     `json:"size"`
	ETag         string    `json:"etag"`
	UploadedBy   string    `json:"uploaded_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UploadRequest struct {
	FileName    string `json:"file_name" binding:"required"`
	ContentType string `json:"content_type"`
}

type UploadResponse struct {
	File         File   `json:"file"`
	PresignedURL string `json:"presigned_url,omitempty"`
}

type PresignedURLRequest struct {
	ObjectName string `json:"object_name" binding:"required"`
	Expiry     int    `json:"expiry"`
	Method     string `json:"method"` 
}

type PresignedURLResponse struct {
	URL    string    `json:"url"`
	Expiry time.Time `json:"expiry"`
}

type MultipartUploadRequest struct {
	FileName   string `json:"file_name" binding:"required"`
	TotalSize  int64  `json:"total_size" binding:"required"`
	PartSize   int64  `json:"part_size"`
	ContentType string `json:"content_type"`
}

type MultipartUploadResponse struct {
	UploadID      string   `json:"upload_id"`
	PartSize      int64    `json:"part_size"`
	TotalParts    int      `json:"total_parts"`
	PresignedURLs []string `json:"presigned_urls"`
}

type ListFilesResponse struct {
	Files      []File `json:"files"`
	Total      int    `json:"total"`
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
}