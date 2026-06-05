# 06_validation.md — Test plan (REVISED)

## Smoke test (Muscle chạy sau apply patch §1-3)

1. **Build**: `cd centralized-data-service && go build ./...` → exit 0.
2. **Unit**: `go test ./internal/handler/... ./internal/service/...` → green.
3. **Restart**: `kubectl rollout restart deploy/centralized-data-service`.
4. **Approve rule** (KHÔNG cần restart CMS):
   - Vào CMS UI hoặc API approve 1 mapping rule cho `source_object_id=66`.
5. **Trigger snapshot**: `POST /api/v1/source-objects/66/snapshot-v2` → 202.
6. **Verify log**:
   ```bash
   kubectl logs deploy/centralized-data-service --since=1m | grep snapshot.mapping_rules.loaded
   ```
   Expect: `count=N` với N = số rule approved trong DB hiện tại (bao gồm rule vừa approve).
7. **Verify shadow**:
   ```sql
   SELECT column_name FROM information_schema.columns
   WHERE table_schema='shadow' AND table_name='<shadow_table_66>';
   ```
   Expect: column rule mới approved có mặt.

## Regression check (realtime CDC không bị ảnh hưởng)
- Stream 1 event qua Debezium → `MapData(bindingID, ...)` vẫn xài cache cũ → realtime work bình thường.
- Tail: `grep "MapData" worker.log` → vẫn có log như trước.

## Pass/Fail
| Step | Pass | Fail action |
|---|---|---|
| 1-2 | Build + test green | Brain re-plan |
| 6 | Log show count=N (N ≥ rule mới + cũ) | Check `mappingV2Repo.ListActiveBySourceObject` query |
| 7 | Column rule mới có | Verify rule status='approved' AND is_active=true trong DB |
| Regression | Realtime CDC ok | Stream test event, verify shadow update |
