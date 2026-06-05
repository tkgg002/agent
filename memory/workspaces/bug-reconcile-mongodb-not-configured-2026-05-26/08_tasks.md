# 08_tasks — Reconcile MongoDB-not-configured fix

## T1 — Populate SourceURL trong V2 metadata reload
- [ ] File: `internal/service/metadata_registry_service.go`
- [ ] Build `connectionURIByCode` map sau khi build `connectionCodeByID` (ReloadAll).
- [ ] Pass URI vào `synthesizeLegacyTableRegistry`.
- [ ] Update signature `synthesizeLegacyTableRegistry(src, binding, sourceURI string)`.
- [ ] Log INFO summary số source URL resolved trong ReloadAll.

## T2 — Tách guard MongoDB.URL khỏi ReconCore init
- [ ] File: `internal/server/worker_server.go`
- [ ] `mongoClientShared` vẫn gate bởi cfg.MongoDB.URL (legacy default).
- [ ] `reconCore` init **luôn** (defaultClient = mongoClientShared có thể nil).
- [ ] Log INFO "ReconCore initialized in V2-only mode (defaultClient unavailable)" khi mongoClientShared=nil.
- [ ] Giữ guard cho ReconHealer + Backfill + TimestampDetector + FullCountAgg.

## T3 — Hard-assert trong ReconSourceAgent.getClient
- [ ] File: `internal/service/recon_source_agent.go`
- [ ] Nếu sourceURL=="" && defaultClient==nil → return error rõ ràng.

## T4 — Update runReconcileCycle log message
- [ ] File: `internal/server/worker_server.go::runReconcileCycle`
- [ ] Update message: khi reconCore=nil giờ là defensive (đáng lẽ không xảy ra sau T2). Log Warn để alert nếu xảy ra.

## T5 — Verify
- [ ] `go build ./...` PASS.
- [ ] `go vet ./internal/...` PASS.
- [ ] `go test ./internal/service/ -count=1 -run Recon` PASS.

## T6 — Documentation
- [ ] 03_implementation.md.
- [ ] 09_tasks_solution.md.
- [ ] report_reconcile_mongodb_not_configured_2026-05-26.md.
- [ ] Append lesson global.
- [ ] Append active_plans Done entry.
