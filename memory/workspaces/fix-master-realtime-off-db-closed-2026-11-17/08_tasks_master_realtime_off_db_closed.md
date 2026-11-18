# 08 Tasks — `fix-master-realtime-off-db-closed-2026-11-17`

## Task List

### T1 — S2 Investigate: Truy Vết Toàn Trình Luồng Tắt Sync Realtime Ở Master, Lỗi `sql: database is closed`, Poison Pill & Trạng Thái Tiến Trình Runtime (`file:line` + Live DB/Process) [DONE]
- **Context:**
  - Kiểm tra tiến trình live đang chạy (`ps`, `lsof`, Docker containers nếu có) của `centralized-data-service` và `cdc-cms-service` xem đang chạy binary/image lúc mấy giờ so với đợt sửa `fix-master-connection-edit-db-closed-2026-11-17` (`11:30`–`14:20 17/11/2026`), và kiểm tra `git log` / `git status` / `git diff` gần nhất trên cả 3 repo (`centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`).
  - Truy vết từ UI (`cdc-cms-web/src/pages/MasterRegistry.tsx` & components) -> API/Command (`cdc-cms-service`) -> Database (`cdc_system.master_binding`, `cdc_system.transmute_schedule`, `cdc_system.activity_logs`) -> NATS notification -> `centralized-data-service` (`BatchBuffer` fanout `CP-3`, `TransmuteHandler` / `TableDebouncer` / `TransmuterModule` `CP-4`) để xác định chính xác vì sao tắt Sync Realtime ở Master mà `kafka-consumer-hook` vẫn kích hoạt `transmute`, `transmute_sub`, `transmute_poison_pill` lúc `15:40:28`–`15:40:35`.
  - Truy vết hàm văng chuỗi lỗi `"failed to verify master table existence"` trong `centralized-data-service` và cơ chế quản lý pool `*gorm.DB` / `*sql.DB` (`ConnectionManager`, `retireGormPoolGracefully`, `multi_dialector`, `MasterDDLGenerator` / `TransmuterModule`) để tìm chính xác thủ phạm gây `sql: database is closed` và lý do lọt vào `transmute_poison_pill`.
- **Logs/Test:** Read-only (`grep`, `read_file`, `ps`/`lsof`, `docker ps`, truy vấn `SELECT` có mask secret trên `cdc_system`).
- **Definition of Done:** Xác định 100% `VERIFIED` bằng chứng `file:line` + DB state + Process PID/uptime trả lời trực diện cả 3 câu hỏi: (1) Ai/luồng nào làm lọt `kafka-consumer-hook` khi đã tắt Sync Realtime; (2) Chỗ nào gây `sql: database is closed` (do regression code mới hay binary cũ hay bug đóng nhầm shared `*sql.DB`); (3) Vì sao bị đánh dấu `transmute_poison_pill`.

### T2 — S2.5 Pre-Plan Critique (DG0) + S3 Single Best Solution + Code Demo + DG1/DG2 [DONE]
- **Context:**
  - Chạy phản biện ngược `DG0` kiểm chứng mọi giả định với ground truth code + runtime DB/process.
  - Thiết kế **ĐÚNG 1 PHƯƠNG ÁN TỐI ƯU NHẤT** ánh xạ 1-1 với từng phát hiện `F1..Fn` trong `09_tasks_solution_master_realtime_off_db_closed.md` kèm **Code Demo** đầy đủ tuân thủ Clean Layered (`centralized-data-service`), Hexagonal+CQRS (`cdc-cms-service`) và Feature/Hook (`cdc-cms-web`), bảo toàn `INV-1..INV-5` và `NOW <= HEAD` trên God-files.
- **Logs/Test:** `$T/laya_gate.py run --stage dg0` và `$T/laya_gate.py run --stage dg1` đạt `PASS`.
- **Definition of Done:** Lưu `09_tasks_solution_master_realtime_off_db_closed.md`, append kết quả vào `05_progress_master_realtime_off_db_closed.md`, hiển thị trực tiếp tóm tắt nguyên nhân + **Code Demo** trên khung chat, và **DỪNG CHỜ `APPROVE`**.

### T3 — S4 Execute + S5 Final Adversarial Audit + S6 Close [DONE]
- **Context:**
  - Tạo git restore-point local trên các repo cần sửa.
  - Thực thi giải pháp đã được User `APPROVE`, tuân thủ SRP, 0 `if/else` mới trong `forbidden_ifelse_files`.
- **Logs/Test:** Chạy kiểm thử và runtime check thật qua `$T/evidence_run.py` lấy biên lai `EV-...`, phản biện từng dòng `git diff` + `wc -l`, chạy `laya_gate.py run --stage dg3` + `laya_gate.py review`, quét bảo mật `/security-agent`.
- **Definition of Done:** Đạt đủ 8 cổng `G1–G8` và `DG3` kèm biên lai `EV-...` trong `audit_report_master_realtime_off_db_closed.md`, append `DONE` vào `05_progress_master_realtime_off_db_closed.md`, và `$T/verify_governance.py --workspace fix-master-realtime-off-db-closed-2026-11-17 --suffix master_realtime_off_db_closed` đạt `PASSED` 🟢.
