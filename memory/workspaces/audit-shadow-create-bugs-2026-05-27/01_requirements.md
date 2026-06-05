# 01_requirements — Audit Shadow Create Bugs

## Functional (audit-only deliverables)

| ID | Item |
|---|---|
| R-1 | Trace luồng tạo shadow từ FE `http://localhost:5173/shadow` → endpoint BE → handler worker → CREATE TABLE PG. Liệt kê file/line từng layer. |
| R-2 | Xác định cơ chế B1 (auto-bind field từ entity cũ): có phải FE prefill từ list shadow gần tên? hay BE backfill từ shadow_binding cùng `source_object_id`? hay là logic clone schema theo tên table normalized? |
| R-3 | Xác định B2: build column list ở đâu, thiếu `_source_ts` ở đâu (CREATE TABLE DDL hoặc default column registry). |
| R-4 | Cross-check shadow đã tồn tại (`shadow_users`, `shadow_test_*`) trong DB local có `_source_ts` không → confirm bug regression hay là từ đầu. |
| R-5 | Đề xuất giải pháp elegant tuân thủ §6 GEMINI (root cause fix, không workaround). |

## Non-functional
- **Không sửa code** trong phase này. Audit-only → ghi vào `09_tasks_solution_*.md` chờ user approve (per §12).
- **Không cheat DB**: cấm `ALTER TABLE shadow ADD COLUMN _source_ts` thủ công. Phải fix tại core flow tạo shadow.
- **Memory file APPEND-only** §11: `05_progress.md` chỉ append.
- **Pre-flight §14**: tất cả doc tạo thành file vật lý trước khi kết thúc câu trả lời.

## Out of scope
- Sửa các shadow đã bị tạo lỗi (data migration). Phase tiếp theo, sau khi fix core.
- Audit FE security XSS/CSRF.
- Audit performance.

## Definition of Done
- File audit này + 03_implementation + 09_tasks_solution + 10_gap_analysis tạo vật lý.
- Build verify cả 3 service: `cdc-cms-service`, `centralized-data-service`, `cdc-cms-web` (chỉ build, không deploy).
- Report cuối cùng có bảng "files changed + line count" — nếu phase này KHÔNG sửa code thì cột thay đổi = 0, ghi rõ.
