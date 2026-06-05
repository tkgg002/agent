# 03_implementation — Normalize Global Lessons

## Phương pháp (parallel chunk + script assembly)
1. **Phân tích tĩnh** bằng shell (không nạp 530KB vào context): wc, grep markers, tag freq, separator positions.
2. **Chia 9 chunk** cân bằng (~560 dòng), ranh giới đặt tại separator `---` để không vỡ lesson.
3. **9 sub-agent (general-purpose) song song**: mỗi agent `Read(offset,limit)` chunk của mình → viết lại TỪNG lesson sang block canonical (Rule 13) có marker máy-đọc:
   `@@LESSON@@ / @@CAT=<key>@@ / @@DATE=<YYYY-MM-DD>@@ / ### […] / bullets / @@END@@`.
   Ghi vào `/tmp/norm_part_NN.md`; chỉ trả summary nhỏ (count + tally + anomalies) → giữ context chính sạch.
4. **Assembly** `tooling_assemble_lessons.py`: parse block, group theo taxonomy (8 nhóm, thứ tự cố định), sort ngày giảm dần (placeholder 0000-00-00 xuống cuối), strip marker, sinh Dashboard + Mục lục + Body + Footer.

## Sự kiện MID-SESSION (quan trọng)
- Mạch việc kéo dài qua nửa đêm (06-03→06-04). `lessons.md` bị **user APPEND thêm** trong lúc sub-agent chạy: 5061 → 5109 dòng (+48).
- 3 lesson mới (Master-UI, Regex-DDL, VCS-granularity) nằm **ngoài** chunk 9 (≤5061) → ban đầu bị bỏ sót.
- Phát hiện qua so khớp `wc -l` trước/sau; gap-analysis bằng token độc nhất; chuẩn hoá bổ sung `part_10` → tổng 229.

## Output
- `agent/memory/global/lessons_global_normalized.md` — 229 pattern, 2190 dòng, 100% canonical Rule 13.
- `agent/memory/global/lessons.md` — APPEND block index (append-only); số liệu index cập nhật 226→229 trên chính block phái sinh.

## Verification
- Marker LESSON=CAT=END cân bằng mọi part; 0 malformed.
- `### ` = 231 (229 lesson + 2 sub-header dashboard); Global Pattern bullets = 229; Nguồn refs = 229; `@@` sót = 0.
- **Rule 11**: `head -5109 lessons.md | md5 = cdbbc29722f23683f6b707b3991a8033` KHÔNG đổi trước/sau → 5109 dòng lesson gốc nguyên vẹn byte-for-byte.
- Security scan: 0 raw password / 0 live connstring / 0 API key / 0 private IP.
