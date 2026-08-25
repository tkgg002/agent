# 05_progress_sftp_file_upsert.md

## Audit Log & Technical Root Cause Analysis

### [2026-08-13 14:53:00] [Brain:GeminiCore] Phản tỉnh Kiến trúc Hệ thống (Architectural Audit):

- **Nhận thức sai lầm**: Đề xuất tạo standalone reader tự tiêu thụ trong `SnapshotRunner` là một dạng **"cheat code" / workaround tạm bợ**, làm phá vỡ kiến trúc Single Source of Truth của `KafkaConsumer`.
- **Định hướng Kiến trúc Chuẩn**:
  - `KafkaConsumer` là nơi duy nhất quản lý vòng đời reader và pipeline tiêu thụ.
  - Expose `RewindTopicOffset(ctx, topic, offset)` dùng active reader (có MemberID valid) để commit offset=0 từ bên trong group — Broker CHẤP NHẬN.
  - `SnapshotRunner` chỉ gọi `topicController.RewindTopicOffset(ctx, topic, 0)`.

---

### [2026-08-13 15:34:00] [Brain+Muscle:Inherit] DONE — SFTP Snapshot RewindTopicOffset

**Thay đổi đã thực thi và verified build OK:**

| File | Thay đổi |
|---|---|
| `internal/handler/shadow/kafka_consumer.go` | +30 dòng: thêm `RewindTopicOffset` dùng active reader CommitMessages |
| `internal/handler/orchestration/snapshot_runner_handler.go` | -46 dòng: xóa `resetTopicConsumerOffset`; đổi interface `topicReloader`→`topicController`; dùng `RewindTopicOffset` |
| `internal/server/server_setup.go` | -1/+1: `SetTopicReloader` → `SetTopicController` |

**Verification:**
- `go build ./cmd/worker` → ✅ exit code 0
- `go test ./internal/handler/shadow/...` → ✅ PASS
- `go test ./internal/handler/orchestration/...` → ✅ PASS
