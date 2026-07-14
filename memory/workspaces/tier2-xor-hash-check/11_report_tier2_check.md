# Báo Cáo Thay Đổi Tài Liệu & Mã Nguồn (Document & Source Code Changes Report)

Báo cáo này liệt kê chi tiết các tệp tin tài liệu và mã nguồn Go đã được tạo mới hoặc cập nhật trong phiên làm việc đối soát Tier 2 nhằm tuân thủ Hiến pháp hệ thống.

---

## Danh Sách Các Tệp Tin Tài Liệu Trong Workspace

| Trạng thái | Tên tệp tin | Số dòng | Mô tả thay đổi / Nội dung |
|------------|-------------|---------|---------------------------|
| **[MỚI]** | [00_context.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/tier2-xor-hash-check/00_context.md) | ~15 | Khai báo bối cảnh workspace, xác định codebase chuẩn. |
| **[MỚI]** | [01_requirements_tier2_check.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/tier2-xor-hash-check/01_requirements_tier2_check.md) | ~13 | Đặc tả các yêu cầu phân tích kỹ thuật của luồng Tier 2. |
| **[MỚI]** | [02_plan.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/tier2-xor-hash-check/02_plan.md) | ~48 | Kế hoạch thực hiện đối soát song ngữ Anh/Việt. |
| **[MỚI]** | [05_progress_tier2_check.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/tier2-xor-hash-check/05_progress_tier2_check.md) | ~25 | Nhật ký tiến độ ghi nhận từng bước thao tác của AI. |
| **[MỚI]** | [08_tasks_tier2_check.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/tier2-xor-hash-check/08_tasks_tier2_check.md) | ~9 | Danh sách task/checklist hoàn thành. |
| **[MỚI]** | [09_tasks_solution_tier2_check.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/tier2-xor-hash-check/09_tasks_solution_tier2_check.md) | ~400 | Hồ sơ giải pháp kỹ thuật chi tiết chứa React Mock và các mã nguồn diffs chi tiết của Go. |
| **[MỚI]** | [11_report_tier2_check.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/tier2-xor-hash-check/11_report_tier2_check.md) | ~30 | Báo cáo chi tiết các thay đổi tài liệu (file này). |
| **[MỚI]** | [12_implementation_plan_tier2_check.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/tier2-xor-hash-check/12_implementation_plan_tier2_check.md) | ~23 | Kế hoạch triển khai chi tiết các turn của AI trong phiên. |
| **[MỚI]** | [13_analysis_tier2_check.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/tier2-xor-hash-check/13_analysis_tier2_check.md) | ~500 | Phân tích chuyên sâu thuật toán XOR-hash, control flow, read-only, timezone skew, DDL core mapping và co-design FE/BE. |
| **[MỚI]** | [14_walkthrough_tier2_check.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/tier2-xor-hash-check/14_walkthrough_tier2_check.md) | ~70 | Tài liệu walkthrough tổng kết kết quả kiểm thử và nghiệm thu. |

---

## Danh Sách Các Tệp Tin Mã Nguồn Go Đã Thay Đổi

| Trạng thái | Tên tệp tin | Mô tả thay đổi | Số dòng thay đổi |
|------------|-------------|----------------|------------------|
| **[CẬP NHẬT]** | `internal/handler/recon/recon_handler_run.go` | Thêm tham số payload `mode`, `start_time`, `end_time` và truyền vào `healSegmentA`. | +8 lines |
| **[CẬP NHẬT]** | `internal/service/recon/recon_stream.go` | Thêm hàm stream range-based `StreamIDsInTimeRange` cho MongoDB/PostgreSQL sources. | +170 lines |
| **[CẬP NHẬT]** | `internal/service/recon/recon_tier_a.go` | Thêm hàm `TimeBoundedDiffMissingFromShadow` thực hiện quét IDs bị lệch theo range an toàn. | +50 lines |
| **[CẬP NHẬT]** | `internal/handler/recon/recon_heal_v4.go` | Redesign signature `healSegmentA`, validate thời gian giới hạn 30 ngày cho Full-diff, định tuyến direct write `FetchAndWriteByIDs` cho Window mode. | +120 lines |
| **[CẬP NHẬT]** | `internal/handler/recon/recon_heal_v4_test.go` | Bổ sung unit test case `TestHealSegmentA_FullDiffMode_InvalidTimeRange` để bảo vệ range validation. | +40 lines |

## Cập Nhật Phiên Làm Việc (Muscle Code Execution)

| Trạng thái | Tên tệp tin | Mô tả thay đổi | Số dòng thay đổi |
|------------|-------------|----------------|------------------|
| **[CẬP NHẬT]** | `src/hooks/useReconStatus.ts` | Thêm tham số `lookback` vào `useHealMutation` mutation payload và POST request body. | +4 lines |
| **[CẬP NHẬT]** | `src/components/ConfirmDestructiveModal.tsx` | Ẩn time input khi ở Window mode, thêm radio buttons chọn lookback mode (Hot/Cold), cập nhật kiểu onConfirm và logic gửi params. | +50 lines |
| **[CẬP NHẬT]** | `src/pages/DataIntegrity.tsx` | Nhận và chuyển tiếp lookback param từ Modal confirm đến `heal.mutateAsync`. | +3 lines |
| **[CẬP NHẬT]** | `internal/handler/recon/recon_handler_run.go` | Unmarshal thêm parameter `lookback` từ NATS message payload, truyền vào `healSegmentA` và đảm bảo `cold_lookback` chỉ set cho segment B. | +10 lines |
| **[CẬP NHẬT]** | `internal/handler/recon/recon_heal_v4.go` | Nhận param `lookback` trong `healSegmentA` và điều kiện hóa việc khởi tạo context (Hot mode dùng context gốc, Cold mode dùng context chứa `cold_lookback=true`). | +15 lines |
| **[XÁC MINH]** | `internal/service/recon/recon_stream.go` | Kiểm tra và xác nhận hàm `StreamIDsInTimeRange` MongoDB filter đã hỗ trợ đầy đủ cả kiểu Date và Epoch Ms. | 0 lines |

## Cập Nhật Phiên Làm Việc (Muscle Code Execution - Hỗ trợ lookback Tier 2)

| Trạng thái | Tên tệp tin | Mô tả thay đổi | Số dòng thay đổi |
|------------|-------------|----------------|------------------|
| **[CẬP NHẬT]** | `src/hooks/useReconStatus.ts` | Mở rộng `useCheckTableMutation` payload struct và body POST request thêm `lookback`. | +5 lines |
| **[CẬP NHẬT]** | `src/components/ConfirmDestructiveModal.tsx` | Nhận thêm `isCheckTier2` prop. Nếu `true`, chỉ hiển thị radio buttons để chọn lookback (Hot/Cold) và ẩn hoàn toàn mode Window/Full-diff hay inputs date. Hàm `handleOk` gọi `onConfirm` truyền `lookback`. | +40 lines |
| **[CẬP NHẬT]** | `src/pages/DataIntegrity.tsx` | Cập nhật `openCheckTable` để truyền `isCheckTier2: String(record.tier) === "2"` vào modal action. Cập nhật `handleConfirm` chuyển tiếp `lookback` sang `checkTable.mutateAsync`. Pass prop `isCheckTier2` vào `ConfirmDestructiveModal` dựa trên modal action. | +10 lines |
| **[CẬP NHẬT]** | `internal/handler/recon/recon_handler_run.go` | Mở rộng NATS unmarshal struct của `HandleReconCheck` nhận thêm `lookback`. Ở switch case `payload.Tier == "2"`: Nếu `payload.Lookback == "cold"`, wrap context `"cold_lookback" = true` truyền vào `RunTier2`. | +10 lines |

