package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type (
	record struct {
		Parent    string    `ch:"parent"`
		Child     string    `ch:"child"`
		Content   []byte    `ch:"content"`
		CreatedAt time.Time `ch:"created_at"`
	}

	dbConn struct {
		conn driver.Conn
	}
)

func connect(ctx context.Context) (*dbConn, error) {
	var (
		conn, err = clickhouse.Open(&clickhouse.Options{
			Addr: []string{"clickhouse:" + os.Getenv("CLICKHOUSE_PORT")},
			Auth: clickhouse.Auth{
				Database: os.Getenv("CLICKHOUSE_DATABASE"),
				Username: os.Getenv("CLICKHOUSE_USER"),
				Password: os.Getenv("CLICKHOUSE_PASSWORD"),
			},
			ClientInfo: clickhouse.ClientInfo{
				Products: []struct {
					Name    string
					Version string
				}{
					{Name: "an-example-go-client", Version: "0.1"},
				},
			},
			Debugf: func(format string, v ...interface{}) {
				fmt.Printf(format, v)
			},
		})
	)

	if err != nil {
		return nil, err
	}

	if err := conn.Ping(ctx); err != nil {
		if exception, ok := err.(*clickhouse.Exception); ok {
			fmt.Printf("Exception [%d] %s \n%s\n", exception.Code, exception.Message, exception.StackTrace)
		}
		return nil, err
	}
	return &dbConn{
		conn: conn,
	}, nil
}

func (d *dbConn) Insert(ctx context.Context, rec []record) error {
	batch, err := d.conn.PrepareBatch(ctx, `INSERT INTO my_app_db.main`)
	if err != nil {
		return err
	}

	for _, record := range rec {
		if err := batch.AppendStruct(&record); err != nil {
			return err
		}
	}

	return batch.Send()
}

func (d *dbConn) List(ctx context.Context) ([]record, error) {
	rows, err := d.conn.Query(ctx, "SELECT * FROM my_app_db.main")
	if err != nil {
		return nil, err
	}

	var records []record

	for rows.Next() {
		var rec record

		if err := rows.ScanStruct(rec); err != nil {
			return nil, err
		}

		records = append(records, rec)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return records, nil
}
