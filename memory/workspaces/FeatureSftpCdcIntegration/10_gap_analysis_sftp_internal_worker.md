# 10_gap_analysis_sftp_internal_worker.md — Phân tích Lỗ hổng Kiến trúc

## Phase: SFTP Internal Polling Worker  
**Ngày:** 2026-08-11

---

## Gap 1: Luồng SFTP bị ĐỨT ĐOẠN tại tầng Publisher [CRITICAL]

**Phạm vi:** `centralized-data-service` → Kafka pipeline  
**Phát hiện:** 2026-08-11 lúc audit code sau error 400 Kafka Connect.  
**Nguyên nhân gốc rễ:**
- Plugin `io.confluent.connect.sftp.SftpSourceConnector` là sản phẩm thương mại của Confluent, không được cài đặt trong Kafka Connect của hệ thống.
- Toàn bộ code phía consumer được viết trước khi verify plugin tồn tại.

**Trạng thái fix:** Đang implement (SFTP Internal Worker).

---

## Gap 2: `SFTPWorkerConfig` chưa có trong `AppConfig` [HIGH]

**Phạm vi:** `config/config.go`  
**Nguyên nhân:** Chưa implement.  
**Trạng thái fix:** Task A4, A5.

---

## Gap 3: `config-local.yml` thiếu section `sftpWorker` [HIGH]

**Phạm vi:** `config/config-local.yml`  
**Nguyên nhân:** Chưa implement.  
**Trạng thái fix:** Task A6.

---

## Gap 4: `server_setup.go` chưa wire SFTPPollingWorker [HIGH]

**Phạm vi:** `internal/server/server_setup.go`  
**Nguyên nhân:** Chưa implement.  
**Trạng thái fix:** Task C2, C3.

---

## Gap 5: `sftp_worker.go` chưa tồn tại [CRITICAL]

**Phạm vi:** `internal/handler/shadow/`  
**Nguyên nhân:** Chưa implement.  
**Trạng thái fix:** Task B1-B11.

---

## Đã Ổn (No Gap)

| Component | Trạng thái |
|:---|:---:|
| `sftp_adapter.go` — Convert flat JSON → CDCEvent | ✅ Đúng |
| `event_handler.go` — HandleRaw SFTP routing | ✅ Đúng |
| `topic_helper.go` — SFTP bypass debeziumTables filter | ✅ Đúng |
| `cdc-cms-web` SourceConnectors form SFTP UI | ✅ Đúng |
| `cdc-cms-service` system_connectors_handler SFTP | ✅ Đúng |
| Migrations 073, 074 schema constraints | ✅ Đúng |
| Seed SQL sftp_reconcile_final | ✅ Đúng |
| Docker SFTP Host container | ✅ Running |
