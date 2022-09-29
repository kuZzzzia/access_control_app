package minio

import (
	"context"
	"fmt"
	"io"

	"github.com/kuZzzzia/access_control_app/backend/service"
	"github.com/minio/minio-go/v7"
)

type Minio struct {
	Address         string `yaml:"address"`
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
	SSLStr          string `yaml:"ssl_string"`
	SSL             bool   `yaml:"use_ssl"`
	Region          string `yaml:"region"`
	BucketID        string `yaml:"bucket_id"`
}

type ObjectStorage struct {
	Minio *minio.Client

	Region string
	Bucket string
}

func (os *ObjectStorage) MakeBucket(ctx context.Context, bucketId, region string) error {
	err := os.Minio.MakeBucket(ctx, bucketId, minio.MakeBucketOptions{Region: region})
	if err != nil {
		return err
	}

	return nil
}

func (os *ObjectStorage) GetBucketName(ctx context.Context) (string, error) {
	exists, err := os.Minio.BucketExists(ctx, os.Bucket)
	if err != nil {
		return "", fmt.Errorf("bucket exists %w", err)
	}

	if !exists {
		err := os.MakeBucket(ctx, os.Bucket, os.Region)
		if err != nil {
			return "", fmt.Errorf("make bucket %w", err)
		}
	}

	return os.Bucket, nil
}

func (os *ObjectStorage) ListBuckets(ctx context.Context) ([]minio.BucketInfo, error) {
	bs, err := os.Minio.ListBuckets(ctx)
	if err != nil {
		return nil, err
	}

	return bs, nil
}

const metaPrefix = "meta-"

func (os *ObjectStorage) PutObject(ctx context.Context, body io.Reader, objectInfo *service.ImageInfo) error {
	_, err := os.Minio.PutObject(ctx, objectInfo.BucketName, objectInfo.ID.String(), body, objectInfo.Size,
		minio.PutObjectOptions{
			ContentType: objectInfo.ContentType,
			UserMetadata: map[string]string{
				metaPrefix + "content-type": objectInfo.ContentType,
			},
		})
	if err != nil {
		return err
	}

	return nil
}

func (os *ObjectStorage) DeleteObject(ctx context.Context, bucketId, objectId string, opts minio.RemoveObjectOptions) error {
	err := os.Minio.RemoveObject(ctx, bucketId, objectId, opts)
	if err != nil {
		return err
	}
	return nil
}

func (os *ObjectStorage) GetObject(ctx context.Context, bucketId, objectId string) (*minio.Object, error) {
	opt := minio.GetObjectOptions{}

	_, err := os.Minio.StatObject(ctx, bucketId, objectId, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("stat object %w", err)
	}

	r, err := os.Minio.GetObject(ctx, bucketId, objectId, opt)
	if err != nil {
		return nil, fmt.Errorf("get object %w", err)
	}

	return r, nil
}
