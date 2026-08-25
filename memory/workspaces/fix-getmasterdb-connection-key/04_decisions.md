# 04_decisions.md — Architecture Decision Records (ADRs)

## ADR-01: Phân tách `master_schema` và `master_table` độc lập ở tầng DTO/Command

- **Status:** Accepted
- **Context:** Khi người dùng gửi request tạo schedule hoặc quản lý bảng master, có 2 cách thiết kế:
  - *Phương án A:* Gộp chung thành 1 chuỗi FQN `master_table: "schema.table"`.
  - *Phương án B:* Tách 2 field độc lập `master_schema` và `master_table`.
- **Decision:** Chọn **Phương án B** (Tách 2 field độc lập).
- **Rationale:**
  1. Giữ nguyên regex validation `schedNameRe` chặt chẽ cho từng identifier riêng biệt (`^[a-z_][a-z0-9_]{0,62}$`).
  2. Dễ dàng map trực tiếp vào schema database của bảng `cdc_system.master_binding` (có 2 cột riêng biệt `master_schema` và `master_table`).
  3. Tránh việc parse chuỗi `strings.Split` nhiều lần ở các tầng controller.
- **Consequences:** Tầng API Controller chịu trách nhiệm nhận cả 2 field và truyền vào Command.

---

## ADR-02: Chiến lược phòng thủ chuỗi nối PostgreSQL NULL-safe

- **Status:** Accepted
- **Context:** Toán tử `||` trong PostgreSQL trả về `NULL` nếu một trong hai toán hạng là `NULL`. Trong dữ liệu lịch sử hoặc các bảng default, `master_schema` có thể là `NULL` hoặc rỗng `""`.
- **Decision:** Sử dụng biểu thức chuẩn hóa:
  ```sql
  COALESCE(NULLIF(mb.master_schema, ''), 'public') || '.' || mb.master_table
  ```
- **Rationale:** Triệt tiêu hoàn toàn khả năng sinh ra chuỗi `NULL`, luôn đảm bảo NATS payload nhận được chuỗi định danh hợp lệ (fallback về `public.<table_name>`).
- **Consequences:** An toàn 100% với dữ liệu cũ và dữ liệu mới.

---

## ADR-03: Kỷ luật Core Systems: Nghiêm cấm Cheat DB thủ công trên Checkpoint

- **Status:** Accepted
- **Context:** Từng có đề xuất xóa thủ công bảng `cdc_system.sync_runtime_state` để ép full sync.
- **Decision:** Tuyệt đối không can thiệp bằng câu lệnh DELETE/UPDATE thủ công vào database.
- **Rationale:** Engine `TransmuterModule` đã có lifecycle tự động quản lý checkpoint và reset về `{}` khi kết thúc. Sửa data thủ công vi phạm Rule #12 và phá vỡ cơ chế tự phục hồi.
