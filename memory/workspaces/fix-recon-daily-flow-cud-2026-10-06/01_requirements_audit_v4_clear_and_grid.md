# 01 — Requirements: Audit V4 Clear Activity Log & Bảng Đối Soát

1. **R1 Cách ly pipeline thật**: 2 connector cùng tên bảng (khác `shadow_schema`) phải xem và xóa log độc lập. Chứng minh trên Postgres thật.
2. **R2 Clear an toàn**: bắt buộc scope `shadow_schema + shadow_table`; không cho xóa theo ngày trên toàn hệ thống; ngày là mốc tuyệt đối (RFC3339); đi qua `destructiveChain` (OpsAdmin + Idempotency + Audit).
3. **R3 Tách Snapshot vs CDC ở cả Shadow lẫn Master** dựa trên dấu do writer ghi, không suy diễn ở UI.
4. **R4 Số liệu theo ngày đúng**: bucket theo giờ local; không bỏ sót `transmute_sub`; (P2) tổng hợp phía server, không giới hạn 200 log.
5. **R5 Minimal impact**: thay đổi additive, log cũ giữ nguyên hành vi; không đổi schema DB.
