# Plan — Phase 12 Reconciliation Scope V2

## Kế hoạch thực hiện

1. Audit `reconciliation_handler.go`, `useReconStatus.ts`, `DataIntegrity.tsx`, routes và models liên quan.
2. Xác định endpoint nào có thể enrich contract mà không tạo thêm surface API thừa.
3. Refactor backend reconciliation:
   - enrich report
   - enrich failed logs
   - cho `check`/`heal` nhận scope theo source/shadow metadata
4. Refactor FE `DataIntegrity` và `useReconStatus` để dùng contract mới.
5. Update swagger/comment cho reconciliation APIs.
6. Chạy verify backend test + frontend build.
7. Ghi đầy đủ artifact phase vào workspace và append log/trạng thái/gap analysis.
