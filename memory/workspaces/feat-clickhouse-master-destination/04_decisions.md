# 04 Architectural Decision Records (ADRs) - ClickHouse Master

> **Workspace**: `feat-clickhouse-master-destination`

---

## ADR-01: Lựa chọn Table Engine `ReplacingMergeTree` có hỗ trợ `is_deleted`
- **Bối cảnh**: ClickHouse là hệ thống append-only. Thao tác `UPDATE` hoặc `DELETE` truyền thống thông qua mutation (`ALTER TABLE UPDATE/DELETE`) chạy bất đồng bộ và tiêu tốn tài nguyên cực lớn.
- **Quyết định**: Sử dụng engine `ReplacingMergeTree(_version, _deleted)`.
  * Mỗi lần có `UPDATE`, chèn 1 dòng mới với `_version` cao hơn.
  * Mỗi lần có `DELETE`, chèn 1 dòng mới với `_deleted = 1` và `_version = source_ts`.
- **Hệ quả**: ClickHouse background merge tự động dọn dẹp các dòng cũ. Khi query phân tích chỉ cần thêm từ khoá `FINAL` hoặc mệnh đề `WHERE _deleted = 0`.

---

## ADR-02: Sử dụng Native Protocol (Port 9000) qua `clickhouse-go/v2`
- **Bối cảnh**: ClickHouse hỗ trợ HTTP interface (port 8123) và Native TCP protocol (port 9000).
- **Quyết định**: Dùng Native TCP protocol qua driver chính hãng `github.com/ClickHouse/clickhouse-go/v2`.
- **Lý do**: Native TCP protocol cung cấp cơ chế streaming block nhị phân nén (lz4), cho thông lượng ghi (throughput) cao gấp 3-5 lần so với HTTP JSON/CSV payload.

---

## ADR-03: Đa hình Master Destination trong ConnectionManager (Polymorphic Destination)
- **Bối cảnh**: Hệ thống đang mặc định `MasterDB` trả về `*gorm.DB` PostgreSQL.
- **Quyết định**: Mở rộng `ConnectionManager` để hỗ trợ đa hình:
  * Trả về PostgreSQL connection khi `engine_type == "postgresql"`.
  * Trả về ClickHouse `driver.Conn` khi `engine_type == "clickhouse"`.
- **Lý do**: Đảm bảo nguyên tắc Open/Closed Principle — mở rộng thêm đích mới mà không làm hỏng code PostgreSQL cũ.
