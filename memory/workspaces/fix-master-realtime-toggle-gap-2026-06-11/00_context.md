# 00_context — Fix: master tắt→bật realtime để lại gap record thiếu vĩnh viễn

**Ngày**: 2026-06-11 · **Agent**: Muscle (Claude-Opus-4.8) · **Trigger**: user mô tả case + "phân tích vụ này" (+ Note: làm theo core, simplicity-first, có code demo, report file, verify service trước done).

## Case (user mô tả)
1. Tạo shadow + stream ON → shadow khớp source.
2. Tạo master + bật realtime → source→shadow→master khớp.
3. **Issue**: master TẮT realtime + source thêm record → source→shadow vẫn khớp (shadow nhận record mới), nhưng **master MISS** (realtime off, không transmute).
4. Bật LẠI realtime → record trong "cửa sổ off" **thiếu VĨNH VIỄN** ở master nếu không nhấn đồng bộ thủ công.

## Bản chất
Realtime transmute là **forward-only** (event-driven qua `cdc.cmd.transmute-shadow` lúc ingest). Khi off→on, KHÔNG có bước **catch-up** đối soát gap shadow↔master ⇒ record xuất hiện trong cửa sổ off mất luôn. Đây là tổng quát hóa của bug binding-11 (workspace `analysis-export-jobs-drift-2026-06-11`: master frozen từ 06-05, 7 record post-06-05 thiếu).

## Kiến trúc liên quan (từ project_context + lessons)
- **TransmuteScheduler**: cron poll 60s, fencing, 3 mode: `cron` / `immediate` / `post_ingest`. Scheduler poll **CHỈ chạy `mode='cron'`** (lesson [2026-06-11]).
- **Transmute**: shadow→master qua gjson(_raw_data, jsonpath) + transform_fn + **OCC upsert** (`_source_ts older` → không overwrite data mới hơn).
- **Trigger realtime**: ingest (batch_buffer.go) bắn `cdc.cmd.transmute-shadow` → handler fan-out mọi master_binding `is_active=true AND schema_status='approved'` của shadow.
- **master_binding**: `is_active` chỉ bật khi `schema_status='approved'` (CHECK). Có sync_mode = "Realtime" (dashboard).
- Gate transmute: master active+approved, shadow active+profile_active, ≥1 approved rule.
- LWW (lesson [2026-05-25]): OCC `_source_ts < EXCLUDED._source_ts OR (= AND _source!='realtime')` → catch-up idempotent, không ghi đè data mới.

## Ràng buộc (Note user)
- Đọc lesson trước (DONE). Theo core /agent. Simplicity-first, minimal-impact, đúng pattern source, tối ưu.
- KHÔNG cheat DB / đổi config để đạt kết quả. Plan rõ ràng + code demo chi tiết.
- Report dựa trên kết quả THỰC, note file thay đổi + số dòng. Verify service trước khi done. Có `report_*.md`.
- Đề xuất 1 hướng tốt nhất (không option 1/2/3 giọng điệu).
