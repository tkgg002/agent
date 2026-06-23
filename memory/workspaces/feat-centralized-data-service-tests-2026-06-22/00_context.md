# Context: Centralized Data Service Go Tests

## 1. Bối cảnh & Mục tiêu
Người dùng yêu cầu bổ sung toàn bộ test Go cho hệ thống `centralized-data-service` tại thư mục `/Users/trainguyen/Documents/work/data-hub/centralized-data-service`.
Mục tiêu là tăng độ bao phủ (coverage) kiểm thử cho các packages cốt lõi (handler, service, repository, model, base, orchestration, recon, v.v.), đảm bảo các logic nghiệp vụ và kỹ thuật hoạt động chính xác, ổn định và không phát sinh lỗi regression.

## 2. Hệ Quy Chiếu Cấu Trúc
Dựa theo `Note1.ini`:
- `base/` (Infra/Utils): Tầng đáy vô tri. Chứa các HTTP helpers, DB utilities, SQL sanitization.
- `source/` (Source Connectors & Discovery): Đăng ký nguồn, registry, Discover/Infer cấu trúc DB gốc.
- `master/` (Master Swaps & Sync Target): DDL cấu trúc cuối và đổ dữ liệu vật lý (Data Plane).
- `shadow/` (Shadow Ingestion & Buffer): Bảng đệm Ingestion và Kafka.
- `recon/` (Data Reconcile, DLQ & Self-Healing): Đối soát, DLQ, Backfill.
- `orchestration/` (State Machine & Scheduler Jobs): Control Plane, State Machine, Log hoạt động.
