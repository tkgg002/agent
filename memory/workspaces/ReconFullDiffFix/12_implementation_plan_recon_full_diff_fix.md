# Kế hoạch triển khai - Chuẩn hóa phân loại đối soát sang TypeRecon và Tích hợp Full Diff lên Dashboard

Dự án này thực hiện hai nhiệm vụ chính:
1. **Chuẩn hóa phân loại đối soát:** Thay thế hoàn toàn cách đặt tên `tier` (0, 1, 2, 3) cũ bằng `type_recon` tường minh (`smoke`, `hash_window`, `full_diff`, `deep_check`, `prune`) xuyên suốt từ Frontend (`cdc-cms-web`), CMS backend (`cdc-cms-service`) tới CDS worker (`centralized-data-service`).
2. **Tích hợp và sửa đổi logic Full Diff:** Hợp nhất dữ liệu đối soát thủ công và tự động, sửa logic tính drift và hiển thị số lượng dòng của các hình thức quét theo window (như `full_diff`, `hash_window`) để tránh gây cảnh báo sai lệch toàn bảng.

## User Review Required

> [!IMPORTANT]
> 1. **Bỏ tham số `tier`, dùng `type_recon`:**
>    - Thay thế query parameter và payload field `tier` bằng `type_recon`.
>    - Cấu trúc lại toàn bộ handler tiếp nhận lệnh check `recon_check_handler.go` trên CDS dựa trên `type_recon` và `segment` để phân phối đúng nghiệp vụ.
> 2. **Hợp nhất nguồn dữ liệu (UNION ALL):**
>    - Thay đổi truy vấn SQL trong `recon_read_repo_gorm.go` của CMS để sử dụng `UNION ALL` giữa `cdc_recon_smoke_result` (smoke check) và `cdc_reconciliation_report` (manual check như full_diff/hash_window).
> 3. **Sửa logic đếm và drift của Full Diff:**
>    - `full_diff` là quét một chiều trong dynamic window. Hàm `TimeBoundedDiffMissingFromShadow` sẽ trả về thêm `destCount` (số lượng record tìm thấy trong Shadow DB).
>    - Report đối soát được lưu xuống DB sẽ lưu đầy đủ thông tin: `SourceCount` (số lượng trong cửa sổ MongoDB), `DestCount` (số lượng trong cửa sổ Shadow DB), `Diff` (số lượng record missing thực tế).
>    - Phía CMS backend và UI sẽ giữ nguyên trạng thái `status` được tính toán trực tiếp từ DB cho `full_diff` (nếu `MissingCount > 0` thì là `drift`, ngược lại là `success`/`ok`), đồng thời loại bỏ việc dùng window counts để làm tổng số lượng của bảng trên giao diện.

## Open Questions

Không có câu hỏi mở. Phương án này giải quyết triệt để sự chồng chéo giữa khái niệm "Tier" và "Check Type", tăng tính minh bạch và độ bền vững của mã nguồn.

## Proposed Changes

### 1. Centralized Data Service (CDS)

---

#### [MODIFY] [recon_check_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_check_handler.go)
- Đổi struct payload nhận từ NATS: thay `Tier string` bằng `TypeRecon string json:"type_recon"`.
- Cấu trúc lại toàn bộ logic phân phối tác vụ check dựa trên `TypeRecon` (`"prune"`, `"smoke"`, `"hash_window"`, `"full_diff"`, `"deep_check"`).
- Khi chạy `full_diff`, gọi `TimeBoundedDiffMissingFromShadow` để lấy `destCount` và populate đầy đủ `SourceCount`, `DestCount`, `Diff` vào report trước khi StampA.
- Cập nhật hàm `handleReconCheckSegmentB` để chạy ở chế độ deep check khi `TypeRecon == "deep_check"`.

#### [MODIFY] [recon_tier_a.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_a.go)
- Thay đổi signature của `TimeBoundedDiffMissingFromShadow` để trả về thêm số lượng bản ghi tìm thấy trong Shadow DB (`destCount`):
  `func (rc *ReconCore) TimeBoundedDiffMissingFromShadow(ctx context.Context, entry source.TableRegistry, startTime, endTime time.Time) (missing []string, srcCount int, destCount int, err error)`
- Trả về `len(shadowIDs)` làm `destCount`.

#### [MODIFY] [recon_heal_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_heal_handler.go)
- Cập nhật dòng gọi `TimeBoundedDiffMissingFromShadow` để đón nhận thêm giá trị trả về `destCount` (bỏ qua bằng `_`).

---

### 2. CMS Service (cdc-cms-service)

---

#### [MODIFY] [recon_check.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/recon/recon_check.go)
- Thay thế trường `Tier string` bằng `TypeRecon string json:"type_recon"`.
- Cập nhật logic `Validate()` kiểm tra `TypeRecon` bắt buộc.

#### [MODIFY] [reconciliation_handler_commands.go](file:///Users/trainguyen/Documents/work/cdc-cms-service/internal/api/recon/reconciliation_handler_commands.go)
- API `TriggerCheck` sẽ lấy `type_recon` từ query param (mặc định `"hash_window"`) và map vào `TypeRecon` của command.
- Thay đổi JSON response trả về trường `type_recon` thay vì `tier`.
- Cập nhật `TriggerCheckAll` và `TriggerPrune` để gán `TypeRecon: "hash_window"` (hoặc `"smoke"`) và `TypeRecon: "prune"`.

#### [MODIFY] [recon_read_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go)
- Cập nhật câu SQL trong `listLatestPrimary` để thực hiện `UNION ALL` kết quả từ `cdc_system.cdc_recon_smoke_result` (smoke check) và `cdc_system.cdc_reconciliation_report` (manual check như full_diff/hash_window), rồi lấy ra bản ghi mới nhất của mỗi segment/table sắp xếp theo `checked_at DESC`.
- Cập nhật hàm `GetTableHistory` tương tự để hợp nhất lịch sử đối soát từ cả hai nguồn.

#### [MODIFY] [reconciliation_handler_reports.go](file:///Users/trainguyen/Documents/work/cdc-cms-service/internal/api/recon/reconciliation_handler_reports.go)
- Trong hàm `LatestReport`, nếu `rows[i].CheckType == "full_diff" || rows[i].CheckType == "hash_window"`:
  - Giữ nguyên trạng thái `status` thô từ database (nếu `success` thì chuyển thành `ok`).
  - Tính `DriftPct` dựa trên `MissingCount` chia cho `SourceCount` (hoặc đặt là 0 nếu `MissingCount` bằng 0).
  - Tránh override status bằng kết quả từ `ComputeDriftStatus` để tránh bị lệch trạng thái do so sánh tổng số lượng bản ghi trong window lệch nhau.

---

### 3. CMS Web (cdc-cms-web)

---

#### [MODIFY] [useReconStatus.ts](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useReconStatus.ts)
- Cập nhật `useCheckTableMutation`: nhận tham số `typeRecon: string` (thay vì `tier`) và gọi endpoint `/api/reconciliation/check?type_recon=${typeRecon}`.

#### [MODIFY] [DataIntegrity.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/DataIntegrity.tsx)
- Cập nhật `ModalAction` and `ModalPlan` để đổi `tier: string` thành `typeRecon: string`.
- Trong `openCheckTable`, thiết lập mặc định `typeRecon: 'hash_window'`.
- Truyền `isCheckTier2={modalPlan.action.kind === 'check-table' && (modalPlan.action.typeRecon === 'hash_window' || modalPlan.action.typeRecon === 'full_diff' || modalPlan.action.typeRecon === 'deep_check')}`.

#### [MODIFY] [ConfirmDestructiveModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ConfirmDestructiveModal.tsx)
- Khi người dùng chọn chế độ:
  - Chế độ `"lookback"` -> gửi `typeRecon = 'hash_window'`.
  - Chế độ `"full_diff"` -> gửi `typeRecon = 'full_diff'`.
  - Chế độ `"deep"` -> gửi `typeRecon = 'deep_check'`.
- Truyền `typeRecon` này về cho callback `onConfirm` để gọi mutation API.

#### [MODIFY] [ReconPipelineGrid.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx)
- Trong hàm `buildPipelines`:
  - Nếu `check_type` là `full_diff` hoặc `hash_window`, bỏ qua việc lấy `source_count`/`dest_count` làm `sourceTotal` / `shadowActive` / `masterCount` của toàn bảng (hiển thị `null` / `—`).
  - Thiết lập `driftAB` và `driftBC` trực tiếp dựa trên `diff` / `missing_count` của report (đảo dấu `-a.diff` để hiển thị `-N (thiếu)`).
- Cập nhật `levelLabel` để hiển thị `'Full Diff'` cho check type là `full_diff`.
- Cập nhật render cột "Kết quả" trong bảng "Nhật ký đối soát" để map trạng thái `'success'` thành màu `'green'` và nhãn `'KHỚP'`.

## Verification Plan

### Automated Tests
- Chạy test suite của CDC service:
  ```bash
  go test -v ./internal/service/recon/...
  ```
- Chạy test suite của CMS service:
  ```bash
  go test -v ./test/internal/app/queries/...
  ```

### Manual Verification
- Chạy ứng dụng local, thực hiện click "Bắt đầu đối soát" trên UI.
- Chọn các chế độ Lookback (Hot/Cold), Full Search (Full Diff) và Deep Check, kiểm tra payload gửi lên ở tab Network có chứa `type_recon` thay vì `tier`.
- Xác nhận các tiến trình check chạy đúng và kết quả được lưu trữ chính xác.
