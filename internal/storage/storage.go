package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Storage struct {
	client    *minio.Client
	bucket    string
	publicURL string
}

func FromEnv() (*Storage, error) {
	client, err := minio.New(os.Getenv("MINIO_ENDPOINT"), &minio.Options{
		Creds:  credentials.NewStaticV4(os.Getenv("MINIO_ACCESS_KEY"), os.Getenv("MINIO_SECRET_KEY"), ""),
		Secure: false,
	})
	if err != nil {
		return nil, err
	}
	return &Storage{
		client:    client,
		bucket:    os.Getenv("MINIO_BUCKET_NAME"),
		publicURL: strings.TrimRight(os.Getenv("MINIO_PUBLIC_URL"), "/"),
	}, nil
}

func (s *Storage) URL(objectName string) string {
	if objectName == "" {
		return ""
	}
	return s.publicURL + "/" + objectName
}

func ObjectName(cometID uint, kind, extension string) string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("comet-%d-%s-%s%s", cometID, kind, hex.EncodeToString(b), extension)
}

func (s *Storage) Put(ctx context.Context, objectName string, r io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, objectName, r, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (s *Storage) Remove(ctx context.Context, objectName string) error {
	return s.client.RemoveObject(ctx, s.bucket, objectName, minio.RemoveObjectOptions{})
}
