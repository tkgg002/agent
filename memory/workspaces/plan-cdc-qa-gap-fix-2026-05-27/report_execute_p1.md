# Report: Execution of Phase P1 (CDC Reliability & Observability Hardening)

## Overview
Đã hoàn thành Phase P1 trong plan `plan-cdc-qa-gap-fix-2026-05-27`. Phase P1 tập trung vào observability (PPROF, Goleak), event integrity (Ordering & OCC) và E2E Test cho schema drift.

## Các thay đổi chính

### 1. PPROF & Memory Leak Detection (G-7)
- **Files Modified**: 
  - `cmd/worker/main.go`
  - `config/config.go`
  - `config-local.yml`
- **Actions**: Tích hợp pprof listener vào worker để phục vụ profiling động.
- **Leak Detection**: Cấu hình `goleak.VerifyTestMain` ở các package test (`internal/handler`, `internal/service`, `internal/sinkworker`) để phát hiện goroutine leaks.

### 2. Event Ordering Verification (G-8)
- **Files Modified**:
  - `internal/service/schema_adapter_ordering_test.go`
- **Actions**: 
  - Sửa lỗi schema reference (`public.test_users`, `public.t`) trong Unit Test.
  - Sửa lỗi panic của GORM khi `Scan` vào interface `any` bằng cách dùng kiểu `string` tường minh.
  - Sửa cấu hình schema parameter (cần thiết lập column `_source_ts` và `_hash`) để Test OCC Guard hoạt động đúng logic.
  - Fix test `TestEventOrdering_HashTiebreaker` bằng cách gán `srcTsMs = 0` (bắt buộc để kích hoạt logic fallback hash-based dedup).
  - Khắc phục goroutine leak do SQL opener bằng hàm `cleanup()` DB ngay sau khi khởi tạo DB cho các Unit test.

### 3. Schema Drift E2E Test (G-9)
- **Files Modified**:
  - `internal/app/commands/approve_schema_proposal_e2e_test.go`
- **Actions**:
  - Xoá dependency sai (`cdc-cms-service/internal/domain/model`).
  - Viết lại DB schema schema mapping để test trực tiếp bằng SQL Query, mô phỏng đúng structure `cdc_system.schema_proposal`, `cdc_system.shadow_binding`, và `cdc_mapping_rules`.
  - Khởi tạo Postgres và NATS bằng Testcontainers-go.
  - Mọi test E2E cho luồng `ApproveSchemaProposalCommand` pass 100% (Thời gian chạy test ~ 30 giây).

## Kết quả Verification
- **Unit test**: `go test ./internal/service/ -run TestEventOrdering` (PASS)
- **Integration/E2E Test**: `go test ./internal/app/commands/ -run TestApproveSchemaProposal_E2E -v -count=1 -tags=integration` (PASS)
- Các module đã compile thành công, không phát hiện goroutine leak.

## Cập nhật Memory
- `05_progress.md` đã được ghi log các entry thực thi.

## Bước tiếp theo (Next Steps)
- Dựa trên plan, user có thể yêu cầu: `execute p2` (Tier3 batching, k6 load test) hoặc `execute ui` (Admin page migrations và component).
