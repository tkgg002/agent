# Yêu cầu: Audit listLatestPrimary / listLatestLegacy

## 1. Mục tiêu
Audit toàn bộ luồng dữ liệu từ SQL query `listLatestPrimary` / `listLatestLegacy` → Go LatestReportRow → API handler enrichment → FE `ReconPipelineGrid` + `DataIntegrity` để xác định:
- Các cột/JOIN nào đang **thực sự** được sử dụng trên UI trang `/data-integrity`.
- Các cột/JOIN nào **dư thừa** có thể loại bỏ để giảm latency SQL.

## 2. DoD
- [ ] Liệt kê tất cả các fields từ SQL → Go → FE, đánh dấu rõ "USED" / "UNUSED".
- [ ] Chỉ ra các LATERAL JOINs đắt đỏ có thể loại bỏ hoặc thay thế.
- [ ] Đề xuất phương án tối ưu cụ thể.
