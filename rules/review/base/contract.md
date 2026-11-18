# L1 Base Rules — Lens CONTRACT (`B-CON-*`)

> Hợp đồng = mọi thứ có **bên kia** đang đọc: API request/response, event/message payload, DB schema & ownership, CLI flags, config keys, file format. Phải kiểm **cả 2 đầu** (producer ↔ consumer), không chỉ 1 repo.
> Mỗi finding cần `REF:` ở **cả hai phía** nếu truy được; không truy được phía kia → ghi `HYPOTHESIS` + tên repo/service cần kiểm.

## 1. Thay đổi tương thích ngược
- **B-CON-01 (S0)** — Có **xoá/đổi tên/đổi kiểu/đổi ý nghĩa** trường trong payload API/event/DTO mà consumer cũ đang đọc không? Chỉ được **thêm trường optional** (`omitempty`, `?`, nullable).
- **B-CON-02 (S0)** — Có đổi đơn vị/định dạng giá trị (đồng ↔ xu, ms ↔ s, UTC ↔ local, string ↔ number, enum value) giữa các version không?
- **B-CON-03 (S1)** — Enum/status mới được **thêm** có khiến consumer cũ crash hoặc hiểu sai không? Consumer có default an toàn cho giá trị lạ không?
- **B-CON-04 (S1)** — Giá trị mặc định khi trường thiếu có khớp giữa producer và consumer không?

## 2. Event / Message
- **B-CON-05 (S1)** — Subject/topic/routing key mới có khớp quy ước đặt tên và **đã có consumer** đăng ký không? Consumer mới có subject nào producer chưa phát không?
- **B-CON-06 (S1)** — Event có đủ trường để consumer **idempotent** (id, version, timestamp, source) và để đối soát không?
- **B-CON-07 (S1)** — Ordering/partition key có thay đổi không? Thay đổi có phá giả định thứ tự của consumer không?
- **B-CON-08 (S2)** — Schema registry/contract test/fixture mẫu có được cập nhật cùng diff không?

## 3. Database ownership & schema
- **B-CON-09 (S0)** — Service này có **ghi vào bảng do service khác sở hữu** (trái quy ước write-owner trong profile) không?
- **B-CON-10 (S1)** — Cột mới có nullable/default để binary cũ vẫn insert được không? Constraint mới có fail trên dữ liệu đang có không?
- **B-CON-11 (S1)** — Model/struct ORM có khớp 100% với migration (tên cột, kiểu, nullable, index) không? Có cột trong struct mà migration chưa có (hoặc ngược lại)?
- **B-CON-12 (S2)** — Query mới có dùng cột/bảng tồn tại ở **mọi môi trường** (không chỉ local) không?

## 4. HTTP API
- **B-CON-13 (S1)** — Route mới có đúng prefix/version/method và không trùng/đè route cũ không? Route cũ có bị đổi path/method không?
- **B-CON-14 (S1)** — Status code & error envelope có khớp chuẩn hiện hữu của repo không (client đang parse theo format nào)?
- **B-CON-15 (S2)** — OpenAPI/spec/types phía client (`types/*.ts`, SDK) có được cập nhật cùng diff không?

## 5. Config / CLI / File
- **B-CON-16 (S1)** — Config key/env var/flag mới có default an toàn khi thiếu không? Key cũ bị đổi tên có alias/migration không?
- **B-CON-17 (S2)** — Format file xuất/nhập (CSV header, JSON lines, report) đổi có bên nào đang parse không?

## 6. Liên repo (bắt buộc khi profile khai báo nhiều repo)
- **B-CON-18 (S1)** — Với mỗi contract bị chạm, đã mở **repo phía kia** và trích `REF:` chứng minh khớp chưa? Chưa mở → ghi rõ `HYPOTHESIS: cần kiểm <repo>/<path>`.

## Lệnh hỗ trợ (0 LLM token)
```bash
# trường JSON bị xoá/đổi tên trong struct Go
git diff <base>..<head> -U0 | grep -nE '^-.*json:"[^"]+"' 
# subject/topic literal mới xuất hiện
git diff <base>..<head> -U0 | grep -noE '^\+.*"[a-z]+\.[a-z.-]+\.[a-z.>*-]+"' | sort -u
# cột trong migration vs struct
git diff <base>..<head> --name-only | grep -Ei 'migrat|schema|\.sql$'
```
