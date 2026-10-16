# KẾ HOẠCH TRIỂN KHAI CHI TIẾT (IMPLEMENTATION PLAN)

## Bước 1: Cập nhật `source_count_runner.go`
- File: `centralized-data-service/internal/service/sourcecount/source_count_runner.go`
- Nhập thêm package `centralized-data-service/internal/activity`, `centralized-data-service/internal/service/governance`, `centralized-data-service/internal/service/source`.
- Bổ sung `connMgr` và `activityLogger` vào struct `SourceCountRunner` và constructor `NewSourceCountRunner`.
- Thay đổi logic `case "master"`: Gọi `connMgr.GetMasterDB(queryCtx, params.ConnectionCode)` thay vì `r.dbMaster`.
- Thay đổi logic `case "shadow"`: Gọi `connMgr.GetShadowDB(queryCtx, params.ConnectionCode)` thay vì `r.dbShadow`.
- Thêm gọi `activityLogger.Quick` khi lỗi và khi thành công.

## Bước 2: Cập nhật `confluence_runner.go`
- File: `centralized-data-service/internal/service/confluence/confluence_runner.go`
- Nhập thêm `centralized-data-service/internal/activity`, `centralized-data-service/internal/service/governance`.
- Bổ sung `activityLogger` vào struct `ConfluenceReportRunner` và constructor `NewConfluenceReportRunner`.
- Thêm gọi `activityLogger.Quick` khi lỗi và khi thành công.

## Bước 3: Cập nhật `server_jobs.go` & `server_setup.go`
- File: `centralized-data-service/internal/server/server_jobs.go`
  - Thêm tham số `connMgr *servicesource.ConnectionManager` vào `SetupSchedulerJobs`.
  - Truyền `connMgr` và `activityLogger` vào `NewSourceCountRunner` và `NewConfluenceReportRunner`.
- File: `centralized-data-service/internal/server/server_setup.go`
  - Dòng 592: Truyền `connectionManager` vào `SetupSchedulerJobs`.

## Bước 4: Tối ưu Poller Ticker trong `server_scheduler.go`
- File: `centralized-data-service/internal/server/server_scheduler.go`
  - Thay đổi `time.NewTicker(10 * time.Second)` $\rightarrow$ `time.NewTicker(2 * time.Second)`.

## Bước 5: Cải thiện Polling Frontend trong `ActivityManager.tsx`
- File: `cdc-cms-web/src/pages/ActivityManager.tsx`
  - Thêm nhịp polling: 1000ms, 2500ms, 4500ms sau khi bấm "Chạy ngay".

## Bước 6: Compile & Runtime Verification G1–G8
- Biên dịch lại `centralized-data-service` (`go build`).
- Biên dịch lại `cdc-cms-web` (`npm run build`).
- Chạy test kiểm thử thực tế trên schedule id=7 (`export_jobs_2`).
- Đối soát dữ liệu trong PostgreSQL (`cdc_activity_log`, `daily_source_counts`).
