package sqlxadapter

import (
	"context"
	"database/sql"

	"github.com/alexbro4u/uowkit/uow"
	"github.com/jmoiron/sqlx"
)

// TxFactory implements [uow.TxFactory] for [*sqlx.DB].
type TxFactory struct {
	db   *sqlx.DB
	opts *sql.TxOptions
}

// NewTxFactory creates a new [TxFactory].
func NewTxFactory(db *sqlx.DB, opts *sql.TxOptions) *TxFactory {
	return &TxFactory{db: db, opts: opts}
}

// Begin implements [uow.TxFactory] by starting a new [*sqlx.Tx].
func (f *TxFactory) Begin(ctx context.Context) (uow.Tx, error) {
	sqlxTx, err := f.db.BeginTxx(ctx, f.opts)
	if err != nil {
		return nil, err
	}
	return &tx{sqlxTx: sqlxTx}, nil
}

type tx struct {
	sqlxTx *sqlx.Tx
}

func (t *tx) Commit() error   { return t.sqlxTx.Commit() }
func (t *tx) Rollback() error { return t.sqlxTx.Rollback() }

// Unwrap extracts the underlying [*sqlx.Tx] from a [uow.Tx].
func Unwrap(t uow.Tx) (*sqlx.Tx, bool) {
	a, ok := t.(*tx)
	if !ok {
		return nil, false
	}
	return a.sqlxTx, true
}
