# 08 Tasks — `fix-target-db-connections-ui-2026-11-17`

## Task List

### T1 — S2 Investigate: Định vị chính xác Component / Table `Target Database Connections` và Kiểm tra Ràng buộc Kiến trúc (`file:line`) [DONE]
- **Context:**
  - Tra cứu `Quản Lý Kết Nối Đích` / `Target Database Connections` trong `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/` (`src/pages/MasterRegistry.tsx`, `src/components/master/`, `src/types/`).
  - Kiểm tra cấu trúc `columns` hiện tại của bảng kết nối đích: cột Mã kết nối, cột Tên hiển thị, cột Engine (icon + label + màu sắc), cột Host/Port (bao gồm edge-case `host`/`port` rỗng, URI/cluster host của MongoDB, multi-host, port `0` hoặc `undefined`), cột Database/Schema, Trạng thái, Thao tác, cũng như logic search/filter/sort nếu có gắn `dataIndex` hoặc `key`.
  - Kiểm tra số dòng (`wc -l`) của file chứa bảng hiện tại và quy tắc `forbidden_ifelse_files: ["src/pages/MasterRegistry.tsx"]` trong `cdc-cms-web/.agent-project.json:23`.
- **Logs/Test:** Chỉ đọc mã nguồn (`grep`, `read_file`, `wc -l`) theo chuẩn S2 Read-Only.
- **Definition of Done:** Xác định chính xác 100% `file:line` của bảng `Target Database Connections`, danh sách toàn bộ các cột hiện tại, nguồn icon/tên engine, cách format `host:port` + edge cases, và yêu cầu tách component SRP ra `src/components/master/` để giảm LOC của `MasterRegistry.tsx` (`NOW <= HEAD`) và không vi phạm `forbidden_ifelse_files`.

### T2 — S2.5 Pre-Plan Critique (DG0) & S3 Solution Design + Visual UI Preview + DG1/DG2 [DONE]
- **Context:**
  - Chạy phản biện ngược `DG0` đối chiếu các edge cases thực tế trong dữ liệu `MasterConnectionItem` (ví dụ: `display_name` rỗng hoặc trùng `connection_code`, `host` đã chứa `:port` hoặc `port` rỗng/`0` khi cấu hình qua DSN/MongoDB SRV, icon + tên hiển thị chuẩn của từng `engine_type`, và lệnh cấm `if/else` trong `MasterRegistry.tsx`).
  - Thiết kế **ĐÚNG 1 PHƯƠNG ÁN TỐI ƯU NHẤT** trong `09_tasks_solution_target_db_connections_ui.md` kèm bản mẫu bố cục trực quan (`L094`) và **Code Demo** đầy đủ.
- **Logs/Test:** Chạy `$T/laya_gate.py run --stage dg0` và `$T/laya_gate.py run --stage dg1` đạt `PASS`.
- **Definition of Done:** Lưu `09_tasks_solution_target_db_connections_ui.md`, append tiến độ vào `05_progress_target_db_connections_ui.md`, hiển thị trực tiếp bản mẫu bố cục + Code Demo trên khung chat và **DỪNG CHỜ `APPROVE`**.

### T3 — S4 Execute + S5 Final Adversarial Audit (Checks/Build + Runtime/Visual Check + DG3 + Security) + S6 Close [DONE]
- **Context:**
  - Tạo git restore-point local trên `cdc-cms-web` (`933396be8dde1dc35320611e5ec23e69aa376b1d`).
  - Thực thi `formatMasterConnectionHostPort` trong `src/components/master/masterConnectionForm.ts:74-90`, tạo `TargetConnectionsTable` & `TargetConnectionsDrawer` (`size={840}` chuẩn AntD v6) trong `src/components/master/TargetConnectionsDrawer.tsx:1-197`, và kết nối vào `src/pages/MasterRegistry.tsx:1-31, 1553-1563` (`1895 -> 1775` dòng, `0` `if/else` mới, `0` `ellipsis` trên định danh kỹ thuật theo `L098`, `100%` top-level hooks theo `L095`).
- **Logs/Test:** Biên lai thực thi khớp `src_fingerprint = 22bfa5536a5f7bdd6d98606d544bb58bd2136cd655451e3e01d55c6dc3482b75`: `EV-20261117T165703-1b90` (`npx tsc -b --noEmit`), `EV-20261117T165713-c879` (`npx eslint` trên file chạm), `EV-20261117T165723-6d80` (`npm run build`), `EV-20261117T165731-2711` (SSR DOM render check); Decision Gate `06_laya_summary_dg3_20261117T165922.json`.
- **Definition of Done:** Lưu báo cáo hợp nhất `audit_report_target_db_connections_ui.md`, append `DONE` vào `05_progress_target_db_connections_ui.md`, và chạy `verify_governance.py --workspace fix-target-db-connections-ui-2026-11-17 --suffix target_db_connections_ui`.
