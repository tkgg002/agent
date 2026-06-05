# 07_status.md — Trạng thái

| Field | Value |
|-------|-------|
| **Workspace** | `feature-cds-k8s-config-mismatch-2026-05-19` |
| **Phase** | Implementation — DELIVERED (Hướng 4 đã thực thi) |
| **Status** | ✅ COMPLETED (Muscle thực thi xong, prod-ready cho devops apply manifest) |
| **Source code changes** | 0 (Brain-only, theo CLAUDE.md §12) |

## Files

| File | Vai trò |
|------|---------|
| `00_context.md` | Bối cảnh + scope + non-goals |
| `10_gap_analysis.md` | **DELIVERABLE** — 14 mismatch, 4 bảng, root cause |
| `09_tasks_solution_proposal.md` | 3 hướng đi + khuyến nghị + 5 câu hỏi pending |
| `05_progress.md` | Audit log |
| `07_status.md` | (file này) |

## Source code changes (Muscle scope)

| File | Loại | Tóm tắt |
|------|------|---------|
| `centralized-data-service/config/config.go` | Refactor (313 → 399 dòng) | Thêm prefix `CDS_`, thay `applyEnvOverrides` bằng `envBinds` map 65 entry + multi-alias, tách `applyConnectionOverrides`. |
| `centralized-data-service/deployments/k8s/cdc-worker-deployment.yaml` | Rewrite (82 → 124 dòng) | Pin image, bỏ `DB_SINK_URL`, thêm 18 ENV `CDS_*_DB_*` + JWT + MASTER_KEY + Kafka + Debezium. |
| `centralized-data-service/README.md` | Edit section ENV | Rewrite section "Environment variables" (138 dòng) theo pattern mới. |

## Verification

- ✅ `go vet ./...` clean.
- ✅ `go build ./...` ok.
- ✅ `go test ./...` — 8/8 test package pass, 4/4 Kafka topic prefix test pass.
- ✅ Smoke test runtime với ENV mix `CDS_*` + legacy aliases + `CONNECTION_OVERRIDE_*` → all bind correctly, `validateConfig` pass.

## Next step (devops — KHÔNG thuộc scope code)

1. Bổ sung key mới vào Secret `cdc-secrets`: `system-db-{host,port,username,password,database}`, `master-db-*`, `shadow-db-*`, `jwt-secret`, `master-key`, `kafka-brokers`, `kafka-connect-url`, `kafka-schema-registry-url`, `debezium-connector-name`.
2. Replace `image: cdc-worker:v1.0.0` placeholder bằng actual tag/digest đã build và push registry.
3. Apply manifest: `kubectl apply -f deployments/k8s/cdc-worker-deployment.yaml`.
4. Verify pod: `kubectl logs deploy/cdc-worker` — không CrashLoopBackOff, log "config path: ./config/config-production.yml" + validation pass.
