---
tier: T2
workspace: fix-masters-cms-grouping-2026-11-10
suffix: masters_cms_grouping_20261110
date: 2026-11-10
status: S6_DONE
---

# 08. Tasks — Gom nhóm hiển thị trang `/masters` trên CMS theo DB Master -> Schema/Database -> Table

## Danh sách công việc chi tiết
- [x] **Task 1 (S2 Investigate & S2.5 DG0):**
  - **Context:** `cdc-cms-web/src/pages/MasterRegistry.tsx:1163-1171`, `1292-1334`; `cdc-cms-web/src/components/master/masterEngineConfig.ts:1-86`; `cdc-cms-service/internal/infra/persistence/master/master_read_repo_gorm.go:28-106`.
  - **Verification:** Chạy `laya_gate.py run --stage dg0` đạt `PASS`.
- [x] **Task 2 (S3 Solution & DG1/DG2):**
  - **Context:** Thiết kế giải pháp gom nhóm 3 tầng (`DB Master` -> `Schema / Database` -> `Table / Collection`) kèm icon cho từng loại Engine (`postgresql`, `mysql`, `clickhouse`, `mongodb`) và icon cho tầng Namespace/Object trong `09_tasks_solution_masters_cms_grouping_20261110.md`.
  - **Verification:** Chạy `laya_gate.py run --stage dg1` đạt `PASS`, hiển thị Code Demo trên chat và chờ User `APPROVE`.
- [x] **Task 3 (S4 Execute — sau khi có APPROVE):**
  - **Context:**
    1. Cập nhật `cdc-cms-web/src/components/master/masterEngineConfig.ts` thêm `brandColor`, `deviconSlug`, `namespaceLabel`, `namespacePluralLabel`, `objectLabel`, `objectPluralLabel`.
    2. Tạo mới `cdc-cms-web/src/components/master/MasterGroupedCollapse.tsx` chứa `MasterEngineIcon`, `MasterNamespaceIcon`, `MasterObjectIcon`, `buildMasterHierarchyGroups`, `MasterGroupedCollapse`.
    3. Cập nhật `cdc-cms-web/src/pages/MasterRegistry.tsx` sử dụng `MasterGroupedCollapse` và sửa 3 lỗi thuộc tính `MasterEngineMeta` (`shortLabel`, `displayNamePlaceholder`, `defaultUserPlaceholder`).
  - **Test / Logs:** Chạy `npm run build` (`tsc -b && vite build`) qua `evidence_run.py` (`EV-20261110T095503-e3f8`).
  - **DoD:** Build pass 100% (`exit code = 0`), hiển thị đúng phân cấp 3 tầng và icon.
- [x] **Task 4 (S5 Audit & S6 Close):**
  - **Context:** Thực hiện 6 bước QC phản biện cuối workflow, chạy `laya_gate.py run --stage dg3`, xuất `audit_report_masters_cms_grouping_20261110.md`, chạy `verify_governance.py`.
  - **DoD:** `verify_governance.py` đạt `PASSED` 🟢.
