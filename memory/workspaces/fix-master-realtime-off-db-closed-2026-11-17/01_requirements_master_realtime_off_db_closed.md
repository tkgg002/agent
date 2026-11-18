---
tier: T2
mode: standard
suffix: master_realtime_off_db_closed
workspace: fix-master-realtime-off-db-closed-2026-11-17
created_at: 2026-11-17T15:45:10+07:00
---
# 01 Requirements — Kiểm Tra Nguyên Nhân Vẫn Phát Sinh Log Lỗi `transmute` / `transmute_sub` / `transmute_poison_pill` (`sql: database is closed`) Dù Đã Tắt Sync Realtime Ở Master

## 1. Yêu cầu của User
- **Thao tác thực tế của User:** Đã tắt **Sync Realtime** ở Master (`master_binding` / Master Registry UI).
- **Hiện tượng lỗi thực tế (tại `15:40:28` – `15:40:35 17/11/2026`):**
  - Dù đã tắt Sync Realtime ở Master, Activity Log vẫn ghi nhận hàng loạt bản ghi lỗi liên tiếp (triggered bởi `kafka-consumer-hook`):
    - `transmute`, `transmute_sub`, `transmute_poison_pill`
    - Scope 1: `shadow_payment_bill_service.payment_bills` -> `public.cdc_payment_bill_payload` & `public.cdc_payment_bill` (`source_ids=[64760]`, `[64761]`)
    - Scope 2: `shadow_payment_service.payments` -> `public.cdc_payment` & `public.cdc_payment_payload` (`source_ids=[43540]`, `[43541]`)
  - Nội dung lỗi chi tiết:
    - `failed to verify master table existence: sql: database is closed`
    - `poison pill isolated (source_ids=[...]): failed to verify master table existence: sql: database is closed`
- **Câu hỏi trọng tâm của User:**
  > "sao tao đã tăt sync realtime ở master. nó vẫn có những log lỗi này, kiểm tra cho tao, lại phá gì rồi"
  - Cần điều tra và làm rõ bằng bằng chứng `file:line` + trạng thái DB/runtime thực tế trên cả 3 câu hỏi gốc rễ:
    1. **Tại sao đã tắt Sync Realtime ở Master mà `kafka-consumer-hook` vẫn kích hoạt `transmute` / `transmute_sub`?** (Kiểm tra cờ tắt realtime được ghi ở đâu trên UI / `cdc-cms-service` / bảng `cdc_system.master_binding` hoặc `transmute_schedule`, cache in-memory nào chưa invalidate hoặc consumer/debouncer/fanout nào bỏ qua cờ realtime sync, hay jetstream backlog / multi-binding nào vẫn bắn message).
    2. **Tại sao lại văng lỗi `failed to verify master table existence: sql: database is closed`?** (Kiểm tra tiến trình daemon live đang chạy binary cũ hay mới sau bản sửa `fix-master-connection-edit-db-closed-2026-11-17` theo `L065`/`L100`, hoặc luồng code mới trong `connection_manager` / `retireGormPoolGracefully` / `TransmuterModule` / `MasterDDLGenerator` vừa gây regression làm đóng `sql.DB`!).
    3. **Tại sao lỗi hạ tầng kết nối `sql: database is closed` vẫn bị cô lập nhầm thành `transmute_poison_pill`?** (Kiểm tra `transmute_poison_pill_split.go` và binary đang chạy thực tế).

## 2. Ràng buộc Thực thi (Technical Constraints từ `GEMINI.md` & `lessons/INDEX.md`)
- **L065 & L100 (Honest Process State, Environment Lock & Restart Live Binary):** BẮT BUỘC kiểm tra ngay trạng thái tiến trình daemon đang chạy thực tế (`ps`, `lsof`, thời điểm build/start của binary `centralized-data-service` và `cdc-cms-service` so với commit/edit lúc `14:20:10`), môi trường DB đang kết nối, và đối chiếu trung thực 100% giữa code trên đĩa vs binary đang chạy trên RAM!
- **L074 & Rule #20 (End-to-End Secret & PII Masking):** Tuyệt đối che mật khẩu/token/URI (`postgres://***:***@host:port/db`) khi đọc cấu hình/DB và khi lưu vào `agent/memory/`.
- **L087 & L088 (JetStream FIFO, TableDebouncer & Poison Pill Guard):** Không để lỗi hạ tầng DB (`sql: database is closed`) nuốt sự kiện CDC vào poison pill DLQ làm sai lệch dữ liệu đích; kiểm tra hàng đợi NATS JetStream `CDC_TRANSMUTE` (`cdc.cmd.transmute-shadow`) và `TableDebouncer`.
- **L101 (Pure Forward Core Fix):** Giải quyết sạch tận gốc luồng xử lý lõi hiện tại và tương lai, không nhồi band-aid bẩn hay cheat DB trực tiếp (Rule #12).
- **Rule #1, #9, #13 (Brain Propose-Only):** Điều tra chỉ đọc ở S2, phản biện ngược ở S2.5 (`DG0`), lập **ĐÚNG 1 PHƯƠNG ÁN TỐI ƯU NHẤT** kèm **Code Demo** trong `09_tasks_solution_master_realtime_off_db_closed.md` + hiển thị trên chat, qua cổng `DG1/DG2` và **DỪNG CHỜ `APPROVE`** trước khi chạm vào bất kỳ dòng source code nào.
