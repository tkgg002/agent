---
tier: T2
mode: audit
suffix: system-design
---
# 01 Requirements — system-design
- User (2026-11-09):
  - "mày vẫn chưa overview được hệ thống của tao, phải build design pattern"
  - "mày phải biết là hệ thống của tao đang chạy. mày tự nhớ điều này dùm. nó ko phải đang bị lỗi."
  - "mới chỉ là systemdessign folow thôi, còn design patten chi tiết. vì bài toán đang hướng tới multi source, multi master, lại có thêm nhiều loại Transform Type rồi. ko rõ ràng sẽ nhồi nhét 1 đống"
  - "connector nó có các action gì, đủ chưa. source nó có các action gì, đủ chưa. Master có các action gì, đủ chưa .... => toàn bộ các chức năng của tao đâu."
- Mục tiêu:
  1. Ghi nhớ vĩnh viễn vào Bộ não dự án (`agent/memory/global/project/`) rằng **hệ thống `data-hub` đang vận hành thực tế (LIVE PRODUCTION) — không phải đang bị lỗi**.
  2. Xây dựng tài liệu **System Overview & Canonical Design Pattern** bám sát 100% kiến trúc phân tầng và các nhóm chức năng (Functional Groups) đang chạy thực tế của từng service (`centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`).
  3. **Kiểm kê 100% Danh mục Hành động Nghiệp vụ (99 Domain Actions Catalog) & Đánh giá "Đủ chưa?" trên cả 7 miền chức năng:**
     - **Miền 1 — System Connector (15 actions):** `List`, `GetStatusAndConfig`, `ListPlugins`, `Create`, `UpdateConfig`, `Delete`, `Restart`, `RestartTask`, `Pause`, `Resume`, `Stop`, `ResetOffsets`, `Recover`, `BridgeOplog`, `SendDebeziumSignal` (+ đề xuất `ValidatePluginConfigDryRun`).
     - **Miền 2 — Source Connection & Source Object (20 actions):** `List/Get/CreateUpsertSourceConnections` (+ đánh giá thiếu `Test/Update/DeleteSourceConnection`), `DiscoverMongoDatabases/Collections`, `DiscoverPostgresTables` (+ đánh giá thiếu `DiscoverMySQL/SFTP`), `IntrospectObjectSchema`, `GetSyncHealth`, `List/Stats/Register/BulkRegister/UpdateSourceObjectV2`, `CreateDefaultColumns`, `ScanFields`, `ScanRawData/ScanArray`, `Standardize`, `DetectTimestampField`, `TriggerSnapshotV2/Pause/Resume/Progress`, `TriggerBatchTransform/Cancel/Status`, `ScheduledSourceCount`.
     - **Miền 3 — Shadow Binding, Mapping V2 & Ingest/DLQ (16 actions):** `List/Toggle/Delete/ProvisionShadowBinding`, `List/Create/UpdateStatus/BatchUpdate/Preview/Reload/BackfillShadowMappingRules`, `InspectShadowColumns`, `AlterShadowColumn/DropGinIndex`, `IngestCDCEvent`, `DLQEvacuateAndReplay`, `RefreshKafkaTopics`.
     - **Miền 4 — Master Connection, Binding, Mapping, DDL & Transmute (26 actions):** `List/Test/Create/Update/DeleteMasterConnection`, `List/Create/UpdateSpec/ApproveAndCreateDDL/Reject/ToggleActive/Swap/DeleteMasterBinding`, `List/SyncFromShadow/Save/BatchUpdate/UpdateTargetCol/UpdateDataType/Delete/ApproveSingleColDDL/ApproveAllColsDDL/DropColDDL` cho `mapping_rule_master`, `TriggerTransmuteFullOrInc/RealtimeShadowHook/GetStatus/Cancel` (+ đề xuất `PreviewMasterTransform`).
     - **Miền 5 — Reconciliation & Self-Healing (10 nhóm / 14 endpoints):** `ScheduledSmokeRecon`, `TriggerDeepCheck`, `TriggerPruneCheck`, `TriggerCheckAndProposeHeal`, `TriggerExecuteHeal` (SegA + SegB), `GetLatest/History/Unhealed/DeleteReport`, `GetActiveReconJobs/Status`, `TriggerBackfillSourceTs/Status`.
     - **Miền 6 — Governance (9 nhóm / 15 endpoints):** `DetectSchemaDrift`, `Pending/History/Approve/RejectSchemaChanges`, `SchemaProposals`, `SensitiveFields & Masking`, `ShadowIndexes`, `ShadowPartitions`, `ProvisioningLifecycle`, `QAAudit`.
     - **Miền 7 — Scheduler, Wizard & System Observability (8 nhóm / 14 endpoints):** `WizardSessionLifecycle`, `TransmuteSchedule`, `WorkerSchedule`, `AsyncJobStatus`, `SystemHealth`, `NATSJetStreamGovernance`, `ActivityLog`, `AlertManagement`.
  4. **Thiết kế chi tiết Code-Level Design Patterns (Strategy + Self-Registering Driver Registry + Lifecycle Hooks)** bao phủ 100% các actions đã kiểm kê trên 4 trục mở rộng đa hình cốt lõi:
     - **Trục 1 — Multi-Source & Connector Engine Pattern (`ConnectorEngineDriver`, `SourceIntrospectorDriver`, `SourceSnapshotDriver`, `BridgeStrategy`, `ReconSourceDriver`, `SourceHealFetcherDriver`)**
     - **Trục 2 — Multi-Master Destination Pattern (`MasterSinkDriver`, `MasterDDLDriver` gồm cả `AddOrAlterColumn`/`DropColumn`/`DropTarget`/`SwapTarget`, `ReconDestDriver`)**
     - **Trục 3 — Multi-Transform Strategy Lifecycle Pattern (`transmute.Strategy` + `DeletePolicyProvider` + `BatchPreEnricher` + `PostIncrementalCatchupHook`)**
     - **Trục 4 — Field-Level Extractor & Transform Registry Pattern (`FieldDirectiveFunc` + `TransformFunc`)**
  5. Thiết lập quy tắc **"1 File Thực Thi Mới + `init() Register()` = 0 `if/else` mới trong Core Engine"** và quy trình 5 bước kiểm tra trước khi sửa code.
- Ràng buộc (Technical Constraints):
  - **L001 (2026-11-09) Inventory Complete Domain Action Catalog Before Freezing Design Pattern Interfaces:** Bắt buộc kiểm kê 100% actions của từng thực thể và đối chiếu độ phủ đa engine ("Đủ chưa?") trước khi chốt các hàm trong Interface.
  - **L002 (2026-11-09) Define Strategy & Driver Registry Patterns For Polymorphic Axes Beyond High-Level System Flow:** Phải định nghĩa rõ Interface Contract, Driver Registry (`Register`/`Get`) và Lifecycle Hooks kèm Code Demo cụ thể cho từng trục đa hình.
  - **L003 (2026-11-09) Respect Live Production Baseline & Codify Existing Design Patterns:** Cấm đánh đồng lỗi môi trường test local hay thư mục phụ (`docs/*_bk*`, `scratch/`) với lỗi hệ thống đang vận hành; cấm đề xuất đập bỏ cấu trúc thư mục hiện hữu.
  - **Cập nhật trạng thái nhánh `recon-heal` (2026-11-09 14:01):** User xác nhận đã push `recon` lên, **gỡ bỏ hoàn toàn khóa hạn chế `recon-heal`** và yêu cầu hoàn thiện **Bản Thiết Kế & Lộ Trình Tổng Thể (`W1 → W6`)** lưu trực tiếp vào `agent/memory/global/project/` (kèm các file `project_context.md` trỏ trực tiếp tới tài liệu tổng thể) để mọi phiên làm việc sau vào đọc ngay.
- DoD: Lưu vĩnh viễn vào `agent/memory/global/project/data-hub/system_overview_and_design_pattern.md` và đồng bộ `project_context.md` tại cả 4 thư mục (`data-hub`, `centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`); cập nhật hồ sơ workspace đạt `verify_governance.py` PASSED.
