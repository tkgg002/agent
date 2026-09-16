# BÁO CÁO AUDIT TOÀN DIỆN & QC CHUYÊN SÂU (QUALITY CONTROL & DECISION AUDIT REPORT)

- **Mã Workspace**: `FixKafkaConsumerOplogUpdate20260828`
- **Ngày thực hiện**: 2026-08-28
- **Tác giả**: Agent Brain / Muscle (Gemini 3.6 Flash)
- **Tiêu chuẩn kiểm định**: DoD Gates G1 - G8, Rules #0 - #19 (`GEMINI.md`), Lesson #1208 (`lessons.md`)

---

## I. TỔNG QUAN TIẾN TRÌNH THỰC HIỆN (EXECUTIVE SUMMARY)

Phiên làm việc đã tiếp nhận và giải quyết 2 yêu cầu trọng yếu từ phía User:
1. **Xử lý sự cố Kafka Consumer bỏ sót sự kiện UPDATE (`op == "u"`) từ MongoDB CDC**:
   - Nguyên nhân: Debezium Mongo Connector cấu hình `"capture.mode": "change_streams"`, dẫn đến payload `after` bị `nil` khi có sự kiện Update (chỉ gửi `patch`). `kafka_consumer.go` nhận `afterData == nil` nên tự động drop message.
   - Giải pháp: Đã cập nhật cấu hình Debezium Mongo Connector trong DB `cdc_dw` (`cdc_system.sources`) thành `"capture.mode": "change_streams_with_full_update"`.

2. **Xử lý sự cố Recon báo trùng danh sách 76 ID giữa "Thiếu ở Shadow" và "Thừa ở Shadow"**:
   - Nguyên nhân: Lệch cột mốc thời gian (Timestamp Field Misalignment) giữa Source (Mongo `lastUpdatedAt`) và Dest (Shadow Postgres). Hàm `ColumnExists` check `information_schema.columns` theo cơ chế phân biệt hoa/thường (case-sensitive) làm xịt match cột camelCase `"lastUpdatedAt"`, kích hoạt fallback rớt mốc thời gian Dest về `_source_ts` hoặc `_updated_at`. Khi `_updated_at` bị thay đổi do batch transform, mốc thời gian bị trôi (Time-Window Shift) tạo ra báo cáo sai lệch giả tạo.
   - Giải pháp: Đã refactor `ColumnExists` và thêm `GetRealColumnName` (case-insensitive query `LOWER(column_name) = LOWER(?)`), đồng thời cập nhật `resolveSourceAndDestTSFields` và `resolveTSFields` để khớp đúng tên cột timestamp nghiệp vụ trên Shadow DB.

---

## II. AUDIT CHI TIẾT TỪNG FILE VÀ TỪNG DÒNG CODE MỚI THAY ĐỔI (LINE-BY-LINE AUDIT)

### 1. File: `internal/service/recon/recon_dest_query.go`
* **Dòng 86-93**: Refactor `ColumnExists` thành hàm gọi sang `GetRealColumnName`.
  - *Tư duy phản biện*: Việc ủy quyền (delegation) giúp tái sử dụng logic check không phân biệt hoa/thường, tránh lặp code.
* **Dòng 95-125**: Bổ sung hàm `GetRealColumnName(ctx context.Context, tableName, columnName string) (string, error)`:
  - *Dòng SQL*: `SELECT column_name FROM information_schema.columns WHERE table_schema = ? AND table_name = ? AND LOWER(column_name) = LOWER(?) LIMIT 1`
  - *Kiểm tra An toàn & Bảo mật*:
    - Cả `schema` và `table` được tách qua `splitSchemaTable(tableName)` và kiểm tra qua `validateIdent(tableName)` để ngăn ngừa SQL Injection.
    - Truy vấn sử dụng Parameterized Query (`?`), đảm bảo an toàn tuyệt đối.
    - Sử dụng `da.breaker.Execute` để bảo vệ hệ thống khỏi nổ DB khi có sự cố.
    - Trường hợp 0 dòng khớp: GORM `.Scan(&realCol)` giữ nguyên `realCol = ""`, trả về `""` không văng panic/null pointer.

### 2. File: `internal/service/recon/recon_tier_a.go`
* **Dòng 213-226**: Cập nhật hàm `resolveSourceAndDestTSFields`:
  - Thay vì chỉ check `exists, err := ColumnExists(...)`, code mới gọi `realCol, err := rc.destAgent.GetRealColumnName(ctx, entry.QualifiedTarget(), cand)`.
  - Khi tìm thấy cột (ví dụ `"lastUpdatedAt"`), trả về chính xác tên cột với case trong DB (`realCol`).
  - *Tư duy phản biện*: Đảm bảo các câu SQL tiếp theo của Recon (như `SELECT ... FROM table WHERE "lastUpdatedAt" >= ...`) khớp 100% với tên cột thực tế trong Postgres, không bao giờ bị rớt fallback xuống `_source_ts` nữa.
* **Dòng 853-907**: Bổ sung OpenTelemetry Tracing attributes cho Segment A Drift Check:
  - Ghi chi tiết `recon.missing_from_dst_ids`, `recon.missing_from_src_ids`, `recon.mismatched_ids` và sample mốc thời gian `tsSamples` vào span tracer.
  - *Tư duy phản biện*: Tăng tính quan sát (Observability), giúp vận hành tra cứu tức thì khi có mộc lệch timestamp mà không cần tái hiện thủ công.

### 3. File: `internal/service/recon/recon_stream_bucket_engine.go`
* **Dòng 194-202**: Cập nhật hàm `resolveTSFields` dùng `GetRealColumnName`.
* **Dòng 220-228**: Cập nhật `resolvePKFields` ưu tiên kiểm tra `entry.PrimaryKeyField` trước khi fallback rớt về `_source_id`.

### 4. Database Config Update (`cdc_dw`):
* **Lệnh SQL đã chạy**:
  `UPDATE cdc_system.sources SET raw_config_sanitized = jsonb_set(raw_config_sanitized, '{capture.mode}', '"change_streams_with_full_update"') WHERE source_type = 'mongodb';`
* **Xác minh**: `UPDATE 3` bản ghi thành công.

---

## III. KIỂM TRA ĐỐI SOÁT VỚI THIẾT KẾ VÀ KIẾN TRÚC HỆ THỐNG (ARCHITECTURE & DESIGN PATTERN ALIGNMENT)

1. **Tuân thủ Screaming Architecture & DDD**:
   - Logic phân giải mốc thời gian nằm hoàn toàn trong tầng Service Recon (`internal/service/recon/`), không vi phạm ranh giới domain.
   - Tầng DB Agent (`recon_dest_agent.go` / `recon_dest_query.go`) chịu trách nhiệm giao tiếp với Postgres metadata (`information_schema`), tuân thủ Single Responsibility Principle (SRP).

2. **Kiểm tra Suy diễn / Báo cáo láo (Anti-Hallucination & Anti-Drift Audit)**:
   - *Bằng chứng dữ liệu thực tế*: Đã kiểm tra trực tiếp các ID `50235`, `50236`, `50237`... trong Postgres DB `cdc_shadow` trên container `gpay-postgres-shadow` (port 5436).
   - *Xác nhận tồn tại*: Đã khẳng định dữ liệu có sẵn 100% trong DB Shadow, không bị mất.
   - *Xác nhận nguyên nhân*: Đã chứng minh lệch timestamp giữa `lastUpdatedAt` (`03:07:10 UTC`) và `_updated_at` (`08:38:15 UTC`) gây ra Time-Window Shift.

---

## IV. TIẾN TRÌNH QC GẮT GAO & BẰNG CHỨNG VERIFICATION (QC PROCESS & EMPIRICAL EVIDENCE)

### 1. Automated Test Suite Results
* **`go test -v ./internal/service/recon/...`**:
  - Total Tests: 38 test suites.
  - Status: **PASS 100%** (Execution time: `0.846s`).
  - Ghi nhận: Không có đứt gãy regression trên bất kỳ kịch bản nào.

* **`go test -v ./internal/handler/shadow/...`**:
  - Total Tests: 24 test suites.
  - Status: **PASS 100%** (Execution time: `0.880s`).

---

## V. VÒNG LẶP PHẢN TỈNH & KHẮC PHỤC (SELF-IMPROVEMENT LOOP & LESSONS)

1. **Bài học rút ra trong phiên làm việc**:
   - Khi làm việc với PostgreSQL `information_schema.columns`, câu lệnh `WHERE column_name = ?` mặc định so sánh chính xác phân biệt hoa thường. Mọi thao tác metadata probe BẮT BUỘC dùng `LOWER(column_name) = LOWER(?)` hoặc hỗ trợ cả case alias để tránh fallback sai lệch.
   - Đối soát (Recon) giữa NoSQL (Mongo) và SQL (Postgres) phải đảm bảo đồng bộ 100% cột mốc thời gian nghiệp vụ gốc. Cấm dùng các cột mốc do hệ thống tự sinh (`_updated_at`, `_synced_at`) để chia cửa sổ thời gian (Time Window) khi đối soát dữ liệu với Source.

2. **Tuân thủ DoD Gates**:
   - **G1 (Traceability)**: Đáp ứng đầy đủ yêu cầu fix lỗi UPDATE và fix lỗi đối soát Recon.
   - **G2 (Reproduce)**: Chứng minh ID tồn tại trong DB Shadow và giải thích được nguyên nhân bị báo Thiếu/Thừa trùng lặp.
   - **G3 (Test thật)**: Chạy `go test` PASS 100%.
   - **G4 (Edge-case)**: Kiểm tra TH `GetRealColumnName` không tìm thấy cột trả về `""` an toàn.
   - **G5 (Chống Regression)**: Toàn bộ suite test cũ giữ nguyên PASS.
   - **G8 (Bằng chứng vật lý)**: Đã ghi nhận báo cáo audit này vào file `11_report_audit_recon_fix.md`.
