# 12_IMPLEMENTATION_PLAN: KẾ HOẠCH TRIỂN KHAI CHI TIẾT

## 1. Migration DDL
- `cdc-cms-service/migrations/schema/recon_dlq/103_add_total_rows_to_jobs.sql`

## 2. Worker Service (`centralized-data-service`)
- `internal/repository/transform_job_repo.go`: Bổ sung TotalRows.
- `internal/repository/transmute_job_repo.go`: Bổ sung TotalRows.
- `internal/handler/shadow/batch_transform_handler.go`: Bổ sung TraceID payload, đếm trước pending rows, realtime progress.
- `internal/service/master/transmuter.go`: Bổ sung countShadowRows, realtime progress.

## 3. CMS Backend (`cdc-cms-service`)
- `internal/infra/persistence/transform_job_repo.go` & `transmute_job_repo.go`.
- `internal/api/source/source_object_actions_handler.go` & `master_transmute_job_handler.go`.
- `internal/infra/persistence/source/source_object_read_repo_gorm.go` & `master_read_repo_gorm.go`.
- `internal/app/queries/source/source_objects_read_models.go` & `list_masters.go`.

## 4. CMS Frontend (`cdc-cms-web`)
- `src/types/index.ts`.
- `src/pages/TableRegistry.tsx` (`TransformJobStatus`).
- `src/pages/MasterRegistry.tsx` (`TransmuteJobStatus`).
