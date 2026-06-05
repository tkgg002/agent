# 09_tasks_solution — Normalize Global Lessons

## Giải pháp đã giao
- **File phái sinh**: `agent/memory/global/lessons_global_normalized.md` (229 Global Pattern, 8 nhóm, Rule 13).
- **Index**: APPEND vào cuối `lessons.md` (append-only) trỏ tới file phái sinh.
- **Tooling tái lập**: `tooling_assemble_lessons.py` (trong workspace này).

## Cách re-generate khi lessons.md tăng
1. Phân tích lại separator `---`, chia ~9 chunk cân bằng tại ranh giới `---`.
2. Dispatch sub-agent rewrite từng chunk → `/tmp/norm_part_NN.md` (format @@-block, taxonomy + Rule 13).
3. `python3 tooling_assemble_lessons.py` → sinh `lessons_global_normalized.md` mới.
4. **LUÔN** kiểm `wc -l lessons.md` trước/sau toàn bộ quá trình; nếu tăng → có lesson mới nằm ngoài chunk → bổ sung part rồi assembly lại.

## Lesson rút ra (Global Pattern — đề xuất append vào lessons.md nếu User đồng ý)
- **Global Pattern**: `[Agent A xử lý file append-only lớn X bằng cách chụp snapshot ranh giới (offset/size) rồi fan-out đọc song song]` → `[nếu X bị APPEND thêm TRONG lúc xử lý, phần delta nằm ngoài mọi snapshot-range → bị bỏ sót thầm lặng]`. **Đúng**: chụp `size/line-count` của X TRƯỚC và SAU pha xử lý; nếu lệch → tính delta `(EOF_cũ, EOF_mới]` và xử lý bổ sung; verify tổng coverage = tổng entity hiện tại trước khi báo done.
- **Phạm vi (≥3 dự án?)**: Có — log processing, file ETL chunked, incremental indexer, bất kỳ pipeline đọc nguồn mutable theo offset.
- **Tags**: #snapshot-during-mutation #chunked-processing #coverage-verification #append-only #root-cause
