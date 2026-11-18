# 08 Tasks — `fix-master-connection-edit-db-closed-2026-11-17`

## Task List

### T1 — S2 Investigate: Kiểm tra thực trạng Database `cdc_system.*` & Luồng Edit Master Connection + Transmute Pool Cache
- **Context:**
  - `cdc-cms-service/internal/api/master/` (`master_registry_handler_connections*.go`, `CreateMasterCommand`, `UpdateConnection`)
  - `centralized-data-service/internal/service/source/connection_manager.go` (`GetMasterDB`, `InvalidatePool`)
  - `centralized-data-service/internal/service/master/` (`transmuter.go`, `master_ddl_generator.go`, poison-pill isolation)
  - `centralized-data-service/internal/server/server_setup.go` (`connection.config.changed` subscriber)
  - Database `cdc_system.*` (`master_connections`, `connection_registry`, `master_binding`, `transmute_schedule`, ...) cho `postgrestestingpaymentv2` (Read-Only, mask tuyệt đối secret/host).
- **Logs/Test:** Truy vấn Read-Only metadata bảng `cdc_system.*` (mask credentials) + đối chiếu log worker & `file:line`.
- **Definition of Done:** Chỉ rõ chính xác 100% bảng/cột nào trong DB đang lưu cứng thông tin kết nối không update khi edit `Master Connection`, và chính xác dòng code nào gây ra lỗi `failed to verify master table existence: sql: database is closed` + `transmute_poison_pill`.

### T2 — S2.5 Pre-Plan Critique (DG0) & S3 Solution Design + DG1/DG2 [DONE]
- **Context:** Đối chiếu giả định "chỗ nào trong DB đang hard" vs cache/pointer `*gorm.DB` trong bộ nhớ runtime của Worker (`DG0`), lập giải pháp duy nhất tối ưu trong `09_tasks_solution_master_conn_edit_db_closed.md` kèm Code Demo.
- **Logs/Test:** Chạy `$T/laya_gate.py run --stage dg0` và `$T/laya_gate.py run --stage dg1` đạt `PASS`.
- **Definition of Done:** Lưu `09_tasks_solution_master_conn_edit_db_closed.md`, ghi log vào `05_progress_master_conn_edit_db_closed.md`, trình bày kết quả kiểm tra + Code Demo rõ ràng cho User và dừng chờ `APPROVE`.

### T3 — S4 Execute + S5 Adversarial Audit (Self-Healing + DG3 + Security) + S6 Close [DONE]
- **Context:** Triển khai `F1..F5` trên `centralized-data-service` (`multi.go` với `PeekDB` + `EvictRole`, `connection_manager.go` với `m.closed` WaitGroup guard + zero-I/O `PeekDB`, `recon_engine.go`, `recon_stream_bucket_engine.go`, `server_setup.go`, `transmute_poison_pill_split.go`), `cdc-cms-service` (`binding.go`, `master_connection_repo_gorm.go`, `master_read_repo_gorm.go`, `master_registry_handler_connections.go`, `master_registry_handler_connections_probe.go`), và `cdc-cms-web` (`MasterRegistry.tsx`).
- **Logs/Test:** Kiểm thử tích hợp Real PostgreSQL 15 với `-race` (`EV-20261117T115017-819e`, `EV-20261117T114325-a969`), build & vet toàn bộ 3 services (`EV-20261117T115030-8dc6`, `EV-20261117T115044-3308`, `EV-20261117T115258-d53d`, `EV-20261117T114338-4ff8`, `EV-20261117T114459-ae4e`, `EV-20261117T114350-c0d0`), chạy `DG3` (`06_laya_summary_dg3_20261117T115147.json`).
- **Definition of Done:** Hoàn tất `audit_report_master_conn_edit_db_closed.md`, ghi `DONE` vào `05_progress_master_conn_edit_db_closed.md`, và `verify_governance.py` đạt `PASSED` 🟢.

### T4 — Post-Done Adversarial Audit, Self-Improvement Loop & `/laya_router` QC [DONE — PROPOSE-ONLY]
- **Context:**
  - Phản biện lại toàn bộ quá trình thực thi `T1..T3`, kiểm tra `HEAD vs NOW` số dòng từng file theo `L049` và Design Pattern gốc (`system_overview_and_design_pattern.md` Rules `DS-1..DS-4`, `CMS-1..CMS-3`, `cdc-cms-web`).
  - Đối chiếu từng dòng update tại `connection_manager.go:463-483, 555-579`, `server_setup.go:147, 169, 228-242, 518`, `recon_stream_bucket_engine.go:192-200`, `transmute_poison_pill_split.go:274-275` + `transmuter.go:372, 879`, `master_connection_repo_gorm.go:84-168, 330-455`, `master_registry_handler_connections.go:219-352`, `master_registry_handler_connections_probe.go:148-223`, `MasterRegistry.tsx:267-282, 430-450`.
- **Logs/Test:** Ghi bài học `L065` vào `lessons.md` + `build_lessons_index.py`, chạy `laya_gate.py` (`dg1` + `dg3`), kiểm tra `verify_governance.py`.
- **Definition of Done:** Chỉ rõ 6 phát hiện thực tế `A1..A6` không che giấu, minh bạch ranh giới kiểm thử PostgreSQL 15 local vs môi trường ngoài `postgrestestingpaymentv2`, lưu đầy đủ vào `audit_report_master_conn_edit_db_closed.md` + `09_tasks_solution_master_conn_edit_db_closed.md`, trình bày Code Demo và dừng chờ User `APPROVE` (`L027`).

### T5 — S4/S5 Refactoring & Bugfix Execution (`A1..A6`) Sau Khi Có `APPROVE` [DONE]
- **Context:**
  - `A1`: Đưa `m.reg.EvictRole("master:" + key)` vào trong `m.masterPoolMu.Lock()` ở `InvalidatePool` và `Close()` (`connection_manager_lifecycle.go`), đóng kín khe hở race hồi sinh `*gorm.DB` cũ.
  - `A2`: Tách `server_config_reload.go` đăng ký NATS sau khi `chunkEngine` khởi tạo xong (triệt tiêu data race biến capture `chunkEngine`) và nối `ws.RegisterOnStop(func() { connectionManager.Close() })` tại `server_setup.go:148` (LIFO shutdown).
  - `A3`: Tách file đơn trách nhiệm (SRP) đảm bảo mọi file lớn đều `NOW <= HEAD`: `connection_manager_lifecycle.go` (`connection_manager.go` `<= 495`), `recon_master_agent_cache.go` (`recon_stream_bucket_engine.go` `<= 1093`), `server_config_reload.go` (`server_setup.go` `<= 615`), `master_connection_options_sync.go` (`master_connection_repo_gorm.go` `<= 475`), `masterConnectionForm.ts` + re-export tương thích ngược (`MasterRegistry.tsx` `<= 1946`).
  - `A4`: Đưa logic thuần phân tích/ráp DSN về `internal/domain/master/connection_dsn.go` và tách use-case ghi `UpdateMasterConnectionCommand` / `RetireMasterConnectionCommand` về `internal/app/commands/master/update_master_connection.go` đúng Hexagonal CQRS (`Rule CMS-1`, `CMS-2`), bọc transaction nguyên tử cho `connection_registry` + `master_binding`.
  - `A5`: Hỗ trợ fallback kế thừa `username`/`password` từ cấu trúc `options_json` (khi không có khóa `"dsn"`) trong `ExtractMasterOptionsCredentials`.
  - `A6`: Bổ sung `"does not exist on target database"` (`transmuter.go:372, 879`) vào `isTransientError` (`transmute_poison_pill_split.go`) để chặn bão `binarySearchSplit` + false `transmute_poison_pill` khi đổi sang DB đích mới chưa chạy DDL.
- **Logs/Test:** Chạy lại toàn bộ test tích hợp Real PostgreSQL 15 với `-race`, `go build`, `go vet`, `npx tsc --noEmit` qua `evidence_run.py`, cập nhật `DG3` và `audit_report_master_conn_edit_db_closed.md`.
- **Definition of Done:** 100% file lớn đạt `NOW <= HEAD`, đúng layer chuẩn, 100% backward compatible, biên lai `EV-...` mới nhất pass sạch 0 race.
