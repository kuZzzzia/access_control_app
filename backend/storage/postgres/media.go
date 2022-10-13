package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/kuZzzzia/access_control_app/backend/service"
	"github.com/vagruchi/sqb"
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
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

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

func addUserFilters(q *sqb.SelectStmt, filters service.ObjectFilter, isCount bool) *sqb.SelectStmt {
	query := *q

	query = query.Where(append(query.WhereStmt.Exprs, sqb.Raw(`f.deleted_at IS NULL`))...)

	if filters.OlderThen != nil {
		query = query.Where(append(query.WhereStmt.Exprs, sqb.BinaryOp(sqb.Column(`f.created_at`), "<=", sqb.Arg{V: *filters.OlderThen}))...)
	}

	if !isCount {
		if len(filters.Pagination.OrderBy) == 0 {
			filters.Pagination.AddOrderByDesc(`a.created_at`)
		}
		query = *filters.Pagination.Apply(&query)
	}

	return &query
}

func (r *Repo) countObjects(ctx context.Context, filters service.ObjectFilter) (int, error) {
	query := sqb.From(sqb.TableName(`files`).As(`f`)).
		Select(sqb.Count(sqb.Column(`f.id`)))

	query = *addUserFilters(&query, filters, false)

	rawquery, args, err := sqb.ToPostgreSql(query)
	if err != nil {
		return 0, err
	}

	return count(ctx, r.tx, rawquery, args)
}

func (r Repo) ListObjects(ctx context.Context, filter service.ObjectFilter) ([]*service.ImageInfo, int, error) {
	total, err := r.countObjects(ctx, filter)
	if err != nil || total == 0 {
		return nil, 0, nil
	}

	query := sqb.From(sqb.TableName(`files`).As(`f`)).
		Select(sqb.Column(`f.id`), sqb.Column(`f.created_at`), sqb.Column(`f.name`),
			sqb.Column(`f.people_number`), sqb.Column(`f.user_id`),
			sqb.Column(`f.extension`), sqb.Column(`f.size`), sqb.Column(`f.bucket_name`))

	q, args, err := sqb.ToPostgreSql(query)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.tx.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}

	defer rows.Close()

	objects := []*service.ImageInfo{}

	for rows.Next() {
		object := &service.ImageInfo{}

		err = rows.Scan(&object.ID, &object.CreatedAt, &object.Name, &object.PeopleNumber, &object.UserID,
			&object.Extension, &object.Size, &object.BucketName)
		if err != nil {
			return nil, 0, err
		}
		objects = append(objects, object)
	}

	err = rows.Err()
	if err != nil {
		return nil, 0, err
	}

	return objects, 0, nil
}

func (r Repo) DeleteObjects(ctx context.Context, deleteOlderThen time.Time) error {
	deleted_at := time.Now()
	_, err := r.tx.ExecContext(
		ctx,
		`UPDATE files
		SET deleted_at = $1
		WHERE created_at <= $2
		`, deleted_at, deleteOlderThen,
	)
	return err
}

func count(ctx context.Context, tx QueryerContext, query string, args []interface{}) (int, error) {
	total := sql.NullInt32{}

	err := tx.QueryRowContext(ctx, query, args...).Scan(&total)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}

	return int(total.Int32), nil
}
