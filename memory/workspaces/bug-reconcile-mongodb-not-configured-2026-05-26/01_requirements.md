# 01_requirements — Fix reconcile scheduler skipped

## DoD (Definition of Done)
- [ ] `go build ./...` PASS.
- [ ] `go vet ./internal/...` PASS.
- [ ] `go test ./internal/service/ -count=1 -run Recon` PASS (no regression).
- [ ] `runReconcileCycle` KHÔNG còn ghi "skipped (MongoDB not configured)" khi local dev start worker với V2 connection_registry đã có mongo source — scheduler dispatch CheckAll thật.
- [ ] V2 entries có `entry.SourceURL` được populate đúng từ `connection_registry` → ReconSourceAgent lazy-resolve mongo client per-source.
- [ ] ReconCore init bỏ phụ thuộc vào legacy `cfg.MongoDB.URL` (chỉ defaultClient=nil khi không có legacy URL).
- [ ] ReconHealer / Backfill / TimestampDetector / FullCountAgg vẫn giữ guard hiện tại (defer ngoài scope) — nhưng phải có WARN log rõ "feature X disabled vì cfg.MongoDB.URL empty AND no V2 mongo connection_registry rows" để operator biết.
- [ ] Hard-assert trong `ReconSourceAgent.getClient`: nếu `sourceURL=="" && defaultClient==nil` → return error rõ ràng (không panic).
- [ ] `report_reconcile_mongodb_not_configured_2026-05-26.md` ghi đủ file thay đổi + verify steps.
- [ ] Workspace docs đầy đủ: 00–09 prefix.
- [ ] `lessons.md` append lesson global pattern.
- [ ] `active_plans.md` append Done entry.

## Non-goals (defer)
- KHÔNG refactor ReconHealer / Backfill / TimestampDetector / FullCountAgg để lazy-resolve. Scope quá lớn cho bug fix này — chỉ làm scheduler reconcile work.
- KHÔNG đổi YAML config / DB.
- KHÔNG xóa field `rc.mongoClient` trong ReconCore (giữ backward compat cho callers + tránh diff lớn).

## Constraint
- Code change minimal — chỉ touch những gì cần (Simplicity First §6 GEMINI.md).
- Không sửa migration / schema DB.
- Backward-compat: khi cfg.MongoDB.URL được set + V2 chưa active, behavior cũ phải giữ nguyên.
