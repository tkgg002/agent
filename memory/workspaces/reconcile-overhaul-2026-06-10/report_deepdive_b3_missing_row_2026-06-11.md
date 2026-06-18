# report_deepdive_b3_missing_row_2026-06-11.md — Soi sâu row b3 bị từ chối ghi

> Muscle:Claude-Opus-4.8 | 2026-06-11 | Verb: "soi sâu đi"

## Kết luận 1 dòng
Row `_gpay_id=58019585403650053` bị chặn bởi **chuỗi 4 tầng** — và sau khi soi + heal lại qua worker code mới: **ĐÃ VÀO MASTER, b3 = `ok 465/465 missing 0`** (vòng kín đóng).

## Chuỗi nguyên nhân 4 tầng (mỗi tầng có evidence)
| Tầng | Nguyên nhân | Evidence | Trạng thái |
|---|---|---|---|
| 1 | b3 **Sync=Manual** → row vào shadow (02:29Z 10/6) sau lần transmute cuối, không ai sync | Boss tự xác nhận | by-design (bật schedule nếu muốn) |
| 2 | Recon window `upper` **exclusive = shadowMax** → row mới nhất vĩnh viễn ngoài window khi idle → B báo "ok" giả | 11=11 ok dù totals 465/464 | ✅ FIXED +1ms (cả A & B) |
| 3 | Heal chết: `EnsureMaster` tạo UNIQUE INDEX theo `spec.pk="_source_id"` — cột đã bỏ khỏi master → 42703 mọi transmute | log `ensure master destination: create index _source_id does not exist` | ✅ FIXED realCols guard |
| 4 | Transmute 22P02: rule `_id` `data_type=JSONB` + giá trị Mongo `_id` = **string trần** → `invalid input syntax for type json`; kèm `type_errors=1` (= chính `_id` validate fail) | log `master upsert failed _source_id=6a28cbecec5b9378333d594a`; soi raw: `_id`=string, `createdAt/lastUpdatedAt`=number epoch | ✅ resolved trên code transmuter MỚI (bên khác đang refactor "_id business col" + coerce marshal JSONB) — heal lại 14:01 ghi thành công `_id="6a28..."` (jsonb quoted) |

## Verify cuối
- Master b3 có row: `58019585403650053 | _source_ts=1781058540804 | _id="6a28cbecec5b9378333d594a"` ✅
- Re-check Segment B: **`ok` — 465/465, missing 0** ✅ (transmute_lag sẽ về 0 ở vòng đo kế)

## Ghi chú vận hành
- Worker hiện chạy process của bên khác (PID 17863, `go run` code working tree mới nhất — gồm các fix recon của tôi + refactor transmuter của họ). Các binary `/tmp/cdc-worker-recon-p4*` của tôi không còn chạy.
- Đã bật tạm `log_min_error_statement` trên PG dest để bắt statement lỗi → **ĐÃ RESET** về mặc định.
- Theo dõi: `realCols` đang hardcode `_id` (contract "Cách 2" mới coi `_id` là business col) — case `spec.pk=_id` mà không có rule `_id` sẽ pass index nhầm; không sửa thêm giữa lúc bên kia đang refactor cùng file (tránh giẫm chân), đã note.
- 0 file source đổi trong turn soi sâu này (chỉ điều tra + heal + 1 PG setting bật/tắt có ghi nhận).
