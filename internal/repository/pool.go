package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type DBExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func NewPool(connUrl string) *pgxpool.Pool {
	logger := zap.L()

	ctx := context.Background()

	var err error
	pool, err := pgxpool.New(ctx, connUrl)

	if err != nil {
		logger.Fatal("Failed to establish database connection", zap.Error(err))
	}

	var one uint
	err = pool.QueryRow(ctx, "SELECT 1").Scan(&one)
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
	return pool
}

func rowToStruct[T any](ctx context.Context, exec DBExecutor, query string, args ...any) (res T, err error) {
	rows, err := exec.Query(ctx, query, args...)
	if err != nil {
		return res, err
	}

	return pgx.CollectOneRow(rows, pgx.RowToStructByName[T])
}

func rowsToStruct[T any](ctx context.Context, exec DBExecutor, query string, args ...any) (res []T, err error) {
	rows, err := exec.Query(ctx, query, args...)
	if err != nil {
		return res, err
	}

	return pgx.CollectRows(rows, pgx.RowToStructByName[T])
}
