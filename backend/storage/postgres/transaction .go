package postgres

import (
	"context"
	"database/sql"

	"github.com/rs/zerolog"
)

type QueryerContext interface {
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type noCloseTx struct {
	*sql.Tx
}

func (noCloseTx) Commit() error {
	return nil
}

func (noCloseTx) Rollback() error {
	return nil
}

type transactionalRepo struct {
	tx QueryerContext
}

func (tr transactionalRepo) beginTx(ctx context.Context) (transactionalRepo, error) {
	switch db := tr.tx.(type) {
	case *sql.DB:
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return transactionalRepo{}, err
		}
		return transactionalRepo{tx}, nil
	case *sql.Tx:
		return transactionalRepo{&noCloseTx{Tx: db}}, nil
	}
	return transactionalRepo{tr.tx}, nil
}

func (tr transactionalRepo) Rollback(ctx context.Context, shouldRollback *bool) {
	if shouldRollback != nil && *shouldRollback {
		tx, ok := tr.tx.(interface{ Rollback() error })
		if !ok {
			return
		}

		err := tx.Rollback()
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("rollback schedule repo")
		}
	}
}

func (tr transactionalRepo) Commit() error {
	tx, ok := tr.tx.(interface{ Commit() error })
	if !ok {
		return nil
	}

	return tx.Commit()
}
