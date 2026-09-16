# Phân tích Kỹ thuật chi tiết - FixDataIntegrityFilterActivePipelines20260826

## 1. Phân tích Nguyên nhân
- Khi người dùng tạo hoặc thử nghiệm nhiều pipeline trong quá khứ, các bản ghi kiểm tra smoke check của các pipeline này ghi nhận vào bảng `cdc_system.cdc_recon_smoke_result`.
- Backend `/api/reconciliation/report` quét lấy tất cả kết quả recon smoke trong vòng 24h qua.
- Ở Frontend (`ReconPipelineGrid.tsx`), logic lọc pipeline active (`isShadowOff`, `isMasterSyncOff`, `activePipelines`) đã bị comment out. Do đó, tất cả 29 bản ghi / 15 pipelines cũ đều bị đẩy ra UI Grid.
- Tương tự ở `DataIntegrity.tsx`, thẻ Thống kê Header tính tổng `pipelines.length` trên mảng chưa lọc.

## 2. Giải pháp Khắc phục
- Bật lại và làm chuẩn xác logic matching FQN của `isShadowOff` và `isMasterSyncOff`.
- Đảm bảo tính nhất quán giữa Header Stats và Grid bằng cách chỉ đếm/hiển thị các Active Pipelines.
