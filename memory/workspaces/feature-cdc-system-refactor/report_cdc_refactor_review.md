# Báo cáo Audit: CDC System Refactor Phase 2 v2
Date: 2026-05-07
Workspace: feature-cdc-system-refactor

## 1. Tình trạng hiện tại của cdc-cms-service (API)
- **Pillar P1 (Structure & Interfaces)**: Đã được khởi tạo. Các thư mục `internal/domain`, `internal/app/ports`, `internal/app/queries`, `internal/app/commands`, `internal/infra` đều tồn tại.
- **Pillar P2 & P3 (CQRS Handlers)**: **CHƯA HOÀN THÀNH TOÀN DIỆN**.
  - Các handler trong `internal/api/` vẫn còn chứa rất nhiều dòng code và logic nghiệp vụ. Ví dụ: `mapping_rule_handler.go` (659 dòng), `reconciliation_handler.go` (638 dòng), `master_registry_handler.go` (594 dòng), `registry_handler.go` (684 dòng). 
  - Chưa đạt Definition of Done (DoD): `wc -l internal/api/*.go` mọi file ≤ 100 dòng.
- **Pillar P4 (Persistence Clean-up)**: Đã xóa thư mục `internal/service`, đẩy logic query xuống `internal/infra/persistence`. Đã loại bỏ phần lớn `db.Raw` và `db.Exec` trong api layer. Tuy nhiên, layer API vẫn còn dính chặt với logic phức tạp, chưa trở thành "thin adapter".

## 2. Tình trạng hiện tại của centralized-data-service (Worker)
- **Worker Handlers**: Đã kiểm tra `internal/handler/`. Các subscription NATS cho lệnh mới là `cdc.cmd.master-swap` và `cdc.cmd.v2-sync` **hoàn toàn chưa tồn tại**.
- **Job Monitor & Events**: Hệ thống đã có handler bắt lệnh (như standardize, backfill, v.v.), nhưng chưa phát (emit) đầy đủ companion events `cdc.evt.*.completed`. Bảng `cdc_jobs` để theo dõi tiến độ Job chưa được tích hợp hoàn chỉnh vào các luồng này.

## 3. Tình trạng hiện tại của cdc-cms-web (Frontend)
- **Frontend Async Support**: Kiểm tra source code (`src/`), không phát hiện logic poll (gọi lặp) endpoint `/api/jobs/:id`. 
- **Thiếu sót của Plan gốc**: Plan gốc ghi rõ `cdc-cms-web` là "Out of scope". Tuy nhiên, nếu BE chuyển các hành động nặng (như master-swap, v2-sync, recon-check) sang trả về HTTP 202 Accepted + `job_id`, FE **bắt buộc** phải được nâng cấp để xử lý UI "Processing..." và poll trạng thái từ API. Nếu không, FE sẽ bị hỏng hiển thị (Facade feature - Lesson 2026-04-16).

## 4. Cập nhật Plan (Đề xuất)
Dựa trên kết quả thực tế, hệ thống vẫn đang kẹt ở giữa Phase 2 v2. Do đó, Kế hoạch cần được cập nhật:
1. **Tiếp tục P2 & P3 triệt để**: Refactor 100% các file lớn trong `internal/api/` thành CQRS calls.
2. **Triển khai P3 trên Worker**: Thêm các NATS subscriber và event emitters cho master-swap, v2-sync.
3. **Thêm Pillar P5 (Frontend Async Integration)**: Loại bỏ FE khỏi danh sách "Out of scope". Thêm task cập nhật React components để xử lý HTTP 202 và polling trạng thái `cdc_jobs`.
