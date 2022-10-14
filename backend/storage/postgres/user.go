package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/vagruchi/sqb"
)

func (r *Repo) AddNotificationToken(ctx context.Context, firebaseToken string) error {
	id := uuid.New()
	timeNow := time.Now().UTC()

	args := []interface{}{
		id,
		timeNow,
		firebaseToken,
	}

	createObjectQuery := `INSERT INTO tokens
		(id, created_at, token)
		VALUES ($1, $2, $3)`

	_, err := r.tx.ExecContext(ctx, createObjectQuery, args...)
	return err
}

func (r *Repo) ListNotificationTokens(ctx context.Context) ([]string, error) {
	query := sqb.From(sqb.TableName(`tokens`).As(`t`)).
		Select(sqb.Column(`t.token`))

	rawquery, args, err := sqb.ToPostgreSql(query)
	if err != nil {
		return nil, err
	}

	rows, err := r.tx.QueryContext(ctx, rawquery, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	tokens := []string{}

	for rows.Next() {
		object := sql.NullString{}

		err = rows.Scan(&object)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, object.String)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return tokens, nil
}
