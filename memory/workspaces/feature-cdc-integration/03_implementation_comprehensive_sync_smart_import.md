# Implementation Plan: Phase 3 - Smart Import (Airbyte to CMS)

Tính năng này cho phép người dùng liệt kê các bảng (streams) hiện có trên Airbyte và đăng ký vào Registry của CMS chỉ với một cú click chuột.

## 1. Mục tiêu
- Giảm thiểu công sức nhập liệu thủ công khi onboarding hệ thống mới.
- Đảm bảo tính chính xác của tên bảng, namespace và connection ID.

## 2. Các thay đổi dự kiến

### Backend (cdc-cms-service)

#### [NEW] `AirbyteHandler.ListImportableStreams`
- API: `GET /api/airbyte/import/list`
- Logic:
    1. Gọi `airbyte.ListConnections` cho toàn bộ workspace.
    2. Đối soát với `cdc_table_registry` hiện tại.
    3. Trả về danh sách các stream kèm trạng thái: `already_registered` hoặc `ready_to_import`.

#### [NEW] `AirbyteHandler.ExecuteImport`
- API: `POST /api/airbyte/import/execute`
- Input: `connectionId`, danh sách `streamNames`.
- Logic:
    1. Lấy Full Catalog của connection từ Airbyte.
    2. Map từng stream sang `model.TableRegistry`.
    3. Gọi `repo.BulkCreate` để lưu vào DB.
    4. Trigger `create_all_pending_cdc_tables()` để khởi tạo schema CDC.

### Database
- Đảm bảo function `create_all_pending_cdc_tables()` hoạt động chính xác cho các bảng mới import.

## 3. Câu hỏi thảo luận

- **Mapping Rule**: Khi import bảng mới, chúng ta có nên tự động tạo luôn một Mapping Rule mặc định (Standardize) không?
- **Target Table Name**: Chúng ta sẽ chuẩn hóa tên bảng đích (Target Table) theo quy tắc nào? (Ví dụ: Giữ nguyên tên gốc hay thêm tiền tố/hậu tố).

## 4. Kế hoạch xác minh (Verification)
- Kiểm tra API danh sách import xem có nhận diện đúng bảng chưa đăng ký không.
- Thực hiện Import 1-2 bảng và kiểm tra DB + Schema CDC.
- Kiểm tra log NATS xem `schema.config.reload` có được gửi đi không.
