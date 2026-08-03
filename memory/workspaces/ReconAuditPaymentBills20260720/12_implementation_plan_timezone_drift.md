# 12 — Kế hoạch Triển khai: Khắc phục Triệt để Timezone Drift bằng Dynamic Column-Type Verification

> Tạo: 2026-07-20T17:40:00+07:00 | Task: Hotfix/Refactor
> Status: Đang chờ phê duyệt (Requesting Review)

---

## 1. Các file sẽ sửa đổi (Proposed Changes)

### [centralized-data-service](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/)

#### [MODIFY] [recon_dest_agent.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_agent.go)
- Thêm `mu sync.RWMutex` và `colTypes map[string]bool` vào `ReconDestAgent` struct.
- Khởi tạo map này trong hàm `NewReconDestAgentWithConfig`.

#### [MODIFY] [recon_dest_query.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_query.go)
- Thêm hàm `IsColTimestamptz(ctx context.Context, tableName, columnName string) (bool, error)` truy vấn schema `information_schema.columns` và thực hiện caching kết quả.
- Cập nhật hàm `ListIDTsInWindow`:
  - Trước khi scan các dòng, lấy flag `isTZ` qua `da.IsColTimestamptz`.
  - Khi scan timestamp, gọi `parsePostgresTimestampWithLocationAndType(*tsVal, da.getDBLocation(), isTZ)` thay vì hàm cũ.

#### [MODIFY] [recon_dest_hash.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_hash.go)
- Cập nhật hàm `HashWindow` (nhánh domain timestamp):
  - Lấy flag `isTZ` qua `da.IsColTimestamptz`.
  - Khi scan timestamp, gọi `parsePostgresTimestampWithLocationAndType(ts, da.getDBLocation(), isTZ)` thay vì hàm cũ.

#### [MODIFY] [recon_query.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_query.go)
- Thêm hàm helper `parsePostgresTimestampWithLocationAndType(val interface{}, dbLoc *time.Location, isTZ bool) time.Time`.
- Sửa hàm `parsePostgresTimestampWithLocation` cũ để gọi qua hàm mới với `isTZ = false`.

#### [MODIFY] [recon_dest_agent_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_agent_test.go)
- Cập nhật unit tests: Mock thêm truy vấn từ `information_schema.columns` trả về data type phù hợp cho các test case kiểm tra `HashWindow` và `ListIDTsInWindow` dùng domain timestamp.

---

## 2. Kế hoạch Kiểm thử & Xác minh (Verification Plan)

### Automated Tests
- Chạy toàn bộ suite test recon để đảm bảo các thay đổi không gây regression và hoạt động đúng logic:
  `go test ./internal/service/recon/... -v`
- Chạy build dự án:
  `go build ./internal/service/recon/...`

### Manual Verification
- Xác nhận logic hoạt động bằng cách đối chiếu logs khi deploy lên staging/production.
- Đảm bảo XOR hash khớp và không còn spam drift_drill_down.
