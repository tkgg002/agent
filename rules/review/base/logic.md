# L1 Base Rules — Lens LOGIC & RUNTIME (`B-LOG-*`)

> Ground truth = code sau diff + callers + runtime thật. Diff chỉ là cửa vào. Mỗi finding cần `REF:`.
> Phân biệt rõ `VERIFIED` (đọc code/log chứng minh) và `HYPOTHESIS` (cần chạy thêm).

## 1. Tính đúng của thay đổi
- **B-LOG-01 (S1)** — Nếu là bugfix: có reproduce Red → Green (test fail trước, pass sau) không? Chỉ có "build OK" ≠ fix.
- **B-LOG-02 (S1)** — Hàm bị sửa có **callers** nào khác không (`grep` tên hàm toàn repo)? Mỗi caller còn đúng hành vi mong đợi không? Liệt kê từng caller `file:line`.
- **B-LOG-03 (S1)** — Có thay đổi hành vi mặc định (default param, default branch, giá trị trả về khi lỗi) mà caller cũ đang dựa vào không?
- **B-LOG-04 (S2)** — Nhánh điều kiện mới có phủ đủ các giá trị enum/type hiện có không? Có `default`/`else` xử lý giá trị lạ an toàn không?

## 2. Biên & lỗi
- **B-LOG-05 (S1)** — nil/null/empty/0/negative/overflow/unicode/khoảng trắng/dấu chấm trong tên (`schema.table`) có được xử lý không?
- **B-LOG-06 (S1)** — Lỗi có bị nuốt (`_ = err`, `catch {}`), bị ghi đè, hay mất context (không wrap) không? Lỗi partial (xử lý được 1 phần batch) có được báo đúng không?
- **B-LOG-07 (S1)** — Retry có giới hạn số lần, backoff, và **idempotent** không? Retry trên lỗi không nên retry (4xx, validation) không?
- **B-LOG-08 (S2)** — Timeout/context cancel có được truyền xuyên suốt (DB, HTTP, queue) không? Có `context.Background()` / không timeout ở luồng request không?

## 3. Concurrency & trạng thái
- **B-LOG-09 (S0)** — Có race trên dữ liệu dùng chung (map, slice, counter, cache, singleton) không? Kiểm tra số dư/điều kiện rồi ghi có nằm trong cùng transaction/lock không (check-then-act)?
- **B-LOG-10 (S1)** — Multi-worker/multi-instance: có 2 tiến trình cùng xử lý 1 item không? Lock có TTL ≥ thời gian xử lý, có fencing token/version không? TTL lock khớp chu kỳ ticker không?
- **B-LOG-11 (S1)** — Goroutine/thread/timer nền có được dừng đúng thứ tự trước khi đóng DB/queue/cache không? Có leak (channel không đóng, listener không huỷ) không?
- **B-LOG-12 (S1)** — Watermark/cursor/offset/sequence có **liên tục** không (không nhảy, không lùi, không mất khi crash giữa chừng)? Ack/commit có xảy ra **sau** khi ghi bền không?
- **B-LOG-13 (S1)** — Batch/debounce/buffer: item lỗi có làm chặn toàn batch không? Flush có xảy ra khi shutdown không? Dedup theo khoá nào, giữ bản mới nhất theo tiêu chí gì?

## 4. Transaction & dữ liệu
- **B-LOG-14 (S0)** — Ghi DB + publish event/gọi external có cơ chế bảo toàn (outbox, saga, idempotency key) không? Commit DB rồi publish fail → mất event?
- **B-LOG-15 (S1)** — Transaction boundary có bao đủ các bước phụ thuộc không? Có gọi external I/O lâu bên trong transaction giữ lock không?
- **B-LOG-16 (S1)** — Dữ liệu thời gian: timezone rõ ràng? Lưu UTC? So sánh cùng độ chính xác? `now()` có được inject để test được không?
- **B-LOG-17 (S1)** — Số học: dùng float cho giá trị cần chính xác (tiền, số lượng, tỉ lệ) không? Rounding policy nhất quán không? Chia cho 0?
- **B-LOG-18 (S2)** — Output đích có khớp nguồn (count nguồn == count đích, không trùng, không thiếu) — có cách đối soát không?

## 5. Test & bằng chứng
- **B-LOG-19 (S1)** — Test có mock/closure giả (`*Fn`, interface fake) **gắn vào struct production** không? Test in-memory có bị suy diễn thành "chạy thật đã pass" không?
- **B-LOG-20 (S2)** — Test mới có cover đúng nhánh thay đổi (happy + ≥1 error + ≥1 edge) không? Có test bị `skip`/`t.Skip` âm thầm không?
- **B-LOG-21 (S1)** — Có bằng chứng runtime thật (lệnh + output, biên lai `EV-...`) cho khẳng định "đã chạy OK" không?

## Lệnh hỗ trợ (0 LLM token)
```bash
# callers của mọi hàm/method bị đổi (Go) — thay <Name>
git diff <base>..<head> -U0 | grep -E '^[-+]func ' | sed -E 's/.*func (\([^)]*\) )?([A-Za-z0-9_]+).*/\2/' | sort -u
grep -rn --include='*.go' '\b<Name>(' <repo> | grep -v _test.go
# test bị skip
git diff <base>..<head> | grep -nE '^\+.*(t\.Skip|xit\(|it\.skip|@pytest\.mark\.skip)'
```
