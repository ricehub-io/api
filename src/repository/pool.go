package repository

import (
	"context"
	"fmt"
	"reflect"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type DBExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type UUIDOrString interface {
	~string | uuid.UUID
}

var db *pgxpool.Pool

func Init(connUrl string) {
	logger := zap.L()

	ctx := context.Background()

	var err error
	db, err = pgxpool.New(ctx, connUrl)

	if err != nil {
		logger.Fatal("Failed to establish database connection", zap.Error(err))
	}

	// run a test query to make sure db is working
	var one uint
	err = db.QueryRow(ctx, "SELECT 1").Scan(&one)
	if err != nil {
		logger.Fatal("Failed to perform a connection test", zap.Error(err))
	}
	if one != 1 {
		logger.Fatal("Invalid connection test result",
			zap.Uint("expected", 1),
			zap.Uint("got", one),
		)
	}

	logger.Info("Connection with database successfully established")
}

func Close() {
	db.Close()
}

func StartTx(ctx context.Context) (pgx.Tx, error) {
	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	return tx, err
}

func _rowToStruct[T any](exec DBExecutor, query string, args ...any) (res T, err error) {
	// compile-time check to make sure T is a struct
	var zero T
	if reflect.TypeOf(zero).Kind() != reflect.Struct {
		return res, fmt.Errorf("T must be a struct, got %v", zero)
	}

	rows, err := exec.Query(context.Background(), query, args...)
	if err != nil {
		return zero, err
	}

	return pgx.CollectOneRow(rows, pgx.RowToStructByName[T])
}

func rowToStruct[T any](query string, args ...any) (res T, err error) {
	return _rowToStruct[T](db, query, args...)
}

func txRowToStruct[T any](tx pgx.Tx, sql string, args ...any) (res T, err error) {
	return _rowToStruct[T](tx, sql, args...)
}

func rowsToStruct[T any](query string, args ...any) (res []T, err error) {
	rows, err := db.Query(context.Background(), query, args...)
	if err != nil {
		return res, err
	}

	return pgx.CollectRows(rows, pgx.RowToStructByName[T])
}
