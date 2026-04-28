# Phase 30 Plan — V2 Direct Re-detect

1. Audit backend CMS handler và worker consumer cho `detect-timestamp-field`.
2. Mở direct CMS route theo `source_object_id`.
3. Mở direct `dispatch-status` route theo `source_object_id`.
4. Enrich `reconciliation report` với `source_object_id` để FE operator-flow gọi route mới.
5. Refactor `ReDetectButton` ưu tiên V2 direct, fallback bridge khi cần.
6. Verify bằng `go test ./...` và `npm run build`.
7. Ghi đủ workspace docs + append progress/status/gap.
