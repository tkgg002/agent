package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/nats-io/nats.go"
)

type ScanFieldsPayload struct {
	RegistryID      uint   `json:"registry_id"`
	SourceObjectID  int64  `json:"source_object_id"`
	ShadowBindingID int64  `json:"shadow_binding_id"`
	TargetTable     string `json:"target_table"`
	SourceTable     string `json:"source_table"`
	SyncEngine      string `json:"sync_engine"`
	SourceType      string `json:"source_type"`
	LegacySourceID  string `json:"legacy_source_id"`
}

func main() {
	// 1. Get shadow binding ID from cdc_dw
	dbDSN := "postgres://gpay_admin:gpay_pass@localhost:5433/cdc_dw?sslmode=disable"
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, dbDSN)
	if err != nil {
		log.Fatalf("Unable to connect to cdc_dw: %v\n", err)
	}
	defer conn.Close(ctx)

	// Query shadow binding id
	var shadowBindingID int64
	err = conn.QueryRow(ctx,
		`SELECT id FROM cdc_system.shadow_binding
		  WHERE source_object_id = 52 LIMIT 1`).Scan(&shadowBindingID)
	if err != nil {
		log.Fatalf("Failed to query shadow_binding: %v\n", err)
	}

	fmt.Printf("Found shadowBindingID: %d\n", shadowBindingID)

	// 2. Publish to NATS
	natsURL := "nats://cdc_worker:worker_secret_2026@localhost:14222"
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v\n", err)
	}
	defer nc.Close()

	payload := ScanFieldsPayload{
		RegistryID:      52,
		SourceObjectID:  52,
		ShadowBindingID: shadowBindingID,
		TargetTable:     "shadow_pg_dev.failed_sync_logs",
		SourceTable:     "cdc_data_testing.failed_sync_logs",
		SyncEngine:      "debezium",
		SourceType:      "postgresql",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("Failed to marshal payload: %v\n", err)
	}

	// Subscribe to result subject to see response
	sub, err := nc.SubscribeSync("cdc.result.scan-fields")
	if err != nil {
		log.Fatalf("Failed to subscribe: %v\n", err)
	}

	fmt.Println("Publishing scan-fields command...")
	err = nc.Publish("cdc.cmd.scan-fields", data)
	if err != nil {
		log.Fatalf("Failed to publish: %v\n", err)
	}

	fmt.Println("Waiting for response on cdc.result.scan-fields...")
	msg, err := sub.NextMsg(5 * time.Second)
	if err != nil {
		log.Fatalf("Timeout/Error waiting for msg: %v\n", err)
	}

	fmt.Printf("Received response: %s\n", string(msg.Data))
}
