# Progress Audit Log — SFTP Snapshot Offset Reset

[2026-08-12T17:15:00+07:00] [Muscle:Gemini] Implementation completed for SFTP Consumer Group Offset Reset.

## Summary of Changes
- `internal/handler/orchestration/snapshot_runner_handler.go`:
  - Added `kafkaGroupID` field and updated constructor `NewSnapshotRunner`.
  - Added `resetTopicConsumerOffset(ctx context.Context, topic string) error` method using `kafka.Client.OffsetCommit` to set partition 0 offset to 0.
  - In `runSnapshot`, for `isSFTP == true`, automatically resolve topic and reset consumer group offset to 0 before logging progress `done`.
- `internal/server/server_setup.go`:
  - Passed `cfg.Kafka.GroupID` into `NewSnapshotRunner`.

## Verification Results
- `go test ./internal/handler/orchestration/...`: PASS
- `go test ./internal/server/...`: PASS
- `go build ./cmd/worker`: SUCCESS (Passes compilation 100%)
- `go build ./cmd/...` (in `cdc-cms-service`): SUCCESS
