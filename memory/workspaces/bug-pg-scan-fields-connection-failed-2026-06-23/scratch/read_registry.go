package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
)

func main() {
	dsn := "postgres://gpay_admin:gpay_pass@localhost:5433/cdc_dw?sslmode=disable"
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer conn.Close(ctx)

	fmt.Println("Connected to cdc_dw successfully!")

	// Query source_object_registry
	rows, err := conn.Query(ctx,
		`SELECT sor.id, sor.source_engine_type, sor.source_database,
		        sor.source_schema, sor.source_object_name, sor.source_connection_id,
		        cr.connection_code
		   FROM cdc_system.source_object_registry sor
		   LEFT JOIN cdc_system.connection_registry cr
		          ON cr.id = sor.source_connection_id
		  WHERE sor.source_object_name = $1`, "failed_sync_logs")
	if err != nil {
		log.Fatalf("Query failed: %v\n", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, connID int64
		var engine, db, schema, obj, code string
		var schemaPtr *string

		err := rows.Scan(&id, &engine, &db, &schemaPtr, &obj, &connID, &code)
		if err != nil {
			log.Fatalf("Scan failed: %v\n", err)
		}
		if schemaPtr != nil {
			schema = *schemaPtr
		}
		fmt.Printf("SourceObjectRegistry ID: %d\n", id)
		fmt.Printf("  Engine: %s\n", engine)
		fmt.Printf("  DB: %s\n", db)
		fmt.Printf("  Schema: %s\n", schema)
		fmt.Printf("  Object: %s\n", obj)
		fmt.Printf("  ConnID: %d\n", connID)
		fmt.Printf("  ConnCode: %s\n", code)
	}
}
