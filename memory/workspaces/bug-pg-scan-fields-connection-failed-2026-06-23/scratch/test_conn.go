package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
)

func main() {
	dsn := "postgres://src_user:src_pass@localhost:5435/cdc_data_testing?sslmode=disable"
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer conn.Close(ctx)

	fmt.Println("Connected successfully!")

	// Query information_schema.columns
	rows, err := conn.Query(ctx,
		`SELECT column_name, data_type, is_nullable
		   FROM information_schema.columns
		  WHERE table_schema = $1 AND table_name = $2
		  ORDER BY ordinal_position`,
		"public", "failed_sync_logs")
	if err != nil {
		log.Fatalf("Query failed: %v\n", err)
	}
	defer rows.Close()

	fmt.Println("Columns in public.failed_sync_logs:")
	count := 0
	for rows.Next() {
		var name, dataType, isNullable string
		if err := rows.Scan(&name, &dataType, &isNullable); err != nil {
			log.Fatalf("Scan failed: %v\n", err)
		}
		fmt.Printf("- %s (%s, nullable: %s)\n", name, dataType, isNullable)
		count++
	}

	if err := rows.Err(); err != nil {
		log.Fatalf("Rows error: %v\n", err)
	}

	fmt.Printf("Total columns found: %d\n", count)
}
