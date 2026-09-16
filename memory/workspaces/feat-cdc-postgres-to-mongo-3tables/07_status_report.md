# 07 Status Report: Pipeline CDC PostgreSQL → MongoDB (3 Tables)

> **Workspace**: `feat-cdc-postgres-to-mongo-3tables`  
> **Trạng thái**: 🟡 Đang lập kế hoạch & Thiết kế chi tiết (Planning & Technical Design)  
> **Cập nhật lần cuối**: 2026-09-10T16:30:00+07:00

---

## 1. Tóm tắt Trạng thái Dự án
- Đã hoàn tất Audit kỹ thuật chuyên sâu theo tiêu chuẩn Staff / Principal Engineer.
- Đã loại bỏ 4 lỗi kiến trúc chí mạng (Topic ordering fallacy, Delete key loss, Zombie Document, Root-level array versioning) và 2 điểm over-engineering (Cartesian JOIN và Stateful in-memory debounce).
- Đã khởi tạo đầy đủ bộ tài liệu Governance theo Rule #4.
- Đang trình User duyệt `12_implementation_plan_cdc_pg_to_mongo_3tables.md` trước khi bàn giao cho Muscle triển khai code.

## 2. Tiến độ theo Khối Công việc

| Khối Công việc | Trạng thái | Tiến độ | Ghi chú |
|:---|:---:|:---:|:---|
| **1. Hạ tầng & Debezium Connector** | 🟡 Ready for Deployment | 80% | Đã hoàn tất file config SMT, chờ apply lên Kafka Connect |
| **2. Stateless Worker Engine** | 🟡 Ready for Code | 50% | Đã thiết kế logic `ProcessBatch`, chờ code |
| **3. Snapshot Backfill Lateral SQL** | 🟡 Ready for Code | 70% | Đã có câu SQL Lateral, chờ tích hợp vào `snapshot_runner` |
| **4. Reconciliation & Testing** | ⚪ Pending | 0% | Chờ hoàn thành worker và backfill để test E2E |

## 3. Rủi ro & Giải pháp Kiểm soát (Risks & Mitigations)
- **Rủi ro lag khi lượng write Postgres quá lớn**: Đã xử lý bằng cách gom batch ở consumer và thực thi `BulkWrite` có thứ tự (`SetOrdered(true)`), giảm số lượng network round-trips tới MongoDB.
- **Rủi ro mất partition key khi delete**: Đã kiểm soát qua SMT `ExtractNewRecordState` với `delete.handling.mode: rewrite`.
