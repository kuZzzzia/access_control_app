package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"

	"github.com/kuZzzzia/access_control_app/backend/pagination"
)

var ErrorObjectNotFound = errors.New("object not found")

type ImageInfo struct {
	ID        uuid.UUID
	CreatedAt time.Time

	PeopleNumber int

	Name        string
	ContentType string
	Extension   string
	Size        int64

	BucketName string
	UserID     uuid.UUID
}

type ObjectFilter struct {
	OlderThen *time.Time

	Pagination pagination.Pagination
}

type Repository interface {
	CreateObject(ctx context.Context, object *ImageInfo) error

	GetObject(ctx context.Context, objectID uuid.UUID) (*ImageInfo, error)
	GetLastObject(ctx context.Context) (*ImageInfo, error)

	ListObjects(ctx context.Context, filter ObjectFilter) ([]*ImageInfo, int, error)

	DeleteObject(ctx context.Context, objectID uuid.UUID) error
	DeleteObjects(ctx context.Context, deleteOlderThen time.Time) error
}

type ObjectStorage interface {
	GetObject(ctx context.Context, bucketId, objectId string) (*minio.Object, error)
	ListBuckets(ctx context.Context) ([]minio.BucketInfo, error)
	DeleteObject(ctx context.Context, bucketId, objectId string, opts minio.RemoveObjectOptions) error
	PutObject(ctx context.Context, body io.Reader, object *ImageInfo) error
	MakeBucket(ctx context.Context, bucketId, region string) error
	GetBucketName(ctx context.Context) (string, error)
}

type Service struct {
	repo  Repository
	store ObjectStorage
}

func NewObjectService(repo Repository, store ObjectStorage) *Service {
	return &Service{
		repo:  repo,
		store: store,
	}
}

func (f *Service) WithNewRepo(repo Repository) *Service {
	return &Service{
		repo:  repo,
		store: f.store,
	}
}

func (c *Service) CreateObject(ctx context.Context, body io.Reader, object *ImageInfo) error {
	var err error

	object.BucketName, err = c.store.GetBucketName(ctx)
	if err != nil {
		return fmt.Errorf("get bucket name %w", err)
	}

	err = c.repo.CreateObject(ctx, object)
	if err != nil {
		return fmt.Errorf("create object %w", err)
	}

	err = c.store.PutObject(ctx, body, object)
	if err != nil {
		return fmt.Errorf("put object %w", err)
	}

	return nil
}

func (c *Service) GetObject(ctx context.Context, objectID uuid.UUID) (*minio.Object, *ImageInfo, error) {
	object, err := c.repo.GetObject(ctx, objectID)
	if err != nil {
		return nil, nil, fmt.Errorf("get object info %w", err)
	}

	obj, err := c.store.GetObject(ctx, object.BucketName, objectID.String())
	if err != nil {
		return nil, nil, fmt.Errorf("get object %w", err)
	}

	return obj, object, nil
}

func (c *Service) GetObjectInfo(ctx context.Context, objectID uuid.UUID) (*ImageInfo, error) {
	object, err := c.repo.GetObject(ctx, objectID)
	if err != nil {
		return nil, fmt.Errorf("get object info %w", err)
	}

	return object, nil
}

func (c *Service) GetLastObject(ctx context.Context) (*minio.Object, *ImageInfo, error) {
	object, err := c.repo.GetLastObject(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("get object info %w", err)
	}
	obj, err := c.store.GetObject(ctx, object.BucketName, object.ID.String())
	if err != nil {
		return nil, nil, fmt.Errorf("get object %w", err)
	}

	return obj, object, nil
}

func (c *Service) DeleteObject(ctx context.Context, objectID uuid.UUID) error {
	object, err := c.repo.GetObject(ctx, objectID)
	if err != nil {
		return fmt.Errorf("get object info %w", err)
	}

	err = c.repo.DeleteObject(ctx, objectID)
	if err != nil {
		return fmt.Errorf("delete object info %w", err)
	}

	err = c.store.DeleteObject(ctx, object.BucketName, objectID.String(), minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("delete object %w", err)
	}

	return nil
}

func (srv *Service) DeleteObjects(ctx context.Context, deleteOlderThen time.Time) error {
	objects, _, err := srv.repo.ListObjects(ctx, ObjectFilter{
		OlderThen: &deleteOlderThen,
	})
	if err != nil {
		return fmt.Errorf("get object info %w", err)
	}

	err = srv.repo.DeleteObjects(ctx, deleteOlderThen)
	if err != nil {
		return fmt.Errorf("delete object info %w", err)
	}

	for i := range objects {
		err = srv.store.DeleteObject(ctx, objects[i].BucketName, objects[i].ID.String(), minio.RemoveObjectOptions{})
		if err != nil {
			return fmt.Errorf("delete object %w", err)
		}
	}

	return nil
}

func (srv *Service) ListObjectInfo(ctx context.Context, filter ObjectFilter) ([]*ImageInfo, int, error) {
	return srv.repo.ListObjects(ctx, filter)
}
