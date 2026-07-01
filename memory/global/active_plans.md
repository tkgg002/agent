# Active Plans Registry

> **Maintained by**: Brain (Antigravity)
> **Last Updated**: 2026-06-23
> **Purpose**: Registry để Brain biết workspace nào đang active → load đúng context khi bắt đầu phiên mới. KHÔNG phải cơ chế agent communication.

| Workspace | Project | Status | Last Active |
|-----------|---------|--------|-------------|
| bug-delete-connector-cleanup-bindings-2026-06-30 | data-hub | 🟡 Active (Planning) | 2026-06-30 |
| bug-recon-heal-missing-shadow-2026-06-30 | data-hub | ✅ Done | 2026-06-30 |
| bug-transmuter-bigint-casting-2026-06-30 | centralized-data-service | ✅ Done | 2026-06-30 |
| doc-investigate-master-sync-new-field-2026-06-30 | data-hub | ✅ Done | 2026-06-30 |
| ReconHealStaleReport | Sửa lỗi healSegmentA lặp lại do lấy stale report | ✅ Done | 2026-06-30 |
| bug-recon-false-drift-payment-bills-2026-06-30 | Sửa lỗi đối soát báo khống 1.410 drift ảo trên bảng payment_bills | ✅ Done | 2026-06-30 |
| bug-reconciliation-heal-noop-2026-06-30 | Sửa lỗi Reconciliation Heal trả về noop do cấu hình sai timestamp field | ✅ Done | 2026-06-30 |
| feat-recon-heal-optimization-2026-06-30 | Tối ưu hóa cơ chế Recon & Heal (Segment A & B) và đồng bộ Soft-delete Master | ✅ Done | 2026-06-30 |
| bug-data-integrity-drift-mismatch-2026-06-29 | Sửa lỗi lệch trạng thái đối soát dữ liệu (Khớp khi lệch) | ✅ Done | 2026-06-29 |
| bug-data-integrity-missing-tables-2026-06-29 | Tìm và sửa lỗi thiếu table shadow/master trong Data Integrity | ✅ Done | 2026-06-29 |
| bug-frontend-safe-modification-2026-06-29 | Sửa đổi an toàn Frontend (chống crash & date format) | ✅ Done | 2026-06-29 |
| upgrade-core-system | Upgrade Core Brain/Muscle System | ✅ Done | 2026-02-25 |
| feature-refactor-2026 | GooPay Core Refactor 2026 | ✅ Done — sẵn sàng tiếp tục | 2026-02-25 |
| optimize-brain-muscle-models | Tối ưu hóa model cho Brain/Muscle | ✅ Done (V2 Quota & Multi-Muscle) | 2026-02-25 |
| compare-disbursement-export | So sánh logic DisbursementTicketExport | ⏸ Paused | 2026-02-25 |
| compare-disbursement-trans-his-export | So sánh logic DisbursementTransHisExport | ✅ Done | 2026-02-27 |
| feature-merchant-export-activation-info | Bổ sung thông tin kích hoạt Merchant Export | ✅ Done | 2026-03-12 |
| feature-id-expired-notification-log-export | Tạo IDExpiredNotificationLogExport type | ✅ Done | 2026-02-27 |
| feature-fee-configuration | Cấu hình phí dịch vụ (Fee Configuration) | ⏸ Paused | 2026-03-03 |
| feature-cdc-integration | CDC Integration (Debezium-only sau commit 8ef7d71 remove airbyte) — Phase F (F1+F3) Done 2026-05-04 | ✅ Done | 2026-05-04 |
| feature-system-refactor-2026-05 | System Refactor 2026-05 — bucket B1+B2 (hygiene + tooling), 4 service local smoke | 🟡 Active | 2026-05-04 |
| feature-export-driver-search | Driver Info & Approximate Search in Exports | ✅ Done | 2026-03-24 |
| upgrade-agent-infrastructure | Nâng cấp hạ tầng Agent v1.10.0 (Brain/Muscle) | ✅ Done | 2026-04-06 |
| feature-trans-his-collection-export | Export TransHis Collection | 🟡 Active | 2026-04-09 |
| feature-multi-pg-isolation-e2e | Tách 4 PG containers (auth/cdc/dest/source) + E2E auto-pipeline | 🟡 Active | 2026-04-28 |
| feature-cdc-system-refactor | Task #19 service-tier drainage — Đợt G/H/I closed by max → Đợt J handed off to x2 (cms-lane locked) | 🟡 Active | 2026-05-07 |
| centralized-data-service-config-audit | Audit DB Connections, GORM configs & Fix Tests | ✅ Done | 2026-05-18 |
| feature-cdc-activity-log-metrics | Sửa đổi RowsAffected & DB Materialization Metrics | 🟡 Active | 2026-05-21 |
| bug-schema-drift-loop-2026-05-25 | Schema drift loop + SLOW SQL in event handler | ✅ Done | 2026-05-26 |
| bug-partition-dropper-relation-missing-2026-05-26 | Postgres relation missing in partition dropper | ✅ Done | 2026-05-26 |
| bug-snapshot-v2-host-uri-2026-05-21 | LWW Guard cho Dual-Stream Consistency | ✅ Done | 2026-05-25 |
| bug-kafka-consumer-empty-topics-2026-05-26 | Kafka Consumer empty topics & CMS schema qualifier | ✅ Done | 2026-05-26 |
| bug-cms-slow-sql-probes-2026-05-26 | Slow SQL in probes & system health checks | ✅ Done | 2026-05-26 |
| feature-sensitive-masking-opt-out-2026-06-02 | Sensitive Masking Opt-Out & UI Controls | ✅ Done | 2026-06-02 |
| bug-snapshot-cache-latency-binding-66 | Giải quyết stale cache masking khi snapshot v2 | ✅ Done | 2026-06-02 |
| feature-masters-page-audit-2026-06-02 | Masters Page Audit & Enhancements | ✅ Done | 2026-06-03 |
| feature-sync-shadow-master-bindings-2026-06-04 | Syncing Shadow And Master Bindings | ✅ Done | 2026-06-04 |
| feature-table-registry-ui-enhancement-2026-06-09 | Table Registry UI Enhancement | ✅ Done | 2026-06-09 |
| feature-system-health-recon-removal-2026-06-11 | System Health Reconciliation Removal | ✅ Done | 2026-06-11 |
| feature-recon-pipeline-grid-ui-2026-06-11 | Recon Pipeline Grid UI Enhancement | ✅ Done | 2026-06-12 |
| feature-data-integrity-overview-ui-2026-06-12 | Data Integrity Overview UI Tách Cột | ✅ Done | 2026-06-12 |
| bug-recon-pipeline-duplicate-rows-2026-06-12 | Sửa lỗi trùng lặp dòng trong ReconPipelineGrid | ✅ Done | 2026-06-12 |
| feat-recon-no-select-count-star-2026-06-16 | Tối ưu hóa hiệu năng Recon (loại bỏ full-collection scan) | ✅ Done | 2026-06-16 |
| feat-refactor-master-mapping-rule-2026-06-16 | Tái cấu trúc master mapping rule handler theo Hexagonal/CQRS | ✅ Done | 2026-06-16 |
| feat-screaming-architecture-refactor-2026-06-16 | Phân chia code commands/queries theo nhóm chức năng | ✅ Done | 2026-06-16 |
| feat-api-handlers-hexagonal-refactor-2026-06-16 | Tái cấu trúc toàn bộ API Handlers sang chuẩn Hexagonal Architecture | ✅ Done | 2026-06-17 |
| bug-master-connection-not-found-2026-06-17 | Điều tra lỗi master_connection_not_found | ⏸ Paused | 2026-06-17 |
| feat-infra-drainage-refactor-2026-06-17 | Di chuyển h.db và nats client về internal/infra và audit SQL/NATS | ✅ Done | 2026-06-19 |

---

## Notes
- **Active** 🟡: Đang làm trong phiên hiện tại hoặc phiên gần nhất
- **Paused** ⏸: Tạm dừng, sẽ tiếp tục sau
- **Done** ✅: Hoàn thành, archived
- Khi bắt đầu phiên mới: Brain đọc bảng này → load workspace có status Active đầu tiên

---
## Appended 2026-06-18
| cds-project-inventory-2026-06-18 | centralized-data-service | ✅ Done | 2026-06-18 |
| feat-worker-subhandlers-refactor-2026-06-19 | centralized-data-service | ✅ Done | 2026-06-19 |
| feat-infra-drainage-refactor-2026-06-17 | centralized-data-service | ✅ Done | 2026-06-19 |
| bug-scan-raw-data-reversion-2026-06-19 | centralized-data-service | ✅ Done | 2026-06-19 |
| feat-admin-helpers-refactor-2026-06-19 | centralized-data-service | ✅ Done | 2026-06-19 |
| feat-admin-server-refactor-2026-06-19 | centralized-data-service | ✅ Done | 2026-06-19 |
| feat-source-register-migration-2026-06-19 | centralized-data-service | ✅ Done | 2026-06-19 |
| feat-base-handler-emit-step-completed-2026-06-19 | centralized-data-service | ✅ Done | 2026-06-19 |
| feat-batch-transform-shadow-refactor-2026-06-19 | centralized-data-service | ✅ Done | 2026-06-19 |
| task-handler-report-2026-06-19 | centralized-data-service | ✅ Done | 2026-06-19 |
| bug-scan-array-child-fields-2026-06-19 | centralized-data-service | ✅ Done | 2026-06-19 |
| bug-scan-handler-logic-audit-2026-06-19 | centralized-data-service | ✅ Done | 2026-06-19 |
| audit-refactoring-gaps-2026-06-20 | centralized-data-service | ✅ Done | 2026-06-20 |
| refactor-clean-large-files-cds-2026-06-20 | centralized-data-service | ✅ Done | 2026-06-21 |
| refactor-large-files-campaign-2026-06-21 | centralized-data-service | ✅ Done | 2026-06-22 |
| doc-architecture-flow-2026-06-22 | centralized-data-service | ✅ Done | 2026-06-22 |
| feat-decouple-handlers-db-2026-06-22 | centralized-data-service | ✅ Done | 2026-06-22 |
| feat-hermes-learning-loop-2026-06-22 | /agent | ✅ Done | 2026-06-22 |
| TracesHardening | data-hub | ✅ Done | 2026-06-23 |
| bug-batch-transform-v1-abandon-2026-06-23 | data-hub | ✅ Done | 2026-06-23 |
| bug-cdc-pipeline-issues-2026-06-23 | data-hub | ⏸ Paused | 2026-06-23 |
| feat-reconcile-pipeline-validation-2026-06-23 | data-hub | ✅ Done | 2026-06-23 |
| bug-metadata-cascade-masking-scan-fix-2026-06-23 | data-hub | ✅ Done | 2026-06-23 |
| bug-exclude-noisy-traces-2026-06-23 | data-hub | ✅ Done | 2026-06-23 |
| bug-sync-mapping-rules-500-2026-06-23 | data-hub | ✅ Done | 2026-06-23 |
| bug-delete-master-mapping-rules-2026-06-23 | data-hub | ✅ Done | 2026-06-23 |
| feat-scheduler-tracing-2026-06-23 | data-hub | ✅ Done | 2026-06-23 |
| bug-missing-debezium-pg-plugin-2026-06-23 | data-hub | ✅ Done | 2026-06-23 |
| feature-postgresql-schema-support | data-hub | ✅ Done | 2026-06-23 |
| bug-pg-scan-mapping-flows-2026-06-23 | data-hub | ✅ Done | 2026-06-23 |
| bug-snapshot-v2-postgresql-support-2026-06-23 | data-hub | ✅ Done | 2026-06-23 |
| bug-pg-scan-fields-connection-failed-2026-06-23 | data-hub | ✅ Done | 2026-06-23 |
| bug-worker-crash-on-start-2026-06-24 | data-hub | ✅ Done | 2026-06-24 |
| bug-snapshot-v2-postgresql-zero-records-2026-06-24 | data-hub | ✅ Done | 2026-06-24 |
| query-db-connection-registry | data-hub | ✅ Done (Direct Scratch Run) | 2026-06-24 |
| bug-mapping-rules-and-security-2026-06-24 | data-hub | ✅ Done | 2026-06-24 |
| bug-scan-array-flatten-dynamic-fields-2026-06-24 | data-hub | ✅ Done | 2026-06-24 |
| feat-recon-hardening-2026-06-24 | centralized-data-service | 🟡 Active (Planning) | 2026-06-24 |
| bug-debezium-delete-not-working-2026-06-25 | centralized-data-service | ✅ Done | 2026-06-25 |
| bug-duplicate-master-table-different-schema-2026-06-25 | centralized-data-service | ✅ Done | 2026-06-25 |
| bug-export-jobs-recon-drift-2026-06-25 | centralized-data-service | ⏸ Paused | 2026-06-25 |
| bug-snapshot-limit-5000-2026-06-26 | centralized-data-service | ✅ Done | 2026-06-26 |
| bug-recon-smoke-structural-drift-2026-06-26 | centralized-data-service | ✅ Done | 2026-06-26 |
| bug-recon-smoke-safety-2026-06-26 | centralized-data-service | ✅ Done | 2026-06-26 |
| feat-recon-smoke-integration-phases-5-6-7-2026-06-26 | data-hub | ✅ Done | 2026-06-29 |
| feat-recon-runtime-check-2026-06-29 | data-hub | ✅ Done | 2026-06-29 |
| audit-sinkworker-update-2026-06-29 | centralized-data-service | ✅ Done | 2026-06-29 |
| bug-mask-mongodb-credentials-log-2026-06-30 | centralized-data-service | ✅ Done | 2026-06-30 |
| feat-metrics-active-count-shadow-master-2026-06-30 | centralized-data-service | ✅ Done | 2026-06-30 |
| bug-export-jobs-reconciliation-drift-2026-06-30 | centralized-data-service | 🟡 Active | 2026-06-30 |
| tier2-xor-hash-check | centralized-data-service | 🟡 Active | 2026-07-01 |



