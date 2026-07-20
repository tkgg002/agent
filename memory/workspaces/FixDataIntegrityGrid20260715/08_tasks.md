# 08 — Tasks: FixDataIntegrityGrid20260715

## Bug 1: `— — 2,718,739` (source/shadow hiển thị dashes)

- [x] Xác định root cause: DISTINCT ON bị split do shadow_schema NULL vs non-null
- [x] Fix SQL: Thêm LATERAL JOIN sb_norm, COALESCE shadow_schema trước DISTINCT ON
- [x] Fix FE: Sửa buildPipelines dedup ưu tiên row có active counts
- [ ] **VERIFY thực tế**: Restart server với binary mới → curl API → xác nhận API trả 2 rows, source_active != null
- [ ] **VERIFY FE**: Mở localhost:5173/data-integrity → xác nhận grid không còn `— —`

## Bug 2: Pipelines xoá connector vẫn hiện

- [ ] **INVESTIGATE**: Hỏi anh cụ thể "xoá connector" là xoá ở đâu (Kafka Connect? source_object_registry? shadow_binding?)
- [ ] Kiểm tra `listLatestPrimary` query có filter connector status không
- [ ] Kiểm tra `buildPipelines` FE có filter null connector không
- [ ] Đề xuất fix và xin approve
- [ ] Implement fix
- [ ] Verify

## Tab Smoke/Recon (đã làm nhưng chưa verify)

- [x] Thêm Tabs component vào antd imports
- [x] Thêm hook historyRecon (excludeSmoke=true)
- [x] Tạo HistoryTable inline component
- [x] TypeScript check pass
- [ ] **VERIFY FE**: Kiểm tra drawer hiển thị 2 tabs đúng
