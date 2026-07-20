# Kế hoạch tối ưu hóa SQL chậm và sửa lỗi thống kê UI

Kế hoạch này bao gồm 2 phần:
1. **Backend:** Tối ưu hóa các câu lệnh SQL chậm (Slow SQL) trong `recon_read_repo_gorm.go`.
2. **Frontend:** Sửa đổi logic tính toán các chỉ số thống kê (Tổng bảng, Khớp, Lệch) ở đầu trang Toàn vẹn dữ liệu trong `DataIntegrity.tsx`.

---

## User Review Required

> [!IMPORTANT]
> - **Backend:** Câu truy vấn `listLatestPrimary` ban đầu thực hiện quét toàn bộ bảng lịch sử lớn để distinct. Giải pháp đề xuất chuyển sang **Registry-driven Lateral Fetch** (sử dụng bảng `cdc_table_registry` làm driving table và `LEFT JOIN LATERAL` sử dụng index trên `shadow_table`) giúp giảm thời gian chạy từ ~1.2s xuống <15ms.
> - **Frontend:** Sửa đổi cách tính toán số lượng thống kê bằng cách gom nhóm các segment report thành Pipeline đại diện cho mỗi bảng (thông qua hàm `buildPipelines` tái sử dụng từ `ReconPipelineGrid.tsx`), giúp hiển thị chính xác số lượng **Bảng** thay vì số lượng **Segment chặng** bị nhân đôi.

---

## Proposed Changes

### Component: cdc-cms-service (Backend)

#### [MODIFY] [recon_read_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go)
- Sửa đổi `ListFailedLogs` để đếm trực tiếp trên bảng `failed_sync_logs` thay vì dùng subquery bọc lateral joins.
- Viết lại `listLatestPrimary` theo hướng **Registry-driven Lateral Fetch** (lọc theo registry active trước).

---

### Component: cdc-cms-web (Frontend)

#### [MODIFY] [ReconPipelineGrid.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx)
- Export interface `PipelineRow`.
- Export function `buildPipelines`.

#### [MODIFY] [DataIntegrity.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/DataIntegrity.tsx)
- Import `buildPipelines` từ `ReconPipelineGrid`.
- Tính toán mảng `pipelines` đại diện cho các bảng thực tế bằng `buildPipelines(reportList)`.
- Cập nhật chỉ số "Tổng bảng" hiển thị `pipelines.length`.
- Tính toán `okCount` (cả chặng A và chặng B nếu có đều ở trạng thái `ok` hoặc `ok_empty`).
- Tính toán `driftCount` (có bất kỳ chặng nào bị `drift`, `warning`, `dest_missing` hoặc chênh lệch `driftBC !== 0`).

---

## Verification Plan

### Automated Tests
- Khởi chạy frontend ở local và kiểm tra hiển thị số lượng ở đầu trang `http://localhost:5173/data-integrity`.
- Gọi API backend để kiểm tra tính đúng đắn và tốc độ phản hồi.

### Manual Verification
- Xác nhận các chỉ số hiển thị đúng: **Tổng bảng: 3, Khớp: 2, Lệch: 1, Lỗi đồng bộ: 0**.
- Đảm bảo khi click làm mới, dữ liệu vẫn được cập nhật chính xác.
