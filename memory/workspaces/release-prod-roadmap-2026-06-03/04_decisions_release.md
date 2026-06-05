# 04_decisions_release.md — Quyết định chốt từ User

> **Workspace**: `release-prod-roadmap-2026-06-03` | **Ngày**: 2026-06-03

## D-1 — Deadline
- **Quyết định**: KHÔNG có deadline cứng → giữ timeline đề xuất (~go-live 2026-06-25 ±2d).
- **Hệ quả**: Ưu tiên đủ DoD 6 tiêu chí mỗi luồng (R2), không cắt scope vì áp lực thời gian.

## D-2 — Staging
- **Quyết định**: Staging gần-prod ĐÃ CÓ.
- **Hệ quả**: Gỡ rủi ro HIGH "chưa có staging". Task E3 đổi từ "dựng staging (~3-5d)" → **"validate staging hiện có + seed data (~1d)"**. Phase E rút ~1-2 ngày → go-live có thể sớm hơn (~06-23/24), vẫn giữ buffer 06-25.

## D-3 — Release mode
- **Quyết định**: **Big-bang** — chờ đủ 5 luồng + E2E full rồi go-live 1 lần.
- **Hệ quả**: KHÔNG release incremental. Critical path giữ nguyên A→B→C→D→E→F; Go/No-Go (F3) gate đủ 5 luồng cùng lúc. Rủi ro: phát hiện gap muộn ở luồng cuối làm trượt cả release → buffer ±2d + escalation re-plan nếu gap > 1 luồng (§8).

## Câu hỏi còn mở (không chặn timeline tổng, cần trước Phase E5)
- **Q3 cũ (SLA/throughput)**: chưa có số → E5 Load/perf smoke cần Boss cung cấp rows/s + latency target shadow→master trước khi chạy. Tạm dùng baseline đo từ Local nếu chưa có.
