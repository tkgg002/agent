# Review Report — `gp2_4633`

- **Repo:** `/Users/trainguyen/Documents/work/centralized-export-service` | **Range:** `origin/develop..origin/feat/GP2-4633` (`3eef807..7d9bcf1`)
- **Ngày:** 2026-11-18 | **Agent:** Claude Sonnet 5.5 | **Mode:** Propose-Only
- **tier:** Audit
- **Rules nạp:** L1 base(4) | L2 packs: `fintech` | L3 profile: none (Design lens: finding layer/chokepoint = `HYPOTHESIS`)
- **Scope file:** `scope_gp2_4633.json` (R0, 0 token)

## 1. Verdict
**`REQUEST_CHANGES`** — S0: `0` | S1: `1` | S2: `6` | S3: `0`
> Không có S0 (không mất tiền, không lộ PII, không phá contract đang chạy). Finding quyết định là **F1**: `npm run lint` exit 1 vì lỗi `arrow-parens` (EV-20261117T164927-96f4) nên CI/lint sẽ fail trước khi merge. Unit 25/25 và integration 7/7 đều pass. Các finding còn lại là nợ kỹ thuật (S2), không chặn merge.

## 2. Scope (từ `scope.py`)
| Chỉ số | Giá trị |
| :--- | :--- |
| Files changed / added / deleted | `14 / 8 / 0` |
| LOC +/- | `+863 / -1` |
| Layers touched | không xác định (không có manifest `layer_order`) |
| `chokepoints_touched` | `none` |
| `god_files` (> 600) | `none` |
| `money_path` files | `6` |
| `layer_violations` | `0` |

## 3. Findings (xếp theo severity)
### [S1] `B-CON-lint` — F1: `arrow-parens` làm `npm run lint` fail
- **REF:** `/Users/trainguyen/.gemini/antigravity/brain/e65b1bd5-416c-443f-be0e-bd229e4ecba0/scratch/wt_gp2_4633/logics/export/trans-his/partner-user-reserve-export.pure.ts:38`
- **Quote:**
  ```
  statuses = (statuses ?? [STATUS_SUCCESS]).filter((s) => s === STATUS_SUCCESS);
  ```
- **Vì sao:** Repo đặt `arrow-parens: as-needed`; ESLint báo `38:59 error Unexpected parentheses around single function argument` (1 error, fixable `--fix`).
- **Blast-radius:** pipeline lint/CI của repo; không ảnh hưởng runtime.
- **Gợi ý sửa:** đổi thành `.filter(s => s === STATUS_SUCCESS)` (hoặc `eslint --fix` trên file này).
- **Gate:** `laya dg3 = PASS` | `EV-20261117T164927-96f4` (exit 1)

### [S2] `B-LOG-page` — F2: skip/offset không có tie-breaker `_id`
- **REF:** `.../wt_gp2_4633/domain/handlers/trans-his/GetAllPartnerUserReserveExportHandler.ts:31-39`
- **Quote:**
  ```
  let findQuery = TransHis.find(filter).sort(sort).select(SELECT_FIELDS).limit(limit);
  if (cursor && cursor > 0) { findQuery = findQuery.skip(cursor); }
  ```
- **Vì sao:** Sort mặc định chỉ `{createdAt:-1}` (`partner-user-reserve-export.pure.ts:57`); khi trùng `createdAt` hoặc có bản ghi chèn trong lúc export, các chunk có thể lặp/sót dòng.
- **Blast-radius:** cùng pattern với handler hiện có `GetAllTransHisExportHandler.ts:26-27` (`skip(cursor)`), nên đây là nợ chung của repo, không phải lỗi riêng của PR này.
- **Gợi ý sửa:** thêm `_id` vào sort làm tie-breaker.
- **Gate:** `laya dg3 = PASS`

### [S2] `B-DES-source` — F3: handler luôn đọc Mongo, bỏ qua công tắc `getTransHisExportSource()`
- **REF:** `.../wt_gp2_4633/domain/QueryHandlerFactory.ts:154-160`
- **Quote:**
  ```
  this.handlers.set("GetAllPartnerUserReserveExportQuery", new GetAllPartnerUserReserveExportHandler(mainProcess));
  if (getTransHisExportSource() === "postgres") {
  ```
- **Vì sao:** Có chú thích cố ý ("Postgres source does not take these filters yet"), nên là quyết định có chủ đích. Rủi ro: khi chuyển nguồn sang Postgres, export này vẫn đọc Mongo → dữ liệu lệch nguồn so với các export trans-his khác. `HYPOTHESIS` (chưa có profile).
- **Gợi ý sửa:** ghi rõ trong tài liệu/ticket hoặc thêm nhánh Postgres khi sẵn sàng.
- **Gate:** `laya dg3 = PASS`

### [S2] `B-DES-model` — F3b: danh sách model bị hardcode
- **REF:** `.../wt_gp2_4633/logics/export/trans-his/partner-user-reserve.export.ts:22-24`
- **Quote:**
  ```
  return ["TransHisModel", "MerchantModel"];
  ```
- **Vì sao:** `trans-his.export.ts` dùng `transHisModelNames()`; hardcode làm model dễ lệch khi nguồn dữ liệu đổi.
- **Gợi ý sửa:** spread `transHisModelNames()` như file anh em.
- **Gate:** `laya dg3 = PASS`

### [S2] `B-LOG-trim` — F4: `status` không `trim()` trong khi `statusAfter` có
- **REF:** `.../wt_gp2_4633/logics/export/trans-his/partner-user-reserve-export.pure.ts:34,36`
- **Quote:**
  ```
  String(params.status).split(",");
  ... String(params.statusAfter).split(",").map((s: string) => s.trim())
  ```
- **Vì sao:** `"SUCCESS, FAILED"` → `" FAILED"` không khớp, kết quả lọc sai âm thầm.
- **Gợi ý sửa:** thêm `.map(s => s.trim())` cho `status`.
- **Gate:** `laya dg3 = PASS`

### [S2] `B-LOG-range` — F5: validator cho phép `dateTo < dateFr`
- **REF:** `.../wt_gp2_4633/data-transfers/params/trans-his/partner-user-reserve-export.params.ts:22-23`
- **Quote:**
  ```
  const dateDiff = moment.duration(moment(new Date(params.dateTo)).diff(moment(new Date(params.dateFr)))).asDays();
  return dateDiff <= MAX_DAYS_EXPORT;
  ```
- **Vì sao:** `dateDiff` âm vẫn `<= MAX_DAYS_EXPORT` → pass; kết quả rỗng thay vì báo lỗi nhập liệu.
- **Gợi ý sửa:** thêm điều kiện `dateDiff >= 0`.
- **Gate:** `laya dg3 = PASS`

### [S2] `B-LOG-tz` — F6: tên file dùng giờ máy chủ, trong khi dữ liệu dùng UTC+7
- **REF:** `.../wt_gp2_4633/logics/export/trans-his/partner-user-reserve-export.pure.ts:113-117,125`
- **Quote:**
  ```
  vi: moment().format("DD-MM-YYYY") + "_Bien_dong_dam_bao",
  ... moment(item.createdAt).utcOffset(7).format("DD/MM/YYYY HH:mm:ss")
  ```
- **Vì sao:** Server chạy UTC thì sau 17:00 UTC+7 ngày trong tên file lệch 1 ngày so với dữ liệu.
- **Gợi ý sửa:** `moment().utcOffset(7).format(...)`.
- **Gate:** `laya dg3 = PASS`

### [S2] `F-MONEY-float` — F7: số dư sau biến động tính bằng `number`
- **REF:** `.../wt_gp2_4633/logics/export/trans-his/partner-user-reserve-export.pure.ts:86`
- **Quote:**
  ```
  return { before, after: release ? before - amount : before + amount };
  ```
- **Vì sao:** Pack fintech cảnh báo cộng/trừ tiền bằng float. `HYPOTHESIS`: chưa kiểm số dư thực tế có phần thập phân hay không; nếu là VND số nguyên thì an toàn. Dòng này chỉ phục vụ **hiển thị** trong file export, không ghi sổ.
- **Gợi ý sửa:** xác nhận đơn vị tiền; nếu có thập phân thì dùng thư viện decimal.
- **Gate:** `laya dg3 = PASS`

## 4. Blast-radius tổng hợp
| Chokepoint / Contract | Thay đổi trong diff | Callers/Consumers phải kiểm | Trạng thái |
| :--- | :--- | :--- | :--- |
| Không có (không profile) | thêm Query/Handler/Export mới; sửa `params/index.ts` (+3/-1), `QueryHandlerFactory.ts` (+3), `logics/index.ts` (+2) | đăng ký handler tại `QueryHandlerFactory.ts:156` | OK (chỉ thêm, không đổi chữ ký cũ) |

## 5. Gate (R5)
- Mode B `laya_gate.py review`: `06_laya_summary_dg3review_20261117T165710.json` → `gate=PASS`, vòng 1/3, 10/10 file `ACCEPT`.
- Mode A `laya_gate.py run --stage dg3`: `06_laya_summary_dg3_20261117T165631.json` → `gate=PASS`; 9 claim (F1–F7 + F3b + `OK-SAFETEXT-FORMULA`), cả 9 `ACCEPT`.
- Dropped: `none`.

## 6. Evidence
| EV | Kind | Lệnh | exit |
| :--- | :--- | :--- | :---: |
| `EV-20261117T164324-ba0b` | runtime | `scope.py --pack fintech` → 14 file, money=6, viol=0 | 0 |
| `EV-20261117T164927-96f4` | unit | `eslint` trên `partner-user-reserve-export.pure.ts` | 1 (chứng minh F1) |
| `EV-20261117T164936-81c2` | unit | `vitest run` 3 file test | 0 (25/25) |
| `EV-20261117T165003-feae` | integration | `vitest --config vitest.integration.config.ts` | 0 (7/7) |

Các EV lỗi khác (`-cf8a` pack path sai, `-678c` npm ETIMEDOUT, `-ba08`/`-11b1`/`-50ea`/`-3d32` sai đường dẫn eslint) là lần chạy thử bị hỏng do môi trường, đã chạy lại thành công ở các EV bảng trên; không dùng làm bằng chứng cho finding.

Lưu ý: log vitest có cảnh báo `ioredis ... Port NaN` (thiếu biến môi trường Redis khi chạy test), không làm test fail; không thuộc diff này.

## 7. Không-finding đáng ghi (đã kiểm, OK)
- `B-SEC-formula` (CSV/Excel injection): `safeText` (`partner-user-reserve-export.pure.ts:61-64`) thêm `'` trước ký tự `= + - @ \t \r`; claim `OK-SAFETEXT-FORMULA` = `ACCEPT`.
- NoSQL injection ở filter: các tham số `reserveId/merchantCode/partnerUserId` được ép `String(...).trim()` (`:31-33`), không nhận object.
- `layer_violations = 0`, `god_files = none`, `chokepoints_touched = none`.
- Thiếu manifest `.agent-project.json` → Design lens để `HYPOTHESIS`.
