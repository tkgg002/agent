# 02_plan — Inventory + Dead-code

## Output sẽ giao
1. `11_current_structure.md` — cây thư mục + bảng chức năng từng file (213 file) + cột STATUS.
2. `12_unused_files.md` — danh sách file CHƯA DÙNG (dead package + dead file + partial-dead funcs), kèm bằng chứng (reachability / deadcode / grep).
3. `13_proposed_structure.md` — đề xuất sắp xếp lại theo 8 Bounded Context (kế thừa v2), map file→vị trí mới + bucket "DEAD → xoá/cô lập".

## Cách làm
- Fan-out 7 sub-agent (Rule 20) theo vùng: api · commands(+ports) · queries · persistence · infra(http/messaging/observability/cache) · domain+model+bootstrap+middleware+router+server+migrate+naming · pkgs+cmd+config+docs.
- Mỗi agent: đọc file → ghi bảng `| file | LOC | chức năng | symbol | STATUS |` vào `/tmp/inv_<area>.md`; trả về compact danh sách DEAD?/PARTIAL.
- Assembly part-files → `11_current_structure.md`. Tổng hợp dead-list → `12`. Tự map reorg → `13`.

## Ràng buộc
- KHÔNG sửa 1 dòng .go (chỉ phân tích + đề xuất). Đề xuất xoá file = chờ user duyệt.
- STATUS phán đoán dựa grep/deadcode thật, không đoán; chú ý handler wired qua router/bus (tránh false-positive).
