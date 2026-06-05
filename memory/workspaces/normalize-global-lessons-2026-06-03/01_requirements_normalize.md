# 01_requirements — Normalize Global Lessons

## Functional Requirements
1. **Thống kê (Statistics)**: Dashboard tổng quan — tổng số lesson, phân bố theo tháng, mức tuân thủ format, top tag, tag sprawl.
2. **Tổng hợp (Synthesis)**: Mỗi lesson thô → 1 Global Pattern chuẩn (giữ toàn bộ, KHÔNG merge).
3. **Sắp xếp (Organization)**: Phân toàn bộ lesson vào taxonomy 8 nhóm global; trong mỗi nhóm sort theo ngày giảm dần.
4. **Chuẩn hoá (Standardization)**: Mỗi pattern viết đúng Rule 13: `Global Pattern [A does B to X] → Result Y. Đúng: [correct flow]` + tags chuẩn hoá + trích nguồn ngày/dòng.

## Output Requirements
- File mới: `agent/memory/global/lessons_global_normalized.md` (toàn bộ bản chuẩn hoá + dashboard + taxonomy + index).
- APPEND vào `lessons.md`: một block index ngắn trỏ tới file mới (append-only, hợp lệ Rule 11).

## Non-Functional / Constraints
- KHÔNG sửa/xoá nội dung cũ của `lessons.md` (chỉ append).
- Không nạp toàn bộ 530KB vào 1 context → dùng sub-agent đọc song song theo chunk, trả summary nhỏ, ghi part-file.
- Ngôn ngữ: tiếng Việt.

## Definition of Done
- [ ] `lessons_global_normalized.md` tồn tại, có đủ 4 phần (Dashboard, Taxonomy, Body chuẩn hoá, Index).
- [ ] Số lesson trong bản chuẩn ≈ số lesson nguồn (sai số khai báo rõ).
- [ ] 100% pattern theo format Rule 13.
- [ ] `lessons.md` được append index, nội dung cũ nguyên vẹn (verify bằng line-count trước/sau).
- [ ] Workspace có đủ doc set + 05_progress append-only.
