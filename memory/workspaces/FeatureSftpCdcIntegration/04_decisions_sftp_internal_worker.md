# 04_decisions_sftp_internal_worker.md — Nhật ký Quyết định Kiến trúc (ADRs)

## Phase: SFTP Internal Polling Worker  
**Ngày:** 2026-08-11

---

## ADR-001: Chọn Hướng B (Internal Polling Worker) thay vì Hướng A (Kafka Connect Plugin)

**Ngày quyết định:** 2026-08-11  
**Trạng thái:** ACCEPTED

**Bối cảnh:**
- Kafka Connect trong hạ tầng không có `io.confluent.connect.sftp.SftpSourceConnector`.
- Hướng A (install plugin mới) đòi hỏi rebuild Docker image Kafka Connect → rủi ro cao, thời gian lâu.

**Quyết định:** Dùng Internal Polling Worker goroutine trong `centralized-data-service`.

**Hệ quả:**
- (+) Không cần thay đổi hạ tầng Kafka Connect.
- (+) Full control over file lifecycle (processed/error).
- (+) Tái dụng toàn bộ consumer pipeline đã có sẵn.
- (-) Thêm 1 goroutine vào centralized-data-service (acceptable).

---

## ADR-002: Dùng `github.com/pkg/sftp` thay vì `golang.org/x/crypto/ssh` trực tiếp

**Ngày quyết định:** 2026-08-11  
**Trạng thái:** ACCEPTED

**Bối cảnh:** `pkg/sftp` là wrapper chuẩn nhất cho SFTP trong Go ecosystem, cung cấp ReadDir, Open, Rename API thân thiện.

**Quyết định:** Dùng `github.com/pkg/sftp` với SSH transport từ `golang.org/x/crypto/ssh`.

---

## ADR-003: `InsecureIgnoreHostKey()` cho local dev, cần FixedHostKey cho Production

**Ngày quyết định:** 2026-08-11  
**Trạng thái:** ACCEPTED (với điều kiện)

**Điều kiện:** Chỉ áp dụng local dev. Production bắt buộc dùng `knownhosts` hoặc `ssh.FixedHostKey()`.

---

## ADR-004: Kafka Writer tạo mới mỗi pollOnce session (không giữ connection lâu dài)

**Ngày quyết định:** 2026-08-11  
**Trạng thái:** ACCEPTED

**Lý do:** Tránh connection leak và timeout khi worker idle giữa các chu kỳ poll. Mỗi lần poll tạo writer mới, flush, close.

---

## ADR-005: Pivot — Huỷ Internal Worker, dùng Kafka Connect với kafka-connect-fs

**Ngày quyết định:** 2026-08-11
**Trạng thái:** ACCEPTED

**Bối cảnh:**
- Hệ thống đang dùng pattern: thêm Kafka Connect plugin JAR (MongoDB, Debezium) vào Dockerfile hoặc /opt/kafka/plugins/.
- Agent đề xuất sai `io.confluent.connect.sftp.SftpSourceConnector` (commercial) mà không đọc Dockerfile trước.
- Đúng hướng: dùng open-source `kafka-connect-fs` (Apache 2.0) — cùng pattern JAR với existing plugins.
- DevOps sẽ cài JAR lên testing Kafka Connect (`10.200.186.203:8083`).

**Quyết định:** Huỷ hướng Internal SFTP Worker goroutine. Dùng `com.github.mmolimar.kafka.connect.fs.FsSourceConnector`.

**Hệ quả:**
- (+) Không cần thêm code Go vào centralized-data-service.
- (+) Đúng architecture pattern với hệ thống hiện tại.
- (+) Kafka Connect quản lý offset, retry, parallelism tự động.
- (-) Phụ thuộc DevOps cài JAR. kafka-connect-fs v1.3.0 (2021) không còn maintain.
- (-) Cần verify compatibility với Kafka 3.x trên testing.

---

## ADR-006: topicPrefix SFTP dùng cdc.sftplocal (không phải sftp.*)

**Ngày quyết định:** 2026-08-11
**Trạng thái:** ACCEPTED

**Lý do:** Thống nhất naming convention với các prefix khác (`cdc.gpaylocal`, `cdc.goopaylocal`).
Topic đầy đủ: `cdc.sftplocal.reconcile.final`.
Removed `cdc.goopay` khỏi local config vì đã có `cdc.goopaylocal`.
