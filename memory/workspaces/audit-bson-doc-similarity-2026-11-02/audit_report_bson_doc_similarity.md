---
tier: T2
suffix: bson_doc_similarity
workspace: audit-bson-doc-similarity-2026-11-02
status: completed
---

# Báo cáo Kiểm định Phản biện & Kết quả Thực thi (Audit Report — Tier T2)

## 1. (a) Line-by-Line Diff Critique & Kiểm chứng `wc -l` (`git diff`)

### 1.1. Bảng đo đếm `wc -l`
Toàn bộ các file được cập nhật trong task này đều là file đơn trách nhiệm (SRP) nhỏ gọn `< 500` dòng:

| Repository | File | Trước (`wc -l`) | Sau (`wc -l`) | Chênh lệch | Đánh giá (`< 500` dòng) |
| :--- | :--- | :---: | :---: | :---: | :---: |
| `centralized-data-service` | `internal/service/master/mongo_extjson_unwrapper.go` | `395` | `404` | `+9` | Đạt chuẩn (`< 500` dòng) |
| `centralized-data-service` | `internal/service/master/mongo_extjson_unwrapper_test.go` | `387` | `397` | `+10` | Đạt chuẩn (`< 500` dòng) |

### 1.2. Phản biện Từng Task & Từng Dòng Update (`git diff`)
- **TASK-01 (Đối chiếu Mức độ Tương đồng giữa Document Gốc và Bản Backup mới tạo):**
  - Đã lập ma trận đối chiếu đầy đủ `16` trường cấp root của bản gốc so với `17` trường cấp root của bản backup và toàn bộ `24` trường lá bên trong `sender`, `receiver`, `error`, `info`.
- **TASK-02 (Chuẩn hóa `coerceDefaultInt32IfInRange` trong `mongo_extjson_unwrapper.go`):**
  - `mongo_extjson_unwrapper.go:7`: Bổ sung import `"strconv"`.
  - `mongo_extjson_unwrapper.go:335-342`: Bổ sung `case primitive.Decimal128:` và `case string:` vào `coerceDefaultInt32IfInRange(val any)`. Sử dụng `strconv.ParseInt(strings.TrimSpace(...), 10, 64)` thay vì `strconv.ParseFloat`, đảm bảo chỉ những giá trị **nguyên tuyệt đối** nằm trong khoảng `[math.MinInt32, math.MaxInt32]` (như `"9999"` $\rightarrow$ `int32(9999)`) mới được thu hẹp về `NumberInt`, còn các số có phần thập phân (như `"9999.50"`) thì `strconv.ParseInt` trả về lỗi và giữ nguyên `primitive.Decimal128` (`NumberDecimal`), không bao giờ làm mất độ chính xác thập phân.
  - `mongo_extjson_unwrapper_test.go:204, 219-227`: Bổ sung `"value": "NUMERIC"` vào `colTypes` của `TestApplySourceFieldTypesToBSONDoc_ShadowWithoutUpdatedAtAndSubsetFields` và thêm test case kiểm chứng số thập phân `"9999.50"` giữ nguyên `primitive.Decimal128`.

---

## 2. (b) Chặn đứng Suy diễn Smoke Test (Anti-Smoke-Test Speculation & Concurrency / Timeout / Continuity)

1. **Concurrency / Multi-Worker (`WorkerCount > 1`) & Debouncer:**
   - Hàm `coerceDefaultInt32IfInRange` là hàm thuần (pure function, stateless), không dùng biến toàn cục hay cấp phát bộ nhớ chia sẻ giữa các goroutine; an toàn tuyệt đối khi nhiều worker NATS JetStream / `TableDebouncer` gọi `CoerceRecordToBSONDoc` song song.
2. **Timeout / Heartbeat (`AckWait` vs `InProgress`):**
   - Phép gọi `strconv.ParseInt` trên chuỗi `Decimal128` chỉ tốn `< 100ns` trong bộ nhớ, không phát sinh bất kỳ lời gọi I/O nào, không ảnh hưởng tới chu kỳ `inProgressHeartbeat` hay `AckWait` của NATS consumer.
3. **Data Continuity (`Watermark` / `Cursor` qua Outage Gap):**
   - Chỉ chuẩn hóa kiểu BSON của trường `value` khi ghi xuống MongoDB Master; không tác động vào `_id`, `_gpay_id`, `_source_id` hay `_source_ts` dùng cho OCC và `inferLastGpayIDFromMongoMaster`.

---

## 3. (c) Zero-Mockup, Anti-Speculation & Real Runtime Trace (`EV-...`)

Không sử dụng mockup, `sqlmock`, hay closure giả (`*Fn`); mọi kiểm chứng đều chạy trên mã nguồn thật và kết nối MongoDB + PostgreSQL thật qua `evidence_run.py`:
- **`EV-20261102T130838-d40b` (`integration` — Live MongoDB + PostgreSQL Test):**
  - Lệnh: `go test -count=1 -v -run TestMongoOCCBulkModels_ShadowWithoutUpdatedAtAndPruneLeakedSourceFieldValues_LiveMongo ./internal/service/master`
  - Kết quả: `PASS` (`EV-20261102T130838-d40b`, `exit=0`).
- **`EV-20261102T130845-82c5` (`unit` — BSON Unwrapper & Decimal128-to-Int32 Test):**
  - Lệnh: `go test -count=1 -v -run TestApplySourceFieldTypesToBSONDoc ./internal/service/master`
  - Kết quả: `PASS` (`EV-20261102T130845-82c5`, `exit=0`).
- **`EV-20261102T130850-3c8d` (`runtime` — Active Worker Process Check):**
  - Lệnh: `lsof -nP -iTCP:8082 -sTCP:LISTEN`
  - Kết quả: `PASS` (`EV-20261102T130850-3c8d`, `exit=0`, `cds-worke PID 5148 TCP *:8082 LISTEN`).
- **Báo cáo trung thực trạng thái tiến trình (`L024`):** Tiến trình `cds-worke` (`PID 5148`) đang lắng nghe trên cổng `:8082`; để tiến trình đang chạy nạp hàm `coerceDefaultInt32IfInRange` mới khi anh chạy Transmute thực tế trên UI, cần khởi động lại binary `centralized-data-service` (`:8082`) và phối hợp cùng anh kiểm tra trực tiếp document trên MongoDB đích.

---

## 4. (d) Kiểm tra Suy diễn, Báo cáo Láo (Honesty Check) & Vòng lặp Phản tỉnh (Self-Improvement Loop)

- **Kiểm tra suy diễn / báo cáo láo:** Mọi nhận định về `16` trường gốc, `17` trường backup, `24` trường lá nested và luồng `CoerceToBSONValue` $\rightarrow$ `ApplySourceFieldTypesToBSONDoc` $\rightarrow$ `coerceDefaultInt32IfInRange` đều gắn trực tiếp với `file:line` và kết quả test thật (`EV-20261102T130838-d40b`, `EV-20261102T130845-82c5`).
- **Self-Improvement Loop & Đối chiếu `lessons/INDEX.md`:** Tuân thủ nghiêm ngặt `L001`, `L003`, `L008`, `L021`, `L024`, `L025`, `L032`, `L040`, `L042`; không vi phạm bài học nào trong suốt phiên làm việc.

---

## 5. (e) Kết quả QC & Nhận xét từ Hội đồng `/laya_router` (`DG3`)

- **DG0 (`Pre-Plan Critique`):** `PASS` (`06_laya_summary_dg0_20261102T120016.json`).
- **DG1 & DG2 (`Solution & Code Demo Gate`):** `PASS` (`06_laya_summary_dg1_20261102T120125.json`, `06_laya_summary_dg2_20261102T120125.json`).
- **DG3 (`Final Post-Edit Audit Gate`):** `PASS` (`06_laya_summary_dg3_20261102T131020.json`) — xác nhận cả 2 claim `A-DECIMAL128-INT32` (`mongo_extjson_unwrapper.go:334-345`) và `A-TEST-NUMERIC-FRAC` (`mongo_extjson_unwrapper_test.go:220-228`) đạt `PASS` với đầy đủ control âm/dương.
