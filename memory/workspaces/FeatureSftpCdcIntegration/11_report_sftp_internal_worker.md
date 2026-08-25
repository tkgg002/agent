# 11_report_sftp_internal_worker.md — Báo cáo Thay đổi

## Phase: SFTP Internal Polling Worker  
**Ngày:** 2026-08-11 | **Trạng thái:** ⏳ Chưa implement (Pending Muscle)

---

## Files Sẽ Thay đổi

| # | File | Action | Dòng ước tính | Mô tả |
|:---|:---|:---:|:---:|:---|
| 1 | `internal/handler/shadow/sftp_worker.go` | NEW | ~180 dòng | SFTPPollingWorker struct, Start, Stop, pollOnce, connect, processFile, parseCSVRows, publishRows, moveFile |
| 2 | `internal/handler/shadow/sftp_worker_test.go` | NEW | ~80 dòng | Unit tests cho parseCSVRows, filePattern |
| 3 | `config/config.go` | MODIFY | +15 dòng | Thêm SFTPWorkerConfig struct + field vào AppConfig |
| 4 | `config/config-local.yml` | MODIFY | +12 dòng | Thêm sftpWorker block |
| 5 | `internal/server/server_setup.go` | MODIFY | +8 dòng | Wire SFTPPollingWorker vào RegisterOnStart/Stop |

---

## Total Estimated Lines
- Thêm mới: ~275 dòng
- Sửa: ~35 dòng
- Tổng delta: ~310 dòng

---

## Sẽ được cập nhật sau khi Muscle hoàn tất implementation.

---

## Report Session 2026-08-11 10:00 — 10:55

### Tổng quan thay đổi

**Pivot kiến trúc:** Internal Worker → Kafka Connect (kafka-connect-fs)

### Files đã thay đổi

| File | Số dòng thay đổi | Mô tả |
|:---|:---:|:---|
| `centralized-data-service/config/config-local.yml` | ~3 dòng | Bỏ cdc.goopay, đổi sftp → cdc.sftplocal |
| `centralized-data-service/internal/handler/shadow/event_handler.go` | ~15 dòng | isSFTP detection + db/table parsing normalize |
| `centralized-data-service/internal/handler/shadow/topic_helper.go` | ~3 dòng | isSFTPTopic: HasPrefix → Contains |
| `cdc-cms-service/internal/api/source/system_connectors_handler.go` | ~35 dòng | parseFingerprint FsSourceConnector + extractCredentials fs.uris |
| `cdc-cms-web/src/pages/SourceConnectors.tsx` | ~80 dòng | connector class, config keys, form UI, ConnectionFormValues |
| `agent/memory/global/lessons.md` | +7 dòng | Lesson: đọc Dockerfile trước khi đề xuất connector |

### Key changes detail

1. **connector.class**: `io.confluent.connect.sftp.SftpSourceConnector` → `com.github.mmolimar.kafka.connect.fs.FsSourceConnector`
2. **Config keys fix**: `policy.sleepy.wait_ms` → `policy.sleepy.sleep`; `file_reader.csv.header` → `file_reader.delimited.settings.header`
3. **Form UI**: thay host/port/username/password riêng lẻ bằng 1 field `fs.uris` (sftp://user:pass@host:port/path)
4. **isSFTP logic**: `HasPrefix("sftp.")` → `Contains("sftplocal")` để nhận `cdc.sftplocal.*`
5. **db/table parse**: tìm segment chứa "sftp", normalize → db="sftp", table=parts[sftpIdx+1:]

### Build verification

- centralized-data-service: go build ./internal/... ./cmd/... → exit 0
- cdc-cms-service: go build ./internal/... ./cmd/... → exit 0
- cdc-cms-web: npx tsc --noEmit → exit 0
