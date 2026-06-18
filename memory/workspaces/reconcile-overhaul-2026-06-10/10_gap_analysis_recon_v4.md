# 10_gap_analysis_recon_v4.md — Hiện trạng Reconcile vs Chuẩn End-to-End

> Workspace `reconcile-overhaul-2026-06-10` | 2026-06-10 | Đối chiếu 4 trụ chuẩn (Boss blueprint)
> Mọi verdict dựa trên code + DB thật (evidence từ khảo sát 2 subagent + psql 5433/5434/5436).

## Kết luận 1 dòng
Engine recon hiện tại là **một nửa hệ thống đúng**: 3-tier + watermark + lock + report store dùng được (GIỮ), nhưng **chỉ đối soát source↔shadow** — tầng transmute (shadow↔master) KHÔNG ai đối soát, heal đi tắt bypass pipeline (và đang chết), không alert, không lag monitoring → **chưa phải End-to-End** theo chuẩn.

## Gap matrix vs 4 trụ chuẩn

| # | Trụ chuẩn (Boss) | Hiện trạng (evidence) | Verdict |
|---|------------------|----------------------|---------|
| 1 | **Watermark / Moving target** — chỉ đối soát tới mốc an toàn quá khứ | Có: `upper = min(srcMax, dstMax, now−5m)`, lookback 7d, window 15' (`recon_core.go:400,370`) | 🟡 CÓ nhưng freeze margin **tĩnh 5'** — không adaptive theo lag thực của ingest+transmute; lag tăng → false positive |
| 2a | **Mức 1 Count/Aggregation** (5-10') | Có: Tier1 count per 15-min window, stagger+jitter, threshold (`recon_core.go:431`) | 🟢 ĐẠT (vừa hồi sinh được trên shadow-plane) — nhưng chỉ segment source↔shadow |
| 2b | **Mức 2 Hash/Checksum chunk** (vài giờ) | Có: Tier2 XOR xxhash(id+ts) per window + drill-down diff IDs (`recon_core.go:545`) | 🟡 CÓ — chỉ source↔shadow; lỗi window bị `continue` nuốt (false negative, `recon_core.go:592-601`) |
| 2c | **Mức 3 Row-by-row diff** (đêm) | Tier3 = 256-bucket hash whole-table off-peak (`recon_core.go:658`) — fingerprint, KHÔNG field-level diff | 🟡 GẦN — thiếu row/field-diff thật cho block lỗi |
| 2-E2E | **Đối soát End-to-End source↔master** | **KHÔNG CÓ.** `TargetTable` = shadow table (`metadata_registry_service.go:739`); không tier nào chạm master (5434). Transmute sai/rớt → master sai mà recon vẫn xanh | 🔴 **THIẾU CẢ TẦNG — gap lớn nhất** |
| 3a | **Audit log + Alert ngưỡng** | Report ghi `cdc_reconciliation_report` + activity_log; **không có** threshold/alert rule/Slack; "0 tables checked" từng báo success | 🔴 THIẾU alert; ✅ store có sẵn |
| 3b | **Self-healing = re-trigger event qua pipeline** | Heal hiện tại = **bypass**: đọc Mongo → OCC upsert thẳng PG shadow (`recon_heal.go`), đi tắt qua masking/mapping/pipeline; phụ thuộc default Mongo client → **đang disabled trên V2** (`worker_server.go` log `features_disabled=recon_healer`); `extractSourceTsFromDoc` hardcode `updated_at` → OCC bypass cho camelCase (`recon_heal.go:876`) | 🔴 SAI TRIẾT LÝ + ĐANG CHẾT → **bỏ, làm lại theo re-trigger** |
| 4 | **Lag monitoring trung gian** (Debezium/Kafka/shadow/worker backlog) | Prometheus có recon metrics (drift/duration/heal) nhưng **không có lag per-segment**; SystemHealth có Kafka consumer info nhưng không nối vào recon report | 🔴 THIẾU |
| — | Hạ tầng nền (lock, fencing, leader, run-state, DLQ retry) | 4 lớp lock + `recon_runs` unique-running + `failed_sync_logs` retry + circuit breaker + rate limiter | 🟢 TỐT — GIỮ NGUYÊN |

## Phân loại GIỮ / BỎ-LÀM-LẠI / XÂY MỚI
- **GIỮ**: engine 3-tier (Segment A), watermark khung, toàn bộ lock/leader/run-state, report store, DLQ retry, metrics khung.
- **BỎ → LÀM LẠI**: cơ chế heal bypass (thay bằng re-trigger qua pipeline chuẩn); semantics "success khi 0 checked" (đã vá hôm nay — giữ bản vá làm nền).
- **XÂY MỚI**: Segment B (shadow↔master), E2E roll-up + drift localization, watermark adaptive, alert threshold, lag monitoring 3 điểm, row-diff thật cho L3.

→ Thiết kế chi tiết + code demo + roadmap: `09_tasks_solution_recon_v4.md`.
