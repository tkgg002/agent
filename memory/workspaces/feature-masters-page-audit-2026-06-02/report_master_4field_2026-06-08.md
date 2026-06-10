# report_master_4field_2026-06-08.md — Tinh gọn Master DB còn 4 system fields + OCC `_source_ts`

> **Agent**: Muscle:Claude-Opus-4.8 | 2026-06-08 | Phạm vi: Shadow→Master (KHÔNG đụng Source→Shadow). KHÔNG commit/push.

## 1. Yêu cầu (User vai kiến trúc)
Master DB CHỈ giữ **4 trường hệ thống**: `_gpay_id` (PK toàn cục + conflict key), `_source_ts` (INT8 epoch-ms = cột mốc OCC, thay `_version`), `_deleted` (soft-delete), `_updated_at` (thời điểm ghi thực tế vào Master). Bỏ rác kỹ thuật. OCC dùng `_source_ts >=` (event cùng mili-giây tới sau vẫn được đè — an toàn nhờ In-Order Processing ở WK-E1 / Debezium partition-by-PK).

## 2. Thiết kế thực thi (1 giải pháp)
- **DDL master**: cols = 4 trường trên. Bỏ `_source_id/_source/_synced_at/_version/_hash/_created_at` (+ `_raw_data` theo I2 — master là bảng nghiệp vụ).
- **Conflict key**: bỏ `_source_id` ⇒ `ON CONFLICT (_gpay_id)`. `_gpay_id` PHẢI deterministic per source-row (trước là Sonyflake ngẫu nhiên mỗi lần → sẽ tạo trùng). → `deterministicGpayID(shadowGpayID, keySuffix)`: copy_1_to_1 dùng thẳng shadow `_gpay_id` (Sonyflake, unique+ổn định); flatten dùng FNV-1a 64 của (shadow gpay_id + "#idx") → int63 dương.
- **OCC**: `ON CONFLICT (_gpay_id) DO UPDATE SET <cols=EXCLUDED>, _updated_at=NOW() WHERE COALESCE(EXCLUDED._source_ts,0) >= COALESCE(master._source_ts,0)`. Bỏ điều kiện `_hash IS DISTINCT`. `_updated_at` = NOW() (không từ nguồn); INSERT dùng DEFAULT NOW().
- **Index**: bỏ `ux_<t>_source_id` (PK `_gpay_id` đã unique) + `ix_<t>_created_at`; thêm `ix_<t>_source_ts` (delta/OCC checkpoint); giữ `ix_<t>_updated_at`.
- **Bảng master CŨ**: Apply emit `ALTER TABLE ... DROP COLUMN IF EXISTS` 7 cột (idempotent, trong tx có lock_timeout 5s/statement_timeout 30s). Bảng đã có data dùng `_gpay_id` ngẫu nhiên cũ ⇒ cần **TRUNCATE + re-sync 1 lần** để conflict-key khớp lại (master = view vật chất hoá của shadow, regenerate an toàn).

## 3. Files thực tế đã sửa + LOC (git diff --stat)
- `internal/service/master_ddl_generator.go` — **129 dòng đổi** (+~, cols 4 trường, index, DROP COLUMN cleanup).
- `internal/service/transmuter.go` — **371 dòng đổi** (406 insert/94 delete tổng cả 2 file; emit loop, upsert OCC, deterministicGpayID thay computeMasterHash, imports).
- `internal/service/transmuter_gpayid_test.go` — **file mới (untracked)** ~45 dòng (5 unit test deterministicGpayID).

## 4. Verify (exercise-driven, Rule 16)
| Hạng mục | Kết quả |
|---|---|
| Build/vet | `go build ./...`=0, `go vet` sạch (file đã sửa) |
| Unit test | `deterministicGpayID` 5/5 PASS (copy=shadow id; flatten ổn định+khác nhau+int63 dương); unwrap/coerce/cache PASS |
| Cột master (b3) | TRƯỚC 10 system → **SAU đúng 4** (`_deleted,_gpay_id,_source_ts,_updated_at`) + business (`__v,_id,...`); **hết `_raw_data`** |
| Index (b3) | `b3_pkey, ix_b3_source_ts, ix_b3_updated_at, ix_b3_totalRecords` — hết `ux_source_id`/`ix_created_at` |
| Sync | truncate+sync → 457 rows, **457 distinct `_gpay_id`** (deterministic key, không trùng) |
| **Idempotent re-sync** | re-sync → count **457→457** (upsert in-place, không sinh dòng trùng) |
| **OCC out-of-order** | bump 1 row `_source_ts=9999999999999`+sentinel → re-sync (shadow ts nhỏ hơn) → **giữ nguyên sentinel** (OCC chặn event cũ đè) |
| Worker | restart no-fatal (PID 71111), confirm trước khi exercise (lesson zombie-process) |

## 5. Vấn đề ORTHOGONAL (KHÔNG do task này — flag riêng)
`type_errors=393` ở field nghiệp vụ `totalRecords` (giá trị shadow = number `0` cho 393 row / absent 64) → lưu null. Code `extractColumns`/`ValidateValue`/`coerceForColumn` **KHÔNG bị đụng** trong task 4-field này ⇒ không phải do thay đổi này gây ra; nghi shadow `export_jobs_4` bị re-ingest trong 3 ngày gap đổi `totalRecords`. **Chưa diagnose đến cùng** (int64(0) qua Validate("BIGINT") đáng lẽ PASS) — cần phiên điều tra riêng nếu cần đúng field totalRecords. KHÔNG gộp vào scope task này.

## 6. Ràng buộc tuân thủ
- KHÔNG đụng Source→Shadow (chỉ master DDL + master upsert).
- KHÔNG cheat DB (recreate/sync qua DDL Apply + NATS transmute thật; truncate là bước migration hợp lệ cho bảng materialized).
- KHÔNG commit/push.

## 7. Lưu ý vận hành cho User
- Master tạo MỚI: tự có 4-field. Master CŨ đã có data: re-Apply (drop cột cũ) **+ truncate + re-sync** 1 lần để `_gpay_id` deterministic khớp (đã làm cho b3).
- OCC `>=` đánh đổi: full re-scan ghi đè lại mọi row (không còn `_hash` no-op) — đúng thiết kế anh chọn; chấp nhận cho đúng-ngữ-nghĩa same-ms.
