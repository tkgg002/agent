# L1 Base Rules — Lens DESIGN & ARCHITECTURE (`B-DES-*`)

> Pattern cụ thể của từng repo nằm ở L3 Profile (`review_profile.md`). L1 chỉ kiểm nguyên tắc chung. Mỗi finding cần `REF:`.
> Nguyên tắc nền: **Open-Closed, Backward Compatible, Single Responsibility, Simplicity First, Minimal Impact.**

## 1. Đúng chỗ, đúng tầng
- **B-DES-01 (S2)** — Code mới đặt đúng **layer/functional group** theo profile không? (Ví dụ: logic nghiệp vụ trong handler/controller; SQL trong handler; gọi hạ tầng trực tiếp từ domain.)
- **B-DES-02 (S2)** — Có import ngược chiều phụ thuộc (tầng trong import tầng ngoài, domain import infra, shared pkg import internal) không? `scope.py` xuất `layer_violations` — xác nhận từng cái.
- **B-DES-03 (S2)** — Có **file mẫu canonical** đang chạy tốt cho luồng tương tự không? Diff có đi theo khuôn đó hay "sáng tác pattern mới"?

## 2. Mở rộng không phá cũ (Open-Closed)
- **B-DES-04 (S0)** — Có đổi chữ ký hàm public, đổi tên/kiểu trường struct/JSON/ORM, đổi hành vi mặc định mà code cũ đang gọi không? Phải mở rộng qua hàm mới / option `With*` / builder / tham số optional.
- **B-DES-05 (S1)** — Thêm loại mới (engine, partner, provider, strategy, type) bằng cách **thêm `if/else`/`switch case` vào bộ điều phối lõi** thay vì thêm 1 implementation mới + đăng ký (Registry/Strategy/Adapter)?
- **B-DES-06 (S1)** — Diff có nhồi thêm vào **god-file** (vượt ngưỡng LOC trong profile, mặc định 600) thay vì tách file đơn trách nhiệm không? `scope.py` xuất `god_files`.
- **B-DES-07 (S2)** — Có duplicate logic đã tồn tại ở nơi khác (copy-paste thay vì tái dùng) không? Có tái dùng nhưng **thiếu field/bước** so với bản gốc không?

## 3. Trách nhiệm & ranh giới
- **B-DES-08 (S2)** — Hàm/struct mới có >1 lý do để thay đổi (vừa parse, vừa validate, vừa ghi DB, vừa publish) không?
- **B-DES-09 (S1)** — Có gắn mock/closure/fake (`*Fn`, `testHook`) vào struct production để tiện test không? Phải dùng interface + DI.
- **B-DES-10 (S2)** — Dependency mới có đi qua constructor/DI/wiring tập trung không, hay khởi tạo rải rác (global, singleton ẩn, `init()` có side-effect nặng)?
- **B-DES-11 (S2)** — Có wiring bị **bỏ sót** (thêm interface/driver nhưng chưa nối vào điểm khởi tạo runtime) không? Kiểm tra file wiring/server setup.

## 4. Đơn giản & tác động tối thiểu
- **B-DES-12 (S2)** — Diff có sửa/di chuyển/đổi tên nhiều hơn phạm vi yêu cầu (refactor kèm, format lại file) khiến blast-radius tăng không?
- **B-DES-13 (S2)** — Có abstraction thừa (interface 1 implementation, generic không cần, config flag không ai dùng) không?
- **B-DES-14 (S3)** — Naming khớp quy ước repo (số ít/số nhiều, prefix, casing, tên file ↔ tên type) không?
- **B-DES-15 (S2)** — Có để lại dead code, nhánh legacy không còn gọi, TODO không có owner/ticket không?

## 5. Dữ liệu & schema
- **B-DES-16 (S1)** — Migration có **tương thích ngược** với binary đang chạy (expand → migrate → contract) không? `DROP/RENAME` cột đang dùng?
- **B-DES-17 (S2)** — Index/constraint mới có khớp pattern query thực tế không? Có tạo index trên bảng lớn mà không `CONCURRENTLY`/online không?

## Lệnh hỗ trợ (0 LLM token)
```bash
# file có thêm dòng và LOC sau diff > ngưỡng
git diff <base>..<head> --numstat | awk '$1>0{print $3}' | while read f; do n=$(git show <head>:"$f" 2>/dev/null | wc -l); [ "$n" -gt 600 ] && echo "$n $f"; done
# switch/if theo type mới trong file bị cấm (đọc danh sách từ profile)
git diff <base>..<head> -U0 -- <forbidden_files> | grep -nE '^\+\s*(if|else if|case)\b.*(==|strings\.(EqualFold|HasPrefix)|switch)'
```
