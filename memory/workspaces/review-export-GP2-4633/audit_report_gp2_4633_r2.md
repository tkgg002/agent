# Audit Report — `gp2_4633_r2` (diff-review chạy lại từ đầu)

- **tier:** T2
- **Repo:** `/Users/trainguyen/Documents/work/centralized-export-service` | **Range:** `origin/develop..origin/feat/GP2-4633` (`3eef807..7d9bcf1`), 1 commit, 14 file, +863/-1
- **Ngày:** 2026-11-18 | **Agent:** Claude Sonnet 5.5 | **Mode:** Propose-Only
- **Rules nạp:** L1 base(4) | L2 `fintech` | L3 none (finding layer/chokepoint = `HYPOTHESIS`)
- **Scope:** `scope_gp2_4633_r2.json` (cps=0, god=0, money=6, layer_violations=0)

## 1. Verdict
**`REQUEST_CHANGES`** — S0: `0` | S1: `1` | S2: `10` | S3: `0`
> Không có S0. Finding chặn merge là **F1** (`npm run lint` exit 1, `EV-20261118T094752-3cf0`). Unit 25/25 (`EV-20261118T094752-acd2`), integration 7/7 (`EV-20261118T094752-25a3`). Phần Design lens (D1–D5) là mới so với lượt đầu.

## 2. Phản biện từng dòng `git diff` (REF tại worktree head `7d9bcf1`, `WT=/Users/trainguyen/.gemini/antigravity/brain/e65b1bd5-416c-443f-be0e-bd229e4ecba0/scratch/wt_gp2_4633`)

### Logic / Contract
| Mã | Sev | Phát hiện | REF |
| :--- | :---: | :--- | :--- |
| F1 | S1 | `(s) => s === STATUS_SUCCESS` vi phạm `arrow-parens: as-needed`, lint fail | `WT/logics/export/trans-his/partner-user-reserve-export.pure.ts:38` |
| F2 | S2 | skip/offset không có tie-breaker `_id` (nợ chung repo, handler canonical cũng vậy: `GetAllTransHisExportHandler.ts:26-27`) | `WT/domain/handlers/trans-his/GetAllPartnerUserReserveExportHandler.ts:31-34` |
| F4 | S2 | `status` split không `trim()`, trong khi `statusAfter` có | `WT/logics/export/trans-his/partner-user-reserve-export.pure.ts:34-36` |
| F5 | S2 | validator cho phép `dateTo < dateFr` | `WT/data-transfers/params/trans-his/partner-user-reserve-export.params.ts:22-23` |
| F6 | S2 | `fileName` dùng `moment()` thiếu `.utcOffset(7)` (dòng 125 thì có) | `WT/logics/export/trans-his/partner-user-reserve-export.pure.ts:113-117` |
| F7 | S2 (HYPOTHESIS) | `before ± amount` bằng float; chỉ để hiển thị | `WT/logics/export/trans-his/partner-user-reserve-export.pure.ts:79-87` |

### Design lens (R3) — đối chiếu file canonical
| Mã | Rule | Sev | File mới | Canonical | Lệch |
| :--- | :--- | :---: | :--- | :--- | :--- |
| D1 | B-DES-03/07 | S2 | `WT/logics/export/trans-his/partner-user-reserve.export.ts:22-24` hardcode `["TransHisModel","MerchantModel"]` | `internal-transfer.export.ts:24-26` dùng `transHisModelNames()` | không theo `TRANS_HIS_EXPORT_SOURCE` |
| D2 | B-DES-05 | S2 | `WT/domain/QueryHandlerFactory.ts:154-160` đăng ký Mongo handler ngoài nhánh `getTransHisExportSource()` (có comment chủ ý) | các handler trans-his khác nằm trong nhánh switch | khi nguồn `postgres`, model Mongo có thể không nạp (HYPOTHESIS, cần xác nhận cấu hình deploy) |
| D3 | B-DES-03 | S2 | `WT/domain/queries/trans-his/GetAllPartnerUserReserveExportQuery.ts:3-11` bỏ `select` | `GetAllTransHisExportQuery.ts:4-13` có `select`; `internal-transfer-export.pure.ts:137,168` truyền `SELECT_FIELDS` từ pure | handler tự hardcode projection (`GetAllPartnerUserReserveExportHandler.ts:7,31`) |
| D4 | B-DES-07 | S2 | `WT/data-transfers/params/trans-his/partner-user-reserve-export.params.ts:15-24` copy `ExceedDateRangeValidator` (bản rút gọn) | `trans-his-export.params.ts:14-30`; repo đã có nhiều bản cùng tên trong `data-transfers` | nợ chung repo nhưng PR thêm bản copy mới |
| D5 | B-DES-03 | S2 | `WT/data-transfers/params/trans-his/partner-user-reserve-export.params.ts:32-53` khai báo lại `dateFr/dateTo/status/sort*` | `InternalTransferExportParams extends TransHisExportParams` | không kế thừa base |

**Tuân thủ (đã kiểm):** tách `pure` / adapter / handler / query đúng khuôn; đăng ký qua `logics/index.ts:126-130`, `export.logic.ts:23-25` tự lấy `Object.keys(AllProcessors)` nên không cần sửa; không layer violation, không god-file, không chạm chokepoint; `safeText` (`pure.ts:61-64`) chặn formula injection; `buildSort` dùng whitelist `SORTABLE` an toàn hơn canonical; `TRANSACTION_STATUS_LIST` được dùng thật (`pure.ts:131`).

## 3. Chống suy diễn & Runtime thật
- Lint: `EV-20261118T094752-3cf0` exit 1 (`38:59 arrow-parens`). Unit `-acd2` exit 0 (25/25). Integration `-25a3` exit 0 (7/7). Chạy trên worktree `7d9bcf1`, không smoke/mock tự dựng.
- Test pass không chứng minh multi-page/tie-breaker: F2 chỉ phân tích code, không có runtime trace trên dữ liệu thật → ghi `HYPOTHESIS` về mức độ ảnh hưởng.

## 4. Concurrency / Timeout / Watermark
- Phân trang skip/offset + cursor trên sort chỉ `createdAt` (F2): ghi chèn trong lúc export có thể trùng/mất dòng giữa các chunk; không có watermark liên tục. Canonical có cùng đặc điểm → nợ chung, không phải regression riêng PR.
- Merchant lookup (`Handler.ts:47-60`) bọc try/catch, lỗi chỉ cảnh báo (kết quả control_true).

## 5. Trung thực / suy diễn
- F7 và D2 là `HYPOTHESIS`. Lượt đầu em tái dùng claims cũ nên thiếu D3–D5; lượt này đã đối chiếu canonical từng file. Không dùng lại PASS/EV của lượt cũ.

## 6. Self-improvement / lessons
- Lessons áp dụng: `L001`, `L010`, `L012`, `L030`, `L049`, `L062`.
- Phát hiện quy trình: DG3 từng treo vì Vega `/v1/predict` kẹt GPU MPS dù `/health`=200 → sửa `laya_gate.py` fail-fast 3s + cooldown + fallback Laya (workspace `vega-failfast-fallback-2026-11-18`, `EV-20261118T101412-2f6f`, 13/13). Cần ghi lesson global sau (đề xuất).

## 7. Kết quả laya_gate (DG3)
- Mode A `run --stage dg3 --workers 1`: **PASS**, engine Vega `:8222`, 13 claim ACCEPT (kèm control), file `06_laya_summary_dg3_20261118T101555.json`.
- Mode B `review` (diff 993 dòng, 14 file): **PASS**, file `06_laya_summary_dg3review_20261118T101621.json`.
- Lưu ý trung thực: lần chạy `workers=3` mặc định (`06_laya_summary_dg3_20261118T101448.json`) bị Vega quá 3s nên toàn bộ rơi Laya và ra `UNVALIDATED` (PASS nhưng không có quyết định thực). Không dùng làm bằng chứng. 

## 8. Timing (bấm giờ thật)
| Bước | Thời gian |
| :--- | ---: |
| R0 scope | 0.69s |
| lint / unit / integration (song song) | 4.54s / 2.24s / 5.09s |
| probe git/grep R3 | 0.1–0.3s mỗi lệnh (probe cả repo 21.4s) |
| R5 Mode A trước fix (Vega kẹt) | 242s → `LAYA_DOWN` |
| R5 Mode A sau restart + fail-fast, workers=3 | 18.6s (rơi Laya) |
| R5 Mode A, workers=1 | 12.0s (Vega thực) |
| R5 Mode B review | 8.0s |
