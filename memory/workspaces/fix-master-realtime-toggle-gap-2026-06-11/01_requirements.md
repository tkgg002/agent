# 01_requirements

## R1 — Bật lại realtime PHẢI tự catch-up gap (no permanent miss)
- R1.1: Khi master realtime chuyển **off→on**, hệ thống PHẢI tự động materialise mọi record shadow tích luỹ trong cửa sổ off vào master — KHÔNG để operator phải nhấn "đồng bộ thủ công".
- R1.2: Catch-up phải **idempotent** (OCC/LWW): chỉ insert record thiếu, KHÔNG ghi đè data master mới hơn, KHÔNG tạo trùng (lesson [2026-05-25]).
- R1.3: Chỉ chạy catch-up khi transition THẬT false→true (tối ưu — không full-scan thừa mỗi lần toggle).
- R1.4: Tắt realtime (on→off) KHÔNG kích hoạt catch-up.
- R1.5: Tôn trọng gate transmute hiện có (master active+approved, shadow active, ≥1 approved rule) — worker tự gate; không bypass.

## R2 — Minimal-impact, reuse pattern source
- R2.1: KHÔNG cơ chế mới — reuse đúng path `cdc.cmd.transmute` (full Shadow→Master, OCC upsert) mà RunNow/"manual sync" đang dùng.
- R2.2: Thay đổi gói gọn trong handler toggle (`TransmuteScheduleHandler.Toggle`), không đụng worker/transmuter.
- R2.3: Negative-path (G4): dispatch fail → KHÔNG để schedule kẹt `last_status='running'` (revert).

## R3 — Verify thật (G2/G3/G6)
- R3.1: Reproduce red→green: tắt realtime → thêm record shadow → master miss → bật lại realtime → master tự khớp (count nguồn=đích).
- R3.2: Build OK + service chạy + endpoint trả `catchup_dispatched=true` đúng transition.

## Non-goals
- Không xây periodic sweep mới (đã có recon Segment B shadow↔master + heal-B làm safety-net — khuyến nghị bật định kỳ, không implement ở task này).
- Không đổi schema/config/DB.
