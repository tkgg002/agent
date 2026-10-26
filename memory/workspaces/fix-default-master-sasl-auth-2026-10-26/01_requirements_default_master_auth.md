# 01 Requirements: Khắc phục lỗi `open master db connection for "default_master": failed SASL auth for user "root" database=cdc_cms_database` (SQLSTATE 28P01)

tier: T2
suffix: default_master_auth
date: 2026-10-26

## 1. Triệu chứng lỗi từ Runtime Log (Testing/Dev)
- Lúc `09:35:09 – 09:36:26 26/10/2026`, `kafka-consumer-hook` thực thi `transmute` và `transmute_sub` trên các bảng `shadow_bidv_connector_service.bank_requests -> master_bidv_connector_service.bank_requests` và `shadow_banvietbank_connector_service.bank_requests -> master_banvietbank_connector_service.bank_requests` liên tục báo lỗi:
  `get master db: open master db connection for "default_master": gorm open: failed to connect to user=root database=cdc_cms_database: 10.200.186.80:5432 (10.200.186.80): failed SASL auth: FATAL: password authentication failed for user "root" (SQLSTATE 28P01)`
- Hệ quả: Toàn bộ bản tin realtime bị đẩy vào `transmute_poison_pill` (`poison pill isolated`).

## 2. Yêu cầu
- Điều tra chính xác Root Cause theo `file:line` trên `centralized-data-service` và `cdc-cms-service` (kể cả dữ liệu đã bị ghi vào bảng `cdc_system.connection_registry` cho `default_master`).
- Phản biện qua cổng `DG0` và thiết kế giải pháp triệt để tại `09_tasks_solution_default_master_auth.md` (`DG1/DG2 PASS`), trình User `APPROVE` trước khi sửa code.

## 3. Yêu cầu bổ sung (`TASK-06` — NATS Control Center trên Testing lúc `10:56:54 26/10/2026`)
- **Triệu chứng trên Testing:**
  - `MISSING_CORE_STREAM[CDC_TRANSMUTE]`, `MISSING_CORE_STREAM[SCHEMA_DRIFT]`, `MISSING_CORE_STREAM[SCHEMA_CONFIG]` (`Stream Governance: 1/4 Core Synced`, chỉ có `CDC_EVENTS` với `512.00 MB`, `4 subjects: cdc.gpay.>, cdc.goopay.>, +2`, `5 consumers (4/5 Active Bound)`).
  - `MISSING_COMMAND_LISTENER`: `0/19 Ready` (`Connections: 1 | Subs: 274`, `JS Storage: 0 B / 1.00 GB`).
- **Yêu cầu:** Chỉ rõ vì sao các lần sửa trước vẫn để sót lỗi trên Testing kèm `file:line` chứng minh, lên thiết kế giải pháp dứt điểm (không viết/chạy test, không dùng `DeleteStream` tự động, không nhồi nhét God-file), kiểm định qua `DG0` & `DG1/DG2`, trình User `APPROVE`.

## 4. Yêu cầu bổ sung (`TASK-07` — Khắc phục triệt để lỗi Recon/Heal Segment B `Shadow -> Master` lúc `10:54:43 26/10/2026`)
- **Triệu chứng trên Testing:**
  - Báo cáo `reconB` (`Shadow -> Master`, cửa sổ `10:49 19/10/2026 - 10:49 26/10/2026`) có `Nguồn = 3,095`, `Đích = 3,082` (lệch thực tế `-13` dòng thiếu), nhưng báo `Thiếu = 1,693`, `Lệch = 0`, `Thừa = 1,680` (`1,693 - 1,680 = 13`).
- **Yêu cầu (User đã `APPROVE` `"thôi làm đi"`):**
  - Tự động chuẩn hóa/backfill `transform_spec` (`{"pk": "<primary_key_field>"}`) từ `cdc_system.source_object_registry` cho các `cdc_system.master_binding` hiện hữu đang có `transform_spec = '{}'` (giữ nguyên nguyên tắc `L014` chỉ đọc `pk` từ `transform_spec`).
  - Gỡ bỏ nhánh tự bịa `_gpay_id = deterministicSourceGpayID` (SHA-256) trong `RunEvents` và gỡ bỏ mảng chắp vá `staleLegacyGpayIDs` trong `transmuter.go`.
  - Trong `bulkUpsertMaster` (`transmuter.go`), khi `conflictTarget != "_gpay_id"`, xóa nguyên tử các dòng trùng khóa nghiệp vụ (`conflictTarget`) nhưng lệch `_gpay_id` (`DELETE FROM <qt> WHERE <conflictTarget> IN (?) AND "_gpay_id" NOT IN (?)`) trước khi `INSERT ... ON CONFLICT ("_gpay_id") DO UPDATE` trong cùng transaction để triệt tiêu hoàn toàn lỗi `SQLSTATE 23505` và `SQLSTATE 42P10` đồng thời tự dọn sạch các dòng nhân đôi/lệch `_gpay_id`.
  - Bật mặc định `pruneMissingSrc` trên `ExecuteHealModal.tsx` đối với Segment B (`shadow_master`) khi `orphan_count > 0`.
  - Tuyệt đối không viết/sửa/chạy test mockup; kiểm chứng bằng `go build ./...`, `npm run build` và trace runtime value thật trên PostgreSQL.

