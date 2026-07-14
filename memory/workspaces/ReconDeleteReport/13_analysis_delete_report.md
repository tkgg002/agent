# Phân tích Kỹ thuật & Rà soát Bảo mật (Adversarial Security Review) - ReconDeleteReport

Tài liệu ghi lại kết quả tự đánh giá bảo mật của Muscle (Chief Engineer) đối với các thay đổi mã nguồn liên quan đến chức năng xoá phiên đối soát.

## 1. Rà soát Lỗ hổng SQL Injection
* **Vị trí kiểm tra**: `internal/api/recon/reconciliation_handler_delete_report.go`
* **Mã nguồn**:
  ```go
  h.db.WithContext(c.UserContext()).Exec("DELETE FROM cdc_system.cdc_reconciliation_report WHERE id = ?", id)
  ```
* **Đánh giá**:
  - Biến `id` được trích xuất từ route param và ép kiểu an sau sang `uint64` thông qua `strconv.ParseUint`.
  - Truy vấn sử dụng cơ chế placeholder `?` của GORM, thực hiện Parameterized Query. Dữ liệu đầu vào không bao giờ được nối chuỗi trực tiếp vào SQL syntax.
  - **Kết luận**: An toàn 100% trước lỗi SQL Injection.

## 2. Kiểm soát Truy cập & Phân quyền (Access Control / IDOR)
* **Vị trí kiểm tra**: `internal/router/router.go`
* **Mã nguồn**:
  ```go
  api.Delete("/reconciliation/report/:id", append(destructiveChain, h.Recon.DeleteReport)...)
  api.Delete("/v1/reconciliation/report/:id", append(destructiveChain, h.Recon.DeleteReport)...)
  ```
* **Đánh giá**:
  - Route được đăng ký dưới group `api` sử dụng middleware `JWTAuth` để xác thực JWT Token của user.
  - Endpoint sử dụng `destructiveChain` bao gồm middleware `RequireOpsAdmin()`. Điều này giới hạn quyền thực thi chỉ dành riêng cho tài khoản có vai trò `OpsAdmin`.
  - **Kết luận**: Ngăn chặn hoàn toàn truy cập trái phép. User thông thường hoặc khách truy cập không thể xoá báo cáo.

## 3. Ghi vết Kiểm toán (Audit Logging & Idempotency)
* **Vị trí kiểm tra**: `internal/router/router.go` & `cdc-cms-web/src/components/ExecuteHealModal.tsx`
* **Đánh giá**:
  - `destructiveChain` tích hợp middleware `destructive.Audit` và `destructive.Idempotency`.
  - Ở phía Frontend, khi gọi API xoá, client bắt buộc phải truyền header `X-Action-Reason` (lý do thực hiện, tối thiểu 10 ký tự) và `Idempotency-Key` (chống gửi trùng yêu cầu).
  - Backend kiểm tra và ghi nhận lý do này vào bảng nhật ký kiểm toán hệ thống.
  - **Kết luận**: Đạt chuẩn Audit và Idempotency cho các thao tác phá huỷ (destructive operations).

## 4. Kiểm thử Tích hợp và Build
* Backend đã biên dịch thành công thông qua `go build ./cmd/server/...`.
* Frontend đã kiểm tra kiểu dữ liệu thành công qua `npx tsc --noEmit`.
