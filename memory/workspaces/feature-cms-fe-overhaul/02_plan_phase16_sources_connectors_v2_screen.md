# Plan — Phase 16 Sources Connectors V2 Screen

## Kế hoạch thực hiện

1. Audit `SourceConnectors.tsx`, `sources_handler.go`, `system_connectors_handler.go`.
2. Xác nhận API hiện có đủ dữ liệu cho màn dual-view.
3. Refactor FE page:
   - fetch connectors
   - fetch source fingerprints
   - render summary + mismatch alerts
   - tách tab `Connectors` và `Source Fingerprints`
4. Build FE để verify.
5. Ghi phase vào workspace cùng note debt backend còn lại.
