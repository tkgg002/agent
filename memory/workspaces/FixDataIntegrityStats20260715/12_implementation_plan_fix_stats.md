# AI Implementation Plan: Sửa lỗi thống kê UI Data Integrity

Nhiệm vụ này tập trung vào sửa đổi logic tính toán các chỉ số thống kê (Tổng bảng, Khớp, Lệch) ở đầu trang Toàn vẹn dữ liệu trong frontend.

---

## Phân tích & Giải pháp kỹ thuật

### 1. Nguyên nhân gốc rễ
Hiện tại, trang `DataIntegrity.tsx` đang tính toán thống kê trực tiếp trên mảng `reportList` (raw reports nhận được từ API). Mỗi bản ghi trong `reportList` tương ứng với một chặng đối soát (`segment` là `source_shadow` hoặc `shadow_master`).
Do một bảng có thể có cả 2 chặng đối soát hoạt động song song, mảng này chứa nhiều bản ghi cho cùng một bảng. Điều này dẫn đến việc số lượng bảng hiển thị bị đúp hoặc sai lệch.

Trong khi đó, Grid hiển thị bên dưới (`ReconPipelineGrid.tsx`) đã gom nhóm các segment này thành các **Pipeline** đại diện cho từng bảng bằng hàm `buildPipelines`.

### 2. Giải pháp sửa đổi

#### Bước 2.1: Sửa đổi `ReconPipelineGrid.tsx`
Export các type và hàm xử lý để `DataIntegrity.tsx` có thể tái sử dụng, tránh duplicate code:
- Export `PipelineRow` interface.
- Export `buildPipelines` function.

#### Bước 2.2: Sửa đổi `DataIntegrity.tsx`
- Import `buildPipelines` từ `../components/ReconPipelineGrid`.
- Tính toán mảng `pipelines` đại diện cho các bảng thực tế:
  ```typescript
  const pipelines = useMemo(() => buildPipelines(reportList), [reportList]);
  ```
- Cập nhật cách tính `okCount` và `driftCount` dựa trên trạng thái của pipeline:
  - Một bảng được coi là **Khớp** khi cả chặng A và chặng B (nếu có master) đều ở trạng thái `ok` hoặc `ok_empty`.
  - Một bảng được coi là **Lệch** khi có bất kỳ chặng nào bị `drift`, `warning`, `dest_missing` hoặc có độ lệch dữ liệu `driftBC !== 0`.
- Hiển thị giá trị `pipelines.length` cho chỉ số "Tổng bảng".

---

## Kế hoạch triển khai (Implementation Plan)
1. **Brain:** Tạo file `implementation_plan.md` ở dạng Artifact và đồng bộ vào Workspace dự án.
2. **User:** Review và Duyệt kế hoạch.
3. **Muscle (Subagent):** Sửa đổi code trong:
   - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx`
   - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/DataIntegrity.tsx`
4. **Muscle (Subagent):** Chạy và xác thực code build thành công.
5. **QA (Subagent):** Xác thực giao diện hiển thị đúng thống kê (Tổng bảng: 3, Khớp: 2, Lệch: 1, Lỗi đồng bộ: 0).
6. **Brain:** Chạy linter quy trình và báo cáo kết quả.
