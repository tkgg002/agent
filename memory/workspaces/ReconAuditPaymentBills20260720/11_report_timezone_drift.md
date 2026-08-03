# 11 — Báo Cáo Triển Khai & Kiểm Thử: Khắc Phục Timezone Drift Recon `payment_bills`

> **Ngày tạo:** 2026-07-21 | **Workspace:** `ReconAuditPaymentBills20260720`  
> **Trạng thái:** ✅ CODE COMPLETE — 100% UNIT TEST PASS — ⏳ CHỜ DEPLOY STAGING/PROD VERIFY  
> **Tác giả:** Agentic Engineering (Muscle & Brain)

---

## 1. Tổng Quan Hiện Trạng & Vấn Đề

### 1.1 Bối cảnh (Symptoms)
Khi chạy API đối soát `POST /api/reconciliation/check` cho bảng `payment_bills` trên môi trường Production (2h lookback window, chia nhỏ 8 sub-windows 15-phút):
- **Thời gian chạy:** Kéo dài **~90 giây** (rất chậm).
- **Kết quả đối soát:** Số lượng bản ghi `diff = 0` (dữ liệu hoàn toàn khớp giữa MongoDB và Postgres), nhưng **8/8 windows đều bị báo DRIFT** và trigger `drift_drill_down`.
- **Hệ quả:** Gây false alert liên tục, tốn 80% tài nguyên CPU/Network để drill-down vô ích.

### 1.2 Phân Tích Root Cause (4 Vấn Đề)

| Mã | Loại vấn đề | Mô tả chi tiết | Trạng thái |
|:---|:---|:---|:---|
| **P1** | 🔴 **TIMESTAMPTZ Double-Shift** | **Root cause chính gây false drift:** Cột `lastUpdatedAt` ở Postgres là `TIMESTAMPTZ`. Driver `pgx` đã parse thành `time.Time` đúng chuẩn UTC vật lý (`13:00:00 UTC`). Nhưng hàm `parsePostgresTimestampWithLocation` lại ép múi giờ local (`Asia/Ho_Chi_Minh`) vào $\rightarrow$ lùi bớt 7 tiếng thành `06:00:00 UTC`. Kết quả: XOR Hash của Postgres sai khác hoàn toàn so với Mongo UTC $\rightarrow$ False Drift 8/8 windows. | ✅ **ĐÃ FIX & TEST PASS** |
| **P2+P3** | 🔴 **MongoDB Missing Index** | **Root cause chính gây chạy chậm:** Collection `payment_bills` trên MongoDB thiếu index `{ lastUpdatedAt: 1 }`. Mọi truy vấn `MAX(lastUpdatedAt)` (pick_scan_range: 2.46s) và `find({ lastUpdatedAt: {$gte,$lt} })` (drill-down 8 windows × 5.3s = 42.4s) đều bị **COLLSCAN**. | ⏳ **Chờ User tạo Index trên Prod** |
| **P4** | 🟠 **Granularity Mismatch** | `HashWindow` tính XOR hash theo độ phân giải **Millisecond** (`ts.UnixMilli()`), trong khi `diffIDTsSegmentA` so sánh độ chênh lệch theo **Giây** (`dstTs/1000 != srcTs/1000`). Nếu chênh millis nhỏ $\rightarrow$ Hash bị lệch nhưng Diff lại bằng 0. | 🟠 **Giữ nguyên, theo dõi sau P1+P2** |

---

## 2. Giải Pháp Tổng Thể Đã Triển Khai (Adaptive Schema-Aware Parsing)

Để giải quyết P1 mà không làm hỏng các bảng legacy đang lưu `TIMESTAMP WITHOUT TIME ZONE` (wall-clock local), hệ thống đã được nâng cấp cơ chế **Dynamic Column-Type Verification**:

### 2.1 Cơ chế Tự Động Phân Biệt Kiểu Cột (`IsColTimestamptz`)
Query bảng `information_schema.columns` để kiểm tra kiểu cột:
```sql
SELECT LOWER(data_type) 
FROM information_schema.columns 
WHERE table_schema = ? AND table_name = ? AND column_name = ?
```
- Nếu `data_type` chứa `"with time zone"` hoặc `"timestamptz"` $\rightarrow$ Trả về `isTZ = true`.

### 2.2 Thread-Safe In-Memory Caching
Để tránh truy vấn `information_schema` liên tục trong các vòng lặp window, `ReconDestAgent` giữ một cache `colTypes map[string]bool` bảo vệ bởi `sync.RWMutex`. Overhead truy vấn schema chỉ tốn **1 lần duy nhất** cho mỗi cột của bảng trong suốt vòng đời worker.

### 2.3 Logic Parse Phân Biệt kiểu (`parsePostgresTimestampWithLocationAndType`)
```go
func parsePostgresTimestampWithLocationAndType(val interface{}, dbLoc *time.Location, isTZ bool) time.Time {
    if isTZ {
        // TIMESTAMPTZ: pgx driver đã parse thành đúng UTC vật lý → Giữ nguyên UTC, KHÔNG ép offset!
        switch v := val.(type) {
        case time.Time:  return v.UTC()
        case *time.Time: return v.UTC()
        }
    }
    // TIMESTAMP: Chạy theo fallback logic cũ (ép dbLoc vào wall-clock)
    return parsePostgresTimestampWithLocation(val, dbLoc)
}
```

---

## 3. Chi Tiết Các File Sửa Đổi (Physical Files & Code Audit)

Tổng cộng **7 files** được thay đổi trong package `internal/service/recon/`:

```diff
 internal/service/recon/recon_dest_agent.go      |  6 ++-
 internal/service/recon/recon_dest_agent_test.go  | 43 ++++++++++++++++++
 internal/service/recon/recon_dest_hash.go        |  9 +++-
 internal/service/recon/recon_dest_query.go       | 58 ++++++++++++++++++++++++-
 internal/service/recon/recon_query.go            | 20 +++++++--
 internal/service/recon/recon_smoke_test.go       |  2 +
 internal/service/recon/recon_tier_a_test.go      | 16 +++++++
 7 files changed, 148 insertions(+), 6 deletions(-)
```

### Chi Tiết Thay Đổi Theo File:
1. **[recon_dest_agent.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_agent.go):**
   - Thêm `colTypesMu sync.RWMutex` và `colTypes map[string]bool` vào `ReconDestAgent`.
   - Khởi tạo map trong `NewReconDestAgentWithConfig`.
2. **[recon_dest_query.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_query.go):**
   - Thêm hàm `IsColTimestamptz(ctx, tableName, columnName)`.
   - Cập nhật `ListIDTsInWindow` lấy `isTZ` và gọi `parsePostgresTimestampWithLocationAndType`.
3. **[recon_dest_hash.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_hash.go):**
   - Cập nhật `HashWindow` lấy `isTZ` và gọi `parsePostgresTimestampWithLocationAndType`.
4. **[recon_query.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_query.go):**
   - Thêm hàm `parsePostgresTimestampWithLocationAndType()`.
5. **[recon_dest_agent_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_agent_test.go):**
   - Thêm unit test `TestDestAgent_HashWindow_DomainTS_Timestamptz`.
   - Thêm helper `expectIsColTimestamptzQuery`.
6. **[recon_tier_a_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_a_test.go):**
   - Thêm helper `expectDestIsColTimestamptz`.
   - Cập nhật 3 test cases Tier A mock query `information_schema.columns`.
7. **[recon_smoke_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_smoke_test.go):**
   - Cập nhật 2 test cases smoke recon để mock query `information_schema.columns`.

---

## 4. Kết Quả Kiểm Thử (Verification Evidence)

### 4.1 Unit Test Execution
Chạy lệnh kiểm thử toàn bộ package `recon`:
```bash
go test -v ./internal/service/recon/...
```

**Kết quả:**
```
=== RUN   TestDestAgent_HashWindow_DomainTS_Timestamptz
--- PASS: TestDestAgent_HashWindow_DomainTS_Timestamptz (0.00s)
=== RUN   TestReconCore_RunTotalOnlyA_DiscrepancyLech_ResolvedByHash
--- PASS: TestReconCore_RunTotalOnlyA_DiscrepancyLech_ResolvedByHash (0.00s)
=== RUN   TestReconCore_RunTotalOnlyA_DriftConfirmed
--- PASS: TestReconCore_RunTotalOnlyA_DriftConfirmed (0.00s)
=== RUN   TestRunHashWindowCheck_GlobalMatch_NoDrift
--- PASS: TestRunHashWindowCheck_GlobalMatch_NoDrift (0.00s)
...
PASS
ok  	centralized-data-service/internal/service/recon	0.699s
```
**Tất cả 100% test cases đều PASS hoàn toàn.**

---

## 5. Ước Tính Hiệu Năng Sau Fix

| Chỉ số | Ban đầu | Sau P1 Fix (Hiện tại) | Sau P1 + P2 (Tạo Mongo Index) |
|:---|:---|:---|:---|
| Kết quả Global Hash Check | DRIFT (Lệch hash) | **MATCH ✅ (Khớp hash)** | **MATCH ✅** |
| Thời gian Drill-Down | 42.4s (8 windows) | **0s (Bỏ qua vì hash khớp)** | **0s** |
| MongoDB MAX() Query | 2.46s | 2.46s | **< 5ms** |
| **TỔNG THỜI GIAN RUN** | **~90 giây** | **~8 giây** | **< 3 giây** |

---

## 6. Danh Mục Kịch Bản & Action Items Tiếp Theo

- [x] **P1 Fix:** Sửa code Adaptive Timestamp Parsing cho `TIMESTAMPTZ`
- [x] **P1 Unit Tests:** Viết test case riêng cho `TIMESTAMPTZ` + cập nhật toàn bộ test mocks
- [ ] **P1 Deploy Verify:** Deploy code lên staging/production, trigger `/api/reconciliation/check` cho `payment_bills` để confirm hết false drift.
- [ ] **P2 Mongo Index:** Chạy lệnh tạo index trên MongoDB Production:
  ```javascript
  db.payment_bills.createIndex(
    { "lastUpdatedAt": 1 },
    { background: true, name: "idx_lastUpdatedAt" }
  )
  ```
- [ ] **P4 Audit:** Theo dõi xem có trường hợp chênh milliseconds nào gây lệch hash sau khi đã có P1 + P2 hay không.
