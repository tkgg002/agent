# Thiết kế kỹ thuật - Recon Dest Agent Refactor

Tài liệu này chi tiết cấu trúc phân tách file `recon_dest_agent.go` thành các module nhỏ có tính gắn kết cao.

## Kiến trúc phân tách đề xuất

### 1. `recon_dest_agent.go` (Rút gọn)
- Giữ lại struct lõi `ReconDestAgent` và các constructor:
  - `NewReconDestAgent`
  - `NewReconDestAgentWithConfig`
- Helper transaction:
  - `readOnlyDB`
- Imports cần thiết: `context`, `github.com/sony/gobreaker`, `golang.org/x/time/rate`, `gorm.io/gorm`, `go.uber.org/zap`.

### 2. `recon_dest_models.go` (Mới)
- Cấu hình & structs:
  - `ReconDestAgentConfig`
  - `BucketStat`
  - `IDTs`
- Imports: `time`.

### 3. `recon_dest_hash.go` (Mới)
- Logic băm XOR trên DB Postgres:
  - `HashWindow`
  - `BucketHash`
- Imports: `context`, `time`, `fmt`.

### 4. `recon_dest_query.go` (Mới)
- Logic các query count/aggregate trên Postgres:
  - `CountRows`
  - `CountInWindow`
  - `BucketCounts`
  - `ListIDTsInWindow`
  - `MaxWindowTs`
- Imports: `context`, `time`, `fmt`.

### 5. `recon_dest_stream.go` (Mới)
- Logic list/stream dữ liệu:
  - `ListIDsInWindow`
  - `GetIDs`
  - `GetAllIDs`
- Imports: `context`, `time`, `fmt`.

### 6. `recon_dest_legacy.go` (Mới)
- Logic shims tương thích ngược:
  - `GetChunkHashes`
- Imports: `context`, `fmt`.

### 7. `recon_dest_safety.go` (Mới)
- Logic kiểm tra và escape SQL identifier an toàn:
  - `validateIdent`
  - `quoteIdent`
  - `quoteRelation`
- Imports: `fmt`, `strings`.
