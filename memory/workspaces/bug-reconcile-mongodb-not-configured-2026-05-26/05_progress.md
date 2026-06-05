# 05_progress — APPEND ONLY

## 2026-05-26 15:20 — Workspace khởi tạo
- Đọc lessons L985 (silent-skip pattern, đã có Bug A fix 2026-04-20).
- Đọc lessons L3100 (conditional subscriber gating, fix đã apply cho NATS subscribers — tương tự pattern cần apply cho scheduler).
- Đọc worker_server.go L174-198 (gate cfg.MongoDB.URL), L838-879 (runReconcileCycle).
- Đọc recon_core.go (ReconCore struct, CheckAll, RunTier1, mongoClient field — dead reference).
- Đọc recon_source_agent.go (multi-source ready, `clients[sourceURL]` map).
- Đọc metadata_registry_service.go (synthesizeLegacyTableRegistry bỏ trống SourceURL).
- Đọc config.go (MongoDBConfig.URL = single legacy field, không có V2-aware).
- Đọc config-local.yml (xác nhận KHÔNG có block `mongodb:`).

## 2026-05-26 15:35 — Root cause locked
- 1 root cause + 1 architectural debt:
  - **Trực tiếp**: `cfg.MongoDB.URL == ""` → reconCore nil → scheduler tick "skipped".
  - **Architectural**: V2 metadata service không populate `entry.SourceURL` từ
    `connection_registry`, nên thậm chí khi reconCore unblock, ReconSourceAgent
    rơi vào defaultClient nil path.

## 2026-05-26 15:40 — Plan + docs viết xong
- 00_context.md, 01_requirements.md, 02_plan.md hoàn thành.
- Phase 1: populate SourceURL từ V2 connection_registry trong synthesizeLegacyTableRegistry.
- Phase 2: bỏ guard cfg.MongoDB.URL quanh reconCore init (giữ guard cho Healer/Backfill/TsDetector/FullCountAgg).
- Phase 3: hard-assert trong ReconSourceAgent.getClient.
- Phase 4: verify go build/vet/test.

## (sẽ append sau khi implement)

---

## 2026-05-26 — Closeout (Memory append)
- APPEND `agent/memory/global/lessons.md` → mục `L-2026-05-26-legacy-config-gate-kills-feature` (Global pattern: legacy single-config gate phủ lên feature-construction → V2 deployment chết âm thầm + correct flow 6 bước + 3 anti-pattern).
- APPEND `agent/memory/global/active_plans.md` → Done entry `bug-reconcile-mongodb-not-configured` (status, root cause 2 lớp, fix, files, verify, lesson ref, open follow-up).
- Task #14 (implementation/build/test) + Task #15 (lessons + active_plans append) hoàn tất.
- Pre-flight (§14): files vật lý tồn tại, Memory APPEND-only tuân thủ (§11), workspace prefix đầy đủ (§7), §12 không vi phạm (Muscle thực hiện edit code; Brain plan/document).
- Manual smoke step vẫn cần operator chạy live (limitation đã note trong report).
