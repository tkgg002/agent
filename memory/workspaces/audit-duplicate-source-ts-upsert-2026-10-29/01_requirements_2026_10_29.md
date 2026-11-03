---
tier: T2
workspace: audit-duplicate-source-ts-upsert-2026-10-29
suffix: 2026_10_29
mode: standard
created_at: 2026-10-29T17:26:15+07:00
---

# 01. Requirements: Audit Lỗi `column "_source_ts" specified more than once (SQLSTATE 42701)` Khi Flush/Transform Shadow (`transaction_history` & `cdc_transaction_history_payload`)

## 1. Bối cảnh & Yêu cầu của Người dùng (User Request)
- Người dùng cung cấp bảng trạng thái chạy Transform/Snapshot trên giao diện với 2 job bị lỗi `SQLSTATE 42701`:
  1. **Job 1 (`17:21:55 29/10/2026`):**
     - **Source:** `transaction_history` / `testcoretrans` (`core-trans-proxy-history-service`)
     - **Shadow:** `transaction_history` (`shadow_testcoretrans`)
     - **Status:** `error`
     - **Rows Processed / Progress:** `15,000` (`15,000 / 249,106` — `6.02%`)
     - **Last Seen ID:** `66f01c1f56f3dfef79f486d9`
     - **Trace ID:** `44c6d5…e542`
     - **Finish Time / Error:** `flush failed: in-place retry exhausted: batch upsert chunk failed: ERROR: column "_source_ts" specified more than once (SQLSTATE 42701) (fallback persisted 0 rows)`
  2. **Job 2 (`17:21:53 29/10/2026`):**
     - **Source:** `cdc_transaction_history_payload` / `testcoretrans` (`core-trans-proxy-history-service`)
     - **Shadow:** `cdc_transaction_history_payload` (`shadow_testcoretrans`)
     - **Status:** `error`
     - **Rows Processed / Progress:** `30,000` (`30,000 / 250,363` — `11.98%`)
     - **Last Seen ID:** `6724b7c6d3902831efa286ec`
     - **Trace ID:** `042e8b…96e1`
     - **Finish Time / Error:** `flush failed: in-place retry exhausted: batch upsert chunk failed: ERROR: column "_source_ts" specified more than once (SQLSTATE 42701) (fallback persisted 0 rows)`
- Yêu cầu: **"audit cho anh"** — điều tra toàn diện nguyên nhân gốc rễ gây ra lỗi `ERROR: column "_source_ts" specified more than once (SQLSTATE 42701) (fallback persisted 0 rows)` trên cả 2 bảng `transaction_history` và `cdc_transaction_history_payload` thuộc `shadow_testcoretrans`, vì sao xử lý được `15,000` / `30,000` dòng rồi mới nổ hoặc nổ ở bước nào, vì sao `fallback persisted 0 rows`, và đề xuất 1 phương án giải quyết triệt để kèm Code Demo (Propose-Only chờ `APPROVE`).

## 2. Ràng buộc Thực thi Trích xuất từ Bài học (Technical Constraints from Lessons)
- **L005 (Propose-Only Discipline & Explicit Approval Gate):** Yêu cầu là `audit cho anh` -> giữ chế độ **100% Read-Only (Propose-Only)** trên mã nguồn, lập `09_tasks_solution_2026_10_29.md` qua `DG0` & `DG1`, hiển thị trực tiếp **Code Demo** trên khung chat và dừng chờ lệnh `APPROVE` tường minh trước khi chạm vào source code.
- **L008 (Single-Responsibility Module Split & Line-Budget Guard):** Tuyệt đối không nhồi thêm hàm lớn vào God-files; đảm bảo tách hàm/module đúng trách nhiệm đơn và đúng layer kiến trúc.
- **L011 (Pre-Plan Adversarial Critique Wave DG0):** Chạy `laya_gate.py run --stage dg0` đối chiếu giả định với `file:line` và dữ liệu DB thực tế trước khi lập giải pháp ở `S3`.
- **L018 (Zero-Mockup Evidence & Real Runtime Trace Verification):** Truy vết code thật + cấu hình mapping/binding trong PostgreSQL thật (nếu có quyền đọc read-only) để xác minh chính xác cấu hình cột của `shadow_testcoretrans.transaction_history` và `shadow_testcoretrans.cdc_transaction_history_payload`.
- **L020 (End-to-End Callchain Tracing & Writer Normalization):** Truy vết toàn trình từ luồng Transform/Snapshot/Flush (`flush failed: in-place retry exhausted: batch upsert chunk failed`) -> hàm sinh câu lệnh SQL `INSERT INTO ... ON CONFLICT ... DO UPDATE` -> cơ chế fallback single-row (`fallback persisted 0 rows`) -> cấu hình `mapping_rules` / system metadata columns (`_source_ts`, `_source_id`, `_op`, `_raw_data`, `_ingested_at`).
- **L022 (Direct-Answer Summary First & 1-to-1 Finding Mapping):** Mở đầu phản hồi bằng khối **Direct Summary** 2–4 câu trả lời trực diện nguyên nhân; ánh xạ giải pháp 1-1 theo từng phát hiện `F1..Fn`.
- **L031 (Zero Runtime DDL & No Full-Table Scans on Large Databases):** Cấm chạy DDL trong luồng runtime worker và cấm sửa dữ liệu DB trực tiếp (`No Cheat DB`).
- **L034 (Transaction Aborted Isolation & GORM SQL Quirks):** Kiểm tra kỹ luồng batch upsert và fallback single-row để xem vì sao `fallback persisted 0 rows` (do cùng lỗi duplicate column `_source_ts` ở từng dòng hay do transaction state).
- **L043 (Spec-Driven PK & Timestamp Field Resolution):** Kiểm tra mối liên hệ giữa cấu hình `timestamp_field` / `mapping_rules` với các cột hệ thống `_source_ts` của bảng Shadow/Master.

## 3. Yêu cầu Bổ sung (2026-10-30T10:21+07:00): Lỗi Trùng Đăng ký Khi Xóa Tạo Lại (`"Cặp (source DB + source table + target table) đã được đăng ký rồi"`)
- **Báo cáo của User:** `"Cặp (source DB + source table + target table) đã được đăng ký rồi. đã xoá tạo lại, lại nhảy ra lỗi này, cái này tao nhớ tao đã fix từ rất lâu rồi"`
- **Ràng buộc Thực thi Bổ sung:**
  - **L031 (Zero DB CLI Or Container Exec When DB Access Forbidden):** Tuyệt đối không chạy bất kỳ lệnh `docker exec ... psql` / DB client nào chạm vào DB; chỉ điều tra 100% qua source code tĩnh và xử lý triệt để bằng code nghiệp vụ chuẩn.
  - **L033 (Full Multi-Tenant Identity Tuple & 1-N-M Lineage SQL Resolution):** Mọi câu lệnh JOIN / DELETE / dọn mồ côi giữa `source_object_registry`, `shadow_binding`, và `cdc_table_registry` bắt buộc phải định danh theo đủ bộ khóa `(source_connection_id, source_db, source_table, target_table)` thay vì chỉ lọc theo `target_table` / `shadow_table` trần.
  - **L005 (Propose-Only Discipline & Explicit Approval Gate):** Giữ nguyên chế độ **100% Propose-Only**, trình bày nguyên nhân + giải pháp + **Code Demo** trực tiếp trên khung chat và dừng chờ lệnh `APPROVE` tường minh trước khi chạm vào source code.

## 4. Yêu cầu Bổ sung (2026-10-30T10:30+07:00): Vì Sao Vừa Scan Bảng `cdc_transaction_history_payload` Vẫn Hiện Đủ 4 Cột Hệ Thống (`_deleted`, `_gpay_id`, `_source_ts`, `_updated_at` — `mapping | pending | in shadow`)?
- **Báo cáo của User:** Sau khi xoá/tạo lại và bấm Scan trên `cdc_transaction_history_payload`, màn hình Mapping Rules vẫn hiện 4 dòng `_deleted`, `_gpay_id`, `_source_ts`, `_updated_at` ở trạng thái `mapping | pending | in shadow` bên cạnh 9 cột nghiệp vụ (`createdAt`..`updatedAt` ở trạng thái `mapping | approved | in shadow`).
- **Ràng buộc Thực thi Bổ sung:**
  - **L021 (Honest Process State, Environment Lock & Physical Metrics):** Kiểm tra minh bạch `PID` và `STARTED` (`ps -ww -o pid,ppid,lstart,command`, `lsof -iTCP:8080-8095 -sTCP:LISTEN`) của các tiến trình đang giữ cổng `:8082` (`centralized-data-service`) và `:8083` (`cdc-cms-service`), nhận lỗi thẳng thắn nếu daemon worker chưa được nạp lại binary mới.
  - **L020 (End-to-End Callchain Tracing & Writer Normalization):** Rà soát toàn bộ đường đi của lệnh `Scan` và `GET /api/mapping-rules` (`mapping_rule_repo_gorm.go`, `discover_handler_sql.go`, `source_object_v2_sync.go`, `mapping_utils.go`) để chặn cứng ở cả đầu ghi, đầu đọc và tự động dọn sạch các rule cột hệ thống tồn dư khi Scan lại mà không cần can thiệp DB thủ công (`L031`).
