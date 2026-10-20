# 01 Requirements: Tích hợp Master MongoDB Destination vào Hệ thống Data Hub

tier: T2
- **Workspace**: `feat-master-mongo-destination-2026-10-20`
- **Ngày tạo**: 2026-10-20

## 1. Bối cảnh & Yêu cầu
Tổng hợp nhanh và đầy đủ danh sách các task kỹ thuật cần thực hiện để bổ sung **MongoDB làm Master Database Destination** trong hệ thống Data Hub (`cdc-cms-service`, `centralized-data-service`, `cdc-cms-web`), đối chiếu trực tiếp từ kiến trúc hiện tại và các hồ sơ liên quan.

## 2. Phạm vi (Scope)
1. **CMS Backend & Frontend**: Quản lý kết nối đích (`master-connections`) hỗ trợ `engine_type = "mongodb"` (Create, Update, Test Connection, UI Form).
2. **CDS Connection Pool & DDL/Index Provisioning**: Mở rộng `ConnectionManager` hỗ trợ `masterMongoPool` (`*mongo.Client`) và `MasterDDLGenerator` tự động tạo Collection + Indexes khi Approve Schema.
3. **Transmute Engine & Type Coercion**: Chuyển đổi kiểu dữ liệu sang BSON chuẩn và thực thi `BulkWrite` kèm kiểm soát OCC (`_source_ts`), Soft-Delete và xử lý mảng con.
4. **Reconciliation & Self-Healing**: Mở rộng `GetMasterAgent` và `recon_execute_heal_handler` để đối soát (Smoke A / Tier B) và tự phục hồi dữ liệu trên Master MongoDB.
