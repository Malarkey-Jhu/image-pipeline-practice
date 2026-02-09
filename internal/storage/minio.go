package storage

import (
	"context"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"sys-design/internal/config"
)

type MinioStore struct {
	Client        *minio.Client
	PresignClient *minio.Client
	Bucket        string
}

func NewMinioStore(cfg *config.Config) (*MinioStore, error) {
	internalEndpoint, internalSSL := normalizeEndpoint(cfg.MinioEndpoint, cfg.MinioUseSSL)
	client, err := minio.New(internalEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: internalSSL,
		Region: cfg.MinioRegion,
	})
	if err != nil {
		return nil, err
	}

	presignClient := client
	if cfg.MinioPublicURL != "" {
		publicEndpoint, publicSSL := normalizeEndpoint(cfg.MinioPublicURL, internalSSL)
		if publicEndpoint != "" {
			if pc, err := minio.New(publicEndpoint, &minio.Options{
				Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
				Secure: publicSSL,
				Region: cfg.MinioRegion,
			}); err == nil {
				presignClient = pc
			}
		}
	}

	store := &MinioStore{Client: client, PresignClient: presignClient, Bucket: cfg.MinioBucket}
	if err := store.ensureBucket(context.Background(), cfg.MinioRegion); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *MinioStore) ensureBucket(ctx context.Context, region string) error {
	exists, err := s.Client.BucketExists(ctx, s.Bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return s.Client.MakeBucket(ctx, s.Bucket, minio.MakeBucketOptions{Region: region})
}

func (s *MinioStore) PresignUpload(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	u, err := s.PresignClient.PresignedPutObject(ctx, s.Bucket, objectKey, expiry)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (s *MinioStore) ObjectExists(ctx context.Context, objectKey string) (bool, error) {
	_, err := s.Client.StatObject(ctx, s.Bucket, objectKey, minio.StatObjectOptions{})
	if err == nil {
		return true, nil
	}
	if minio.ToErrorResponse(err).Code == "NoSuchKey" {
		return false, nil
	}
	return false, err
}

func normalizeEndpoint(raw string, fallbackSSL bool) (string, bool) {
	endpoint := raw
	useSSL := fallbackSSL
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		endpoint = u.Host
		switch u.Scheme {
		case "http":
			useSSL = false
		case "https":
			useSSL = true
		}
	}
	return endpoint, useSSL
}
