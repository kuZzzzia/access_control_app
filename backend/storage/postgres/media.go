package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/kuZzzzia/access_control_app/backend/service"
)

type Repo struct {
	transactionalRepo
}

func NewRepository(tx QueryerContext) *Repo {
	return &Repo{
		transactionalRepo{
			tx: tx,
		},
	}
}

func (pr *Repo) BeginTx(ctx context.Context) (*Repo, error) {
	traR, err := pr.beginTx(ctx)
	if err != nil {
		return nil, err
	}

	return &Repo{
		transactionalRepo: traR,
	}, nil
}

func (r Repo) CreateObject(ctx context.Context, object *service.ImageInfo) error {
	timeNow := time.Now().UTC()

	args := []interface{}{
		object.ID,
		timeNow,
		object.Name,

		object.PeopleNumber,

		object.UserID,

		object.Extension,
		object.Size,
		object.BucketName,
	}

	createObjectQuery := `INSERT INTO files
		(id, created_at, name, people_number, user_id, extension, size, bucket_name)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.tx.ExecContext(ctx, createObjectQuery, args...)
	return err
}

func (r Repo) GetObject(ctx context.Context, objectID uuid.UUID) (*service.ImageInfo, error) {
	query := `SELECT id, created_at, name, people_number, user_id, extension, size, bucket_name
			FROM files
 			WHERE id = $1 and deleted_at is null`

	rows, err := r.tx.QueryContext(ctx, query, objectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, service.ErrorObjectNotFound
	}

	object := &service.ImageInfo{}

	err = rows.Scan(&object.ID, &object.CreatedAt, &object.Name, &object.PeopleNumber, &object.UserID,
		&object.Extension, &object.Size, &object.BucketName)
	if err != nil {
		return nil, err
	}

	return object, nil
}
func (r Repo) GetLastObject(ctx context.Context) (*service.ImageInfo, error) {
	query := `SELECT id, created_at, name, people_number, user_id, extension, size, bucket_name
			FROM files
 			WHERE deleted_at is null
			ORDER BY created_at desc
			LIMIT 1`

	rows, err := r.tx.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, service.ErrorObjectNotFound
	}

	object := &service.ImageInfo{}

	err = rows.Scan(&object.ID, &object.CreatedAt, &object.Name, &object.PeopleNumber, &object.UserID,
		&object.Extension, &object.Size, &object.BucketName)
	if err != nil {
		return nil, err
	}

	return object, nil
}

func (r Repo) DeleteObject(ctx context.Context, objectID uuid.UUID) error {
	deleted_at := time.Now()
	_, err := r.tx.ExecContext(
		ctx,
		`UPDATE files
		SET deleted_at = $1
		WHERE id = $2
		`, deleted_at, objectID,
	)
	return err
}
