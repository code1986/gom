package gom

import "database/sql"

type any = interface{}

type SQLConn interface {
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
	Query(query string, args ...any) (*sql.Rows, error)
}

var _ SQLConn = (*sql.DB)(nil)
var _ SQLConn = (*sql.Tx)(nil)

type Model interface {
	Exec(c SQLConn, name string, args ...any) (int64, int64, error)
	MultiInsert(c SQLConn, name string, slice any, batchSize int) (int64, int64, error)
	QueryRow(c SQLConn, name string, args ...any) (any, error)
	Query(c SQLConn, name string, args ...any) ([]any, error)
}

type Scanable interface {
	Scan(dest ...interface{}) error
}
