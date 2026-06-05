# 02_plan_lww_guard — Roadmap

> **Phase**: `lww_guard`
> **Strategy**: Phương án **D** (= C + tie-breaker `_source` discriminator)

## Milestones

| ID | Milestone | DoD | ETA |
|---|---|---|---|
| **M1** | Source code change: `cdcCols` + DDL inline + OCC guard extension | `git diff` clean, `go build` PASS | T+1h |
| **M2** | Migration files: `060_v1_add_source_ts_to_shadow.sql` + `061_v1_snapshot_progress_cluster_time.sql` | Apply trên DB metadata 5433 + DB shadow 5436 success, không lock > 5s | T+30m |
| **M3** | Snapshot envelope dùng Mongo `clusterTime` thay `time.Now()` | Unit test `TestBuildSnapshotEnvelope_ClusterTime` PASS, log mới `snapshot.v2 cluster time captured` | T+1h |
| **M4** | Snapshot path force `_source='snapshot:v2'` ở downstream record map | Trace evidence trong shadow: `SELECT DISTINCT _source FROM <shadow>` có cả `'snapshot:v2'` lẫn `'debezium-v125'` | T+30m |
| **M5** | Unit + integration tests | 5 test case PASS, coverage `BuildUpsertSQLInSchema` ≥ 80% | T+1h |
| **M6** | Race smoke test thực tế trên `source_object_id=18` | Snapshot không clobber realtime, evidence SQL log | T+1h |
| **M7** | Security review `/security-agent` | Không có finding nghiêm trọng | T+30m |
| **M8** | Report file + lesson append + workspace doc update | `report_lww_guard_2026-05-21.md` đầy đủ, `lessons.md` APPEND, `05_progress.md` APPEND | T+30m |

**Tổng ETA**: ~6h (1 dev session). Buffer 30% → 8h max.

## Phase ordering (sequential — không parallel)

```
M1 (source) ─→ M2 (migration) ─→ M3 (clusterTime) ─→ M4 (_source set) ─→ M5 (test) ─→ M6 (smoke) ─→ M7 (security) ─→ M8 (report)
```

Lý do sequential: M2 phụ thuộc M1 (schema phải có column); M3+M4 không break nếu M2 chưa apply (guard skip khi `hasSourceTs=false`); M5 cần M1-M4 xong để test integration; M6 cần worker restart sau khi build binary mới.

## Critical paths & rollback

### Critical path
- **M1 → M2**: nếu migration ADD COLUMN fail trên DB lớn, fallback chạy `ALTER TABLE ... ADD COLUMN ... NULL` (không default → metadata-only instant) + backfill async qua job.
- **M3 (clusterTime)**: nếu `db.hello()` không trả `$clusterTime` (Mongo standalone không phải replica set?) → fallback `replSetGetStatus` HOẶC dùng `oplog.rs` last entry ts. **Verify trong M3 chứ không assume**.
- **M6 (race smoke)**: cần coordination với 1 source insert/update thực để inject realtime event.

### Rollback strategy
- M1: `git revert` 1 commit hash.
- M2: migration KHÔNG có DROP COLUMN — rollback là **forward-only** (run migration 062 hoặc tương đương `ALTER TABLE ... DROP COLUMN IF EXISTS "_source_ts"`). KHÔNG khuyến nghị rollback vì data tồn tại.
- M3, M4: revert commit, worker rebuild.

## Resource & gate ownership

| Gate | Owner | Tool |
|---|---|---|
| Code change | Muscle (CC CLI) | Edit tool |
| Migration apply | Muscle | `psql` qua bash, hoặc CMS migration runner |
| Test execution | Muscle | `go test`, `go vet` |
| Smoke test | Muscle + User coordination | Worker restart (k8s rollout OR local `go run`) + trigger snapshot FE |
| Security review | Muscle | `/security-agent` skill |
| Report + lesson | Muscle | Write tool (.md only) |

## Decision tree khi gặp bất ngờ

```
Step M3 — đọc clusterTime
├── Mongo driver trả $clusterTime OK?
│   ├── Yes → bind vào envelope → continue M4
│   └── No (standalone mode hoặc permission)
│       ├── replSetGetStatus có optime?
│       │   ├── Yes → dùng optime → continue
│       │   └── No → fallback wall-clock + log WARN + bỏ M4 force, revisit phase sau
│       └── (alternative) đọc local oplog.rs cuối → continue

Step M6 — race smoke
├── Snapshot quét xong + realtime đến trước/sau snapshot?
│   ├── Realtime đến trước → kiểm tra _source_ts shadow == realtime ts → PASS
│   ├── Realtime đến sau → kiểm tra shadow update với realtime ts > snapshot ts → PASS
│   └── ts bằng nhau (rare) → kiểm tra _source = 'debezium-v125' (realtime win) → PASS
```

## Communication plan

- **Trước M1**: APPEND `05_progress.md` Followup #7 với "phase plan approved, Muscle started".
- **Sau mỗi milestone**: APPEND `05_progress.md` 1 dòng `[Timestamp] [Muscle:claude-sonnet-4-6] Mx done — evidence: <file:line / SQL count>`.
- **Sau M8**: APPEND `lessons.md` Global Pattern + update `active_plans.md` workspace row.
