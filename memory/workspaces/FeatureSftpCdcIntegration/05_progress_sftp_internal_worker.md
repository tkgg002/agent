# 05_progress_sftp_internal_worker.md — Nhật ký Tiến độ (Audit Log — Append ONLY)

## Phase: SFTP Internal Polling Worker  
**Ghi chú:** File này là Audit Log bất biến. TUYỆT ĐỐI không xóa/sửa nội dung cũ. Chỉ Append.

---

- [2026-08-11 09:52:00] [Agent:Gemini-3.6-Flash] Brain bắt đầu khởi tạo Workspace & Full Doc Set cho Phase SFTP Internal Polling Worker theo yêu cầu của User.
- [2026-08-11 09:53:00] [Agent:Gemini-3.6-Flash] Đã tạo 00_context_sftp_internal_worker.md — Scope & Components.
- [2026-08-11 09:53:30] [Agent:Gemini-3.6-Flash] Đã tạo 01_requirements_sftp_internal_worker.md — BR + TR + DoD.
- [2026-08-11 09:53:50] [Agent:Gemini-3.6-Flash] Đã tạo 02_plan_sftp_internal_worker.md — Roadmap cao tầng + Risk.
- [2026-08-11 09:54:10] [Agent:Gemini-3.6-Flash] Đã tạo 03_implementation_sftp_internal_worker.md — Technical Design chi tiết (struct, flow, Kafka, SSH, CSV).
- [2026-08-11 09:54:30] [Agent:Gemini-3.6-Flash] Đã tạo 04_decisions_sftp_internal_worker.md — ADR-001 đến ADR-004.
- [2026-08-11 09:54:45] [Agent:Gemini-3.6-Flash] Đang tạo các file còn lại: 05 đến 13.
- [2026-08-11 09:57:20] [Agent:Gemini-3.6-Flash] HOÀN THÀNH khởi tạo Full Doc Set 13 file cho Phase SFTP Internal Worker. Sẵn sàng chờ User APPROVE để Muscle thực thi.
- [2026-08-11 10:33:00] [Agent:Gemini-3.6-Flash] User APPROVE. Bắt đầu thực thi G1+G2+G3+Backend.
- [2026-08-11 10:33:35] [Agent:Gemini-3.6-Flash] G1 DONE: thêm `- sftp` vào `kafka.topicPrefix` trong `config-local.yml`.
- [2026-08-11 10:34:14] [Agent:Gemini-3.6-Flash] G2+G3 DONE: sửa `SourceConnectors.tsx` — connector.class → `com.github.mmolimar.kafka.connect.fs.FsSourceConnector`, config keys → fs.uris/policy.*/file_reader.*, parseConnectionSeed đọc fs.uris, detectDbKind nhận FsSourceConnector.
- [2026-08-11 10:34:49] [Agent:Gemini-3.6-Flash] Backend DONE: sửa `system_connectors_handler.go` — parseFingerprint nhận FsSourceConnector class, extractCredentialsAsOptions đọc credentials từ fs.uris.
- [2026-08-11 10:43:10] [Agent:Gemini-3.6-Flash] BUILD PASS: cdc-cms-service (exit 0) + centralized-data-service (exit 0). Tất cả thay đổi hợp lệ.

---

### Session 2026-08-11 10:43 — Pivot sang Kafka Connect (kafka-connect-fs)

**Quyết định kiến trúc (ADR-005):** Huỷ hướng Internal SFTP Worker. Chọn Kafka Connect với plugin open-source `kafka-connect-fs` (Apache 2.0) — đúng pattern với hạ tầng đang dùng (Debezium JAR).

**Root Cause lesson:** Em đề xuất `io.confluent.connect.sftp.SftpSourceConnector` (commercial) mà không đọc Dockerfile.connect trước → đã ghi lesson vào `lessons.md`.

- [2026-08-11 10:33:35] [Agent:Gemini-3.6-Flash] **config-local.yml**: Thêm `- sftp` vào `topicPrefix`.
- [2026-08-11 10:43:00] [Agent:Gemini-3.6-Flash] **config-local.yml**: Bỏ `- cdc.goopay`, đổi `- sftp` → `- cdc.sftplocal`.
- [2026-08-11 10:43:00] [Agent:Gemini-3.6-Flash] **SourceConnectors.tsx** `TOPIC_PREFIX_BY_DB.sftp`: `sftp.reconcile` → `cdc.sftplocal.reconcile.final`.
- [2026-08-11 10:43:00] [Agent:Gemini-3.6-Flash] **event_handler.go**: `isSFTP` detection nhận `cdc.sftplocal.*`; db/table parsing normalize → `db="sftp"`, `table=parts-after-sftp_*`.
- [2026-08-11 10:43:00] [Agent:Gemini-3.6-Flash] **topic_helper.go**: `isSFTPTopic` → `strings.Contains(topic, "sftp")`.
- [2026-08-11 10:45:41] [Agent:Gemini-3.6-Flash] BUILD PASS: centralized-data-service (exit 0).
- [2026-08-11 10:50:42] [Agent:Gemini-3.6-Flash] **SourceConnectors.tsx** — Fix config keys đúng spec kafka-connect-fs:
  - `policy.sleepy.wait_ms` → `policy.sleepy.sleep`
  - `file_reader.csv.header` → `file_reader.delimited.settings.header`
  - Thêm `policy.recursive: false`
  - `ConnectionFormValues`: thêm `fsUris`, `sleepMs`; bỏ `inputPath`
  - Form UI: field `fsUris` (sftp URI nhúng credentials), ẩn host/port/username/password, ẩn database, label topic → "Kafka Topic (topic)".
  - `parseConnectionSeed` đọc `policy.sleepy.sleep` và `policy.regexp`.
- [2026-08-11 10:51:33] [Agent:Gemini-3.6-Flash] TypeScript check PASS (exit 0). Toàn bộ thay đổi hợp lệ.

---
## [2026-08-11 11:18] QC Round 2 — Phản tỉnh & Fix

### Bugs tìm thêm sau QC lần 2 (đọc code thực tế)

**BUG-4 [FIXED] — FilterSafeConfig không sanitize fs.uris**
- File: `cdc-cms-service/internal/infra/http/kafka_connect.go`
- Vấn đề: `fs.uris = sftp://user:pass@host:port/path` đi thẳng vào `raw_config_sanitized` DB — lộ credentials
- Fix: Thêm `sanitizeSFTPURI()` helper + branch `lk == "fs.uris"` trong `FilterSafeConfig`
- Pattern: giống hệt `sanitizeMongoURI()` đã có trong codebase

**BUG-5 [FIXED] — sleepMs InputNumber→string type mismatch**
- File: `cdc-cms-web/src/pages/SourceConnectors.tsx` L305
- Vấn đề: `InputNumber` trả về `number`, nhưng Kafka Connect REST API chỉ nhận `string` values
- Fix: `String(values.sleepMs ?? 30000)` thay vì `values.sleepMs || '30000'`

### Bugs QC lần 1 (đã fix trước đó)
- BUG-1: topicPrefix fallback cfg["topic"] — FIXED
- BUG-2: isSFTP HasPrefix narrow — FIXED
- BUG-3: isSFTPTopic HasPrefix narrow — FIXED

### Build verification sau toàn bộ fixes
- centralized-data-service: go build → exit 0 ✅
- cdc-cms-service: go build → exit 0 ✅
- cdc-cms-web: npx tsc --noEmit → exit 0 ✅

### Còn lại
- config.yml (testing env): cần thêm `cdc.sftplocal` prefix thủ công trước khi deploy
- DevOps cài kafka-connect-fs JAR v1.3.0

- [2026-08-14 09:37:00] [Agent:Gemini-3.5-Flash] Đã tạo file 08_tasks_sre_sftp.md chứa trạng thái SRE-1454->1459 và các task vận hành/DevOps lớn còn thiếu.
- [2026-08-14 09:38:00] [Agent:Gemini-3.5-Flash] Đã cập nhật chi tiết Description & Todo List của SRE-1454->1459 và các task lớn còn thiếu vào file 08_tasks_sre_sftp.md theo yêu cầu của User.
- [2026-08-14 09:52:00] [Agent:Gemini-3.5-Flash] Đã bổ sung các todo con nghiệp vụ bắn NATS và nhận/xử lý kết quả phản hồi của CMS-API vào SRE-1456 trong file 08_tasks_sre_sftp.md.
- [2026-08-14 09:57:00] [Agent:Gemini-3.5-Flash] Đã bổ sung các todo con nghiệp vụ custom rewind offset và luồng custom snapshot của CDC-WORKER vào SRE-1457 trong file 08_tasks_sre_sftp.md.
- [2026-08-14 10:00:00] [Agent:Gemini-3.5-Flash] Đã cập nhật đúng logic thực tế của SRE-1457 (disable RewindTopicOffset, thay bằng luồng Resume/Restart connector qua API Connect) trong file 08_tasks_sre_sftp.md.





