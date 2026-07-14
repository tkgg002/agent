# Báo cáo overview các file đã phân tích (Audit Overview)

## 1. Thống kê file phân tích

### Gói: `internal/handler/recon/`

| Tên file | Kích thước (Bytes) | Vai trò | Số lượng dòng code |
|---|---|---|---|
| `recon_base_handler.go` | 6635 | Định nghĩa struct Base, constants, interfaces và helper chung | ~243 |
| `recon_check_handler.go` | 12573 | Handler các event check (`recon-check`) từ NATS | ~381 |
| `recon_check_heal_handler.go` | 12568 | Handler event check + tự động heal | ~363 |
| `recon_execute_heal_handler.go` | 11294 | Logic xử lý lệnh execute heal, kéo dữ liệu và sửa đổi Master/Shadow DB | ~308 |
| `recon_heal_fetch.go` | 3731 | Hàm fetch dữ liệu theo danh sách IDs từ MongoDB | ~129 |
| `recon_heal_v4_test.go` | 13109 | Bộ test suite kiểm thử luồng Heal | ~333 |
| `recon_sysops_handler.go` | 16163 | Handler xử lý retry, debezium signals và thao tác sysops | ~378 |

*Tổng số dòng của gói handler:* **~1,735 dòng**.

---

### Gói: `internal/service/recon/`

| Tên file | Kích thước (Bytes) | Vai trò | Số lượng dòng code |
|---|---|---|---|
| `dlq_worker.go` | 12961 | Xử lý dữ liệu đẩy vào DLQ (Dead Letter Queue) | ~340 |
| `recon_alert.go` | 3320 | Quản lý phát cảnh báo drift / lỗi qua Slack/Alertmanager | ~110 |
| `recon_dest_agent.go` | 1964 | Khởi tạo connection và cấu hình database đích (PG Master/Shadow) | ~70 |
| `recon_dest_hash.go` | 6585 | Tính toán hash của bucket / chunk dữ liệu phía database đích | ~180 |
| `recon_dest_legacy.go` | 731 | Chứa các hàm hỗ trợ check chunk hash cũ (legacy) | ~30 |
| `recon_dest_models.go` | 1105 | Struct định nghĩa config và model của Agent đích | ~50 |
| `recon_dest_query.go` | 13032 | Thực thi truy vấn đếm dòng, lấy dữ liệu đích | ~420 |
| `recon_dest_safety.go` | 1453 | Bảo vệ an toàn SQL, kiểm tra SQL identifiers hợp lệ | ~55 |
| `recon_dest_stream.go` | 3002 | Stream danh sách IDs và timestamp từ database đích | ~120 |
| `recon_engine.go` | 9761 | Khởi tạo cấu trúc Engine Recon lõi (ReconCore) | ~350 |
| `recon_engine_run.go` | 11952 | Điều phối các phiên chạy Recon, dọn dẹp orphan runs | ~400 |
| `recon_engine_segment_b.go` | 3050 | Quản lý active bindings và map Segment B | ~110 |
| `recon_hash.go` | 10935 | Logic băm (hash) ID + timestamp (source + dest) | ~340 |
| `recon_heal.go` | 21850 | Điều phối tiến trình heal dữ liệu lỗi lệch | ~620 |
| `recon_heal_utils.go` | 2031 | Hàm phụ trợ build JSON raw, chunking strings | ~90 |
| `recon_legacy.go` | 1227 | Code hash legacy trên MongoDB | ~56 |
| `recon_models.go` | 4652 | Các struct và cấu hình metadata dùng chung trong engine | ~170 |
| `recon_query.go` | 17884 | Thực thi các lệnh truy vấn đếm tài liệu và dữ liệu MongoDB | ~520 |
| `recon_smoke.go` | 24310 | Điều phối smoke check nhanh (Tier 0) | ~720 |
| `recon_source_agent.go` | 5667 | Kết nối MongoDB nguồn và cấu hình circuit breaker | ~200 |
| `recon_stream.go` | 21069 | Logic stream IDs cho Tier A & B từ nguồn/đích | ~820 |
| `recon_tier_a.go` | 43267 | Logic reconcile Tier A (Mongo ↔ Shadow PG) | ~1,250 |
| `recon_tier_b.go` | 29360 | Logic reconcile Tier B (Shadow PG ↔ Master PG) | ~900 |

*Tổng số dòng của gói service:* **~6,800 dòng**.

---

## 2. Kết quả thực thi thay đổi
Đã thực thi các thay đổi mã nguồn (source code) thành công bởi Muscle (Chief Engineer) dựa trên kế hoạch kỹ thuật đã phê duyệt:

### Danh sách các file đã sửa đổi:
1. `internal/handler/recon/recon_base_handler.go`:
   - Import package `centralized-data-service/internal/naming` để đổi `ShadowPrefix` từ hardcoded constant sang dynamic variable (`naming.ShadowSchemaPrefix()`).
   - Thêm helper function `quoteRelation` tương thích PostgreSQL để xử lý SQL Injection.
2. `internal/handler/recon/recon_execute_heal_handler.go`:
   - Sửa lỗi SQL Injection tại hàm `mapGpayToSourceIDs` bằng cách thay thế `%q.%q` bằng `quoteRelation(shadowRel)`.
3. `internal/service/recon/recon_models.go`:
   - Khai báo các context keys struct ẩn (`manualLookbackKey`, `coldLookbackKey`) và các hàm accessor an toàn kiểu để truy cập context (`WithManualLookback`, `GetManualLookback`, `WithColdLookback`, `GetColdLookback`).
4. `internal/service/recon/recon_engine.go`:
   - Thay thế việc đọc context key kiểu string `"cold_lookback"` bằng `GetColdLookback(ctx)`.
5. `internal/service/recon/recon_tier_a.go`:
   - Thay thế việc đọc context key kiểu string `"manual_lookback"` bằng `GetManualLookback(ctx)`.
6. `internal/handler/recon/recon_check_handler.go`:
   - Thay thế việc gán context key kiểu string bằng `servicerecon.WithManualLookback` và `servicerecon.WithColdLookback`.
7. `internal/handler/recon/recon_check_heal_handler.go`:
   - Thay thế việc gán context key kiểu string bằng `servicerecon.WithManualLookback` và `servicerecon.WithColdLookback`.

### Kết quả kiểm thử:
- Lệnh chạy test `go test` bị timeout do cơ chế nhắc nhở phân quyền (permission prompts) của hệ thống không nhận được phản hồi trực tiếp từ User kịp thời.
- Linter quy trình `verify_governance.py` chạy thành công cho workspace `ReconCodebaseAudit20260707` với kết quả **PASSED**.
