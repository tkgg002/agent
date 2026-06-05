# 02_plan — Normalize Global Lessons

## Chiến lược thực thi (parallel chunk + script assembly)
Vì file 530KB không thể nạp 1 context, dùng **9 sub-agent** đọc 9 chunk song song, mỗi agent:
- Đọc range dòng được giao (ranh giới cắt tại `---`, không vỡ lesson).
- Viết lại TỪNG lesson sang block canonical (Rule 13) vào part-file riêng `/tmp/norm_part_NN.md`.
- Trả về summary nhỏ (count + tally theo category + anomalies). KHÔNG trả nội dung (tránh nổ context).

Sau đó: script assembly gom part-files → group theo taxonomy → sort theo ngày → ghi file đích.

## Chunk map (9 chunks, cắt tại separator `---`)
| # | Range dòng |
|---|---|
| 1 | 1 – 558 |
| 2 | 559 – 1126 |
| 3 | 1127 – 1678 |
| 4 | 1679 – 2238 |
| 5 | 2239 – 2808 |
| 6 | 2809 – 3393 |
| 7 | 3394 – 3941 |
| 8 | 3942 – 4394 |
| 9 | 4395 – 5061 |

## Taxonomy 8 nhóm Global (mỗi lesson thuộc đúng 1 nhóm)
1. `01-process-governance` — Brain/Muscle discipline, plan-before-code, gatekeeper approval, verification-before-done, tuân thủ Rule, autonomy/recidivism, session handoff, skill-listing.
2. `02-architecture-design` — coupling/decoupling, CQRS/CommandBus, DRY, over-engineering, single-source-of-truth, layering, observability/telemetry design.
3. `03-schema-migration` — schema drift, DDL gen/ordering, migrations, search_path, GORM/pgx model↔DB, add/rename column.
4. `04-cdc-data-pipeline` — Kafka, Debezium, connector, snapshot, connection-registry, masking, DLQ, reconcile, shadow tables (domain CDC).
5. `05-config-environment` — env vars, DSN/secret resolution, fallback merge, config-local, docker/compose, k8s config, .env.
6. `06-serialization-type` — BSON/Extended-JSON, cast expr, type conversion, serialization-form drift, dual-stack field-routing/silent-drop, identifier migration.
7. `07-testing-verification` — exercise-driven verification, PASS criteria, test uplift, no-sqlmock convention, build≠test, integration testing.
8. `08-memory-knowledge` — workspace mgmt, audit-log immutability, knowledge retention, documentation discipline, lesson-writing standard.

## Block format mỗi sub-agent xuất ra part-file
```
@@LESSON@@
@@CAT=04-cdc-data-pipeline@@
@@DATE=2026-06-02@@
### [2026-06-02] <tiêu đề canonical ngắn>
- **Global Pattern**: `[A] <làm B> lên [X]` → `[Y]`. **Đúng**: <correct flow>.
- **Bối cảnh (Trigger)**: ...
- **Root Cause**: ...
- **Fix/Correct Flow**: ...
- **Phạm vi (≥3 dự án?)**: ...
- **Tags**: #t1 #t2 #t3
- **Nguồn**: lessons.md [2026-06-02] (~dòng X–Y)
@@END@@
```

## Các bước
1. [done] Thống kê sơ bộ + chunk map + workspace.
2. Dispatch 9 sub-agent song song → part-files.
3. Verify part-files (đủ 9, tổng count khớp).
4. Script assembly → group/sort → body.
5. Ghi `lessons_global_normalized.md` (Dashboard + Taxonomy + Body + Index).
6. APPEND index vào `lessons.md` (verify nội dung cũ nguyên vẹn).
7. Cập nhật 03_implementation, 07_status, 09_solution, 05_progress.
