# L2 Domain Pack — FINTECH (`F-*`)

> Nạp khi `.agent-project.json` → `review.packs` chứa `"fintech"`. Bổ sung lên L1, chỉ **nâng** severity. Mỗi mục là câu hỏi có/không; finding phải có `REF: <path>:<start>-<end>` + trích code.
> **Evidence bắt buộc cho mọi finding `S0` trong pack này:** trích đúng dòng tính toán/ghi tiền hoặc dòng log/response chứa dữ liệu nhạy cảm; không chấp nhận "có thể".

## 0. Money-path keywords (mặc định; manifest `review.money_path_keywords` ghi đè)
```
amount, balance, ledger, payment, payout, refund, fee, charge, settle, settlement, wallet, txn, transaction, transfer, debit, credit, currency, fx, rate, invoice, limit, quota, otp, pan, card, cvv, kyc
```
Hunk chứa ≥1 keyword → `money_path=true` → lens Logic & Security **phải** đọc cả hàm bao quanh, không chỉ hunk.

## 1. Lens SECURITY (`F-SEC-*`)
- **F-SEC-01 (S0)** — PAN/CVV/số tài khoản/CCCD/OTP có xuất hiện ở log, response, message queue, DB không mã hoá, test fixture không? Chỉ chấp nhận mask (`****1234`) hoặc token hoá.
- **F-SEC-02 (S0)** — Webhook/callback từ đối tác (PSP, ngân hàng, ví) có verify chữ ký + timestamp + chống replay (nonce / id đã xử lý) không?
- **F-SEC-03 (S1)** — Endpoint chuyển tiền / đổi hạn mức / đổi thông tin thụ hưởng có xác thực bước 2 (OTP/PIN/MFA) và kiểm tra tenant/owner không?
- **F-SEC-04 (S1)** — Secret đối tác (merchant key, HMAC, cert) có nạp từ secret store/env, không commit, có rotate được không?
- **F-SEC-05 (S1)** — Audit trail cho hành động tiền (ai, khi nào, trước/sau) có được ghi append-only không? Diff có xoá/sửa log audit cũ không?

## 2. Lens LOGIC (`F-LOG-*`)
- **F-LOG-01 (S0)** — Tiền có dùng số thực (`float`/`double`/`number` JS) ở bất kỳ bước nào không? Bắt buộc integer minor-unit hoặc decimal cố định; làm tròn phải khai báo rõ chiều (half-up/banker) và chỉ 1 lần ở biên.
- **F-LOG-02 (S0)** — Thao tác tiền có **idempotency key** (request id / txn id) và kiểm tra trùng trước khi ghi không? Retry/consumer redelivery có thể ghi 2 lần không?
- **F-LOG-03 (S0)** — Debit và credit (hoặc trừ ví + ghi ledger) có nằm trong **cùng 1 transaction/atomic op** không? Có nhánh lỗi nào để lại trạng thái lệch (ví trừ nhưng ledger không ghi) không?
- **F-LOG-04 (S0)** — Số dư có kiểm tra đủ **trong cùng câu lệnh ghi** (`UPDATE ... WHERE balance >= ?` / OCC version) thay vì read-then-write không?
- **F-LOG-05 (S1)** — Trạng thái giao dịch (pending → success/failed/reversed) có máy trạng thái rõ, cấm nhảy ngược trái phép, có xử lý timeout/unknown không?
- **F-LOG-06 (S1)** — Currency/đơn vị có đi kèm amount ở mọi struct/record không? Có phép cộng 2 amount khác currency không?
- **F-LOG-07 (S1)** — Phí/thuế/tỷ giá có lấy từ bảng cấu hình có hiệu lực theo thời gian (`valid_from/to`) không, hay hardcode?
- **F-LOG-08 (S1)** — Múi giờ & cut-off ngày (`T+0`/`T+1`, đối soát theo ngày nghiệp vụ) có nhất quán giữa nguồn và đích không?
- **F-LOG-09 (S1)** — Đối soát (recon) có so **count + sum(amount)** 2 chiều, có xử lý bản ghi `DELETE`/`reversed` không?

## 3. Lens DESIGN (`F-DES-*`)
- **F-DES-01 (S1)** — Luật tính tiền (fee, FX, rounding) có nằm ở 1 service/domain duy nhất không, hay bị copy vào handler/UI?
- **F-DES-02 (S2)** — Mã tiền có dùng Value Object / kiểu riêng (`Money{Amount, Currency}`) thay vì `int64` trần trộn lẫn count/id không?
- **F-DES-03 (S2)** — Thêm phương thức thanh toán / kênh / đối tác có đi qua registry/strategy không, hay thêm `case` vào god-file?

## 4. Lens CONTRACT (`F-CON-*`)
- **F-CON-01 (S0)** — Đổi tên/kiểu/đơn vị trường tiền (`amount` cent → đồng, `string` → `number`) trong event/API/bảng đang có consumer không?
- **F-CON-02 (S0)** — Event tiền có đủ: `txn_id`, `amount` (minor unit), `currency`, `occurred_at`, `idempotency_key`, `status`? Bỏ trường nào consumer đang dùng không?
- **F-CON-03 (S1)** — Mã lỗi/trạng thái trả về đối tác hoặc UI có đổi nghĩa (ví dụ `FAILED` thành `REJECTED`) không? Có bảng map tương thích không?
- **F-CON-04 (S1)** — Migration bảng tiền/ledger có `backfill` an toàn, chạy ngoài luồng worker, có rollback không?

## Lệnh hỗ trợ (0 LLM token)
```bash
# float trên money-path (Go/TS/Python)
git diff <base>..<head> -U0 | grep -nE '^\+' | grep -niE '(float(32|64)?|double|parseFloat|toFixed|Number\()' | grep -iE 'amount|balance|fee|total|price|rate'
# read-then-write số dư
git diff <base>..<head> -U3 | grep -niE 'balance\s*(-|\+)?=|balance\s*<|balance\s*>='
# log PAN/OTP
git diff <base>..<head> -U0 | grep -nE '^\+' | grep -iE '(log|print|console\.)' | grep -iE 'pan|card|cvv|otp|account_?no'
```
