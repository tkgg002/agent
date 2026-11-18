# Báo Cáo Kiểm Định & Code Review Hợp Nhất (`audit_report_gp2_4633.md`)

- **Workspace:** `review-central-export-gp2-4633-2026-11-17`
- **Repository:** `/Users/trainguyen/Documents/work/centralized-export-service`
- **Phạm vi Review:** `origin/feat/GP2-4633` (`7d9bcf1`) $\rightarrow$ `origin/develop` (`3eef807`) — 14 files thay đổi (`+863` dòng thêm, `-1` dòng xóa)
- **Chế độ:** Read-Only Code Audit (`Propose-Only` — giữ nguyên trạng working tree `feat/GP2-5081`, kiểm định trên Git Worktree cô lập)

---

## Direct Summary
Nhánh `origin/feat/GP2-4633` triển khai tính năng xuất Excel bất đồng bộ `PartnerUserReserveExport` (UC04 — Biến động khoản đảm bảo user) bám sát kiến trúc Pure/Adapter/CQRS của `centralized-export-service`, khớp danh sách tham số lọc với `admin-portal-web` (`0a273871`) và khớp công thức số dư ví đảm bảo (`DAA <-> DRA`) với `core-trans-proxy-history-service-v2` (`9bd6dab`). Bộ kiểm thử của nhánh đạt `25/25` unit tests PASS (`EV-20261117T113407-0bb6`), `7/7` MongoMemoryServer SAGA integration tests PASS (`EV-20261117T113410-f2aa`), biên dịch TypeScript thành công PASS (`EV-20261117T115808-b39e`), nhưng lệnh ESLint hiện FAILED đỏ với `exit=1` (`EV-20261117T113440-cced`). Qua rà soát phản biện từng dòng, em ghi nhận **6 vấn đề (`3 P1`, `3 P2`)** cần khắc phục trước khi merge vào `develop` (đề xuất chi tiết và Code Demo tại `09_tasks_solution_gp2_4633.md`).

---

## Bảng Tổng Hợp Issues & Tổng Kết Hành Động (`Executive Issues Summary Table`)

| Mã | Nhóm | Mức độ | Vị trí (`file:line`) | Mô tả ngắn | Trạng thái & Hành động |
| :--- | :--- | :---: | :--- | :--- | :--- |
| **F1** | CI / Lint | **P1 (High)** | `logics/export/trans-his/partner-user-reserve-export.pure.ts:38` | Ngoặc đơn `(s) =>` vi phạm rule `arrow-parens`, làm `npx eslint` FAILED đỏ (`exit=1`, `EV-20261117T113440-cced`). | **Open** — Sửa thành `s => s === STATUS_SUCCESS` |
| **F2** | Data Integrity & Perf | **P1 (High)** | `domain/handlers/trans-his/GetAllPartnerUserReserveExportHandler.ts:31-35,74`<br>`logics/export/trans-his/partner-user-reserve-export.pure.ts:51-58,118` | Dùng `skip(cursor)` tới `2,000,000` bản ghi và `sort` 1 trường thiếu khóa phụ `_id`: gây lặp/mất dòng khi trùng `value`/`status`/`createdAt` hoặc khi có giao dịch mới phát sinh, quét $O(N^2)$ và dễ chạm giới hạn 100MB in-memory sort của MongoDB. | **Open** — Chuyển sang Keyset Cursor kèm tie-breaker `_id` & `.allowDiskUse(true)` |
| **F3** | Architecture / Cutover | **P1 (High)** | `domain/QueryHandlerFactory.ts:155-160`<br>`logics/export/trans-his/partner-user-reserve.export.ts:22-24`<br>`models/trans-his-pg/trans-his-pg.query.ts:119-123` | Hardcode `TransHisModel` (Mongo) bỏ qua `TRANS_HIS_EXPORT_SOURCE=postgres`: khi ngắt `TRANS_HIS_MONGO_URI` sẽ gãy ở `validatePrerequisites`, hoặc xuất lệch nguồn so với màn danh sách v2 (đọc PostgreSQL). | **Open** — Mở rộng `JSON_STRING_PATHS` + thêm `GetAllPartnerUserReserveExportPgHandler` + `transHisModelNames()` |
| **F4** | Filter & Coercion | **P2 (Medium)** | `logics/export/trans-his/partner-user-reserve-export.pure.ts:34-40,67-72` | `params.status.split(",")` thiếu `.trim()` và lọc rỗng (gửi `"FAILED, SUCCESS"` kèm `statusAfter` làm kết quả rỗng); `toNumber` ép nhầm `"   "`, `boolean`, `[]` thành `0`. | **Open** — Thêm `splitCsv` dùng chung và siết điều kiện đầu vào của `toNumber` |
| **F5** | Validation, Sort & TZ | **P2 (Medium)** | `data-transfers/params/trans-his/partner-user-reserve-export.params.ts:22-23`<br>`logics/export/trans-his/partner-user-reserve-export.pure.ts:54,115-116` | `ExceedDateRangeValidator` cho lọt `dateTo < dateFr`; `buildSort` đảo `"ASC"`/`"ascend"` thành `-1`; `fileName` dùng `moment()` giờ UTC thiếu `.utcOffset(7)` làm lệch ngày từ `00:00` đến `06:59` GMT+7. | **Open** — Chặn `dateDiff >= 0`, chuẩn hóa `sortType` chữ thường, thêm `.utcOffset(7)` cho `fileName` |
| **F6** | Lookup & Test Hygiene | **P2 (Low)** | `domain/handlers/trans-his/GetAllPartnerUserReserveExportHandler.ts:41-43,64`<br>`test/unit/pure/partner-user-reserve.pure.test.ts:8-10` | `_.isEmpty(v)` trả `true` nếu `merchantCode` là kiểu `number`; unit test mock cứng `validateParams` nên chưa kiểm tra thực tế nhánh ném `ValidationError`. | **Open** — Chuẩn hóa `String(...).trim()` và bổ sung test gọi `validateParams` thật |

---

## Mục 1: Phản biện từng dòng update / `git diff` & từng file (`Line-by-Line Review`)

Đối chiếu trực tiếp `git diff origin/develop...origin/feat/GP2-4633` trên 14 files:
1. **`data-transfers/params/index.ts` (+4 dòng, -1 dòng) & `partner-user-reserve-export.params.ts` (+60 dòng):**
   - Khai báo đầy đủ `dateFr`, `dateTo`, `reserveId`, `partnerUserId`, `merchantCode`, `status`, `statusAfter`, `sortBy`, `sortType` khớp với `toReserveMovementParams` trên `admin-portal-web` (`0a273871`).
   - Điểm hở (`F5`): dòng 22–23 tính `dateDiff <= MAX_DAYS_EXPORT` nhưng không kiểm tra `dateDiff >= 0`, cho phép `dateTo < dateFr` đi thẳng vào hàng đợi xuất file rỗng.
2. **`domain/queries/index.ts` (+1 dòng), `GetAllPartnerUserReserveExportQuery.ts` (+11 dòng), `domain/QueryHandlerFactory.ts` (+3 dòng):**
   - Wiring đúng tên class query và handler.
   - Điểm hở (`F3`): `QueryHandlerFactory.ts:155-156` đặt cứng `GetAllPartnerUserReserveExportHandler` ngoài khối `if (getTransHisExportSource() === "postgres")`.
3. **`domain/handlers/trans-his/GetAllPartnerUserReserveExportHandler.ts` (+78 dòng):**
   - Đã dùng `Promise.all` gộp truy vấn trang đầu với `countDocuments`, bỏ `countDocuments` từ trang thứ hai trở đi và bọc `try-catch` khi tra cứu `MerchantModel` để không làm gãy luồng xuất khi thiếu merchant.
   - Đã kiểm chứng chéo với `core-trans-proxy-service` (`publish/reserve-transfer.publish.ts` & `publish/wallet.publish.ts:438-454`): cả hai ví `DAA` và `DRA` đều thuộc cùng chủ ví (Driver), nên đọc `sender.customerId`, `sender.phone`, `sender.fullName` cho cả `PARTNER_RESERVE` lẫn `PARTNER_RESERVE_RELEASE` là đúng dữ liệu gốc.
   - Điểm hở (`F2`, `F6`): dòng 31–35 & 74 phân trang bằng `skip(cursor)` không khóa phụ `_id`; dòng 42 dùng `!_.isEmpty(v)` làm rơi `merchantCode` nếu mang kiểu số.
4. **`logics/export/index.ts` (+1 dòng), `logics/index.ts` (+2 dòng), `partner-user-reserve.export.ts` (+27 dòng):**
   - Adapter mỏng kế thừa `BaseExportProcessor`, tên class `PartnerUserReserveExport` khớp chính xác tham số `exportType: 'PartnerUserReserveExport'` từ CMS (`admin-portal-web/src/pages/partner-user-reserve/list/index.tsx:21,49`).
   - Điểm hở (`F3`): `getRequiredModelName()` (dòng 22–24) trả cứng `["TransHisModel", "MerchantModel"]` thay vì `[...transHisModelNames(), "MerchantModel"]`.
5. **`logics/export/trans-his/partner-user-reserve-export.pure.ts` (+150 dòng):**
   - `reserveBalance` (dòng 79–87), ẩn `statusAfter`/`reserveLimit` khi giao dịch không thành công (dòng 128–130), chống công thức độc hại `safeText` (dòng 61–64), thứ tự 14 cột và giới hạn tên sheet $\le 31$ ký tự đều chính xác.
   - Điểm hở (`F1`, `F4`, `F5`): lỗi ESLint `arrow-parens` ở dòng 38 (`EV-20261117T113440-cced` FAILED đỏ), `status.split(",")` thiếu `trim()` ở dòng 34, `sortType` chưa chuẩn hóa chữ thường/`ascend` ở dòng 54, `toNumber` ép nhầm chuỗi khoảng trắng/boolean thành `0` ở dòng 68–71, và `fileName` thiếu `.utcOffset(7)` ở dòng 115–116.
6. **Bộ 4 file kiểm thử (`+527` dòng):**
   - Unit tests đạt `25/25` PASS (`EV-20261117T113407-0bb6`) và Integration tests với `MongoMemoryServer` + `ExcelJS` thực tế đạt `7/7` PASS (`EV-20261117T113410-f2aa`), tuy nhiên `partner-user-reserve.pure.test.ts:8-10` đang mock cứng `validateParams` (`F6`).

---

## Mục 2: Chống suy diễn Smoke Test / Mockup / Hallucination & Trace Runtime thật

- Kiểm tra thực thi bằng biên lai vật lý trên Git Worktree cô lập (`7d9bcf1`):
  - Unit tests (`vitest run`): `3/3` test files, `25/25` tests PASS (`EV-20261117T113407-0bb6`).
  - Integration tests (`MongoMemoryServer` + `ExcelJS` ghi và đọc lại file `.xlsx` thật qua SAGA): `1/1` test file, `7/7` tests PASS (`EV-20261117T113410-f2aa`), chứng minh luồng runtime tích hợp in-process từ `ExportLogic` $\rightarrow$ `PartnerUserReserveExport.execute` $\rightarrow$ `QueryBus` $\rightarrow$ `FileGenerator` tạo đúng 14 cột Excel (`EV-20261117T113410-f2aa`).
  - Kiểm tra kiểu tĩnh TypeScript (`tsc -p tsconfig.build.json --noEmit`): biên dịch sạch với `exit=0` PASS (`EV-20261117T115808-b39e`).
  - Kiểm tra ESLint (`npx eslint ...`): FAILED đỏ với `1` lỗi `arrow-parens` tại `partner-user-reserve-export.pure.ts:38:59` (`EV-20261117T113440-cced`).
- Tuyệt đối không lấy kết quả vượt qua `MongoMemoryServer` với 5 bản ghi mẫu (`EV-20261117T113410-f2aa`) để suy diễn rằng hệ thống sẽ chịu tải an toàn ở mức `maxRecords: 2_000_000` hay hoạt động được khi bật `TRANS_HIS_EXPORT_SOURCE=postgres` — đây chính là cơ sở xác lập phát hiện `F2` và `F3`.

### Giới hạn & Các hạng mục chưa kiểm chứng trên môi trường thật (Limitations & Not Verified)
- Do đang ở pha Read-Only Code Audit (`Propose-Only`) và tuân thủ nguyên tắc không chạm DB môi trường thật, em chưa chạy thử nghiệm xuất dữ liệu thực tế trên cluster MongoDB/PostgreSQL staging có hàng triệu bản ghi `trans-his` và chưa kích hoạt end-to-end qua `ExportWorker` kết nối Redis/NATS thật. Các hạng mục này sẽ được kiểm chứng tiếp nếu anh duyệt phương án sửa code hoặc phối hợp kiểm tra trên staging.

---

## Mục 3: Kiểm tra Concurrency / Multi-Worker / Timeout-Heartbeat / Data Continuity

1. **Data Continuity & Watermark/Cursor dưới tải ghi đồng thời (`F2`):**
   - `ExportWorker` (`workers/ExportWorker.ts:38-41`) đọc từng chunk tuần tự theo `cursor` cho mỗi job, trong khi dịch vụ giao dịch (`core-trans-proxy-service`) liên tục ghi mới các bản ghi `PARTNER_RESERVE` / `PARTNER_RESERVE_RELEASE` vào `trans-his`.
   - Với `sort = { createdAt: -1 }` và `skip(cursor)` hiện tại, mỗi bản ghi mới chèn vào đầu tập kết quả làm dịch chuyển vị trí offset của tất cả các chunk kế tiếp, gây **nhân bản bản ghi ở ranh giới giữa hai chunk** (mất tính liên tục `continuity` của con trỏ phân trang).
   - Tệ hơn, khi `sortBy` là `value` hoặc `status` (nhiều bản ghi đồng hạng) mà không có khóa phụ duy nhất `_id`, thứ tự trang trong MongoDB không tất định gây cả lặp lẫn sót bản ghi. Chuyển sang **Keyset Cursor `(sortField, _id)`** giải quyết triệt để bài toán continuity này.
2. **Multi-Worker & Memory Footprint:**
   - `FileGenerator` dùng `ExcelJS.stream.xlsx.WorkbookWriter` với `useSharedStrings: false` và `.commit()` từng dòng (`FileGenerator.ts:40-59`), giữ mức tiêu thụ RAM phía Node.js ổn định theo kích thước 1 chunk (`chunkSize = 5000`). Tuy nhiên phía MongoDB server, sort không index trên tập lớn sẽ văng lỗi vượt 100MB RAM nếu thiếu `.allowDiskUse(true)` và keyset filter.
3. **Timeout & Fault Tolerance:**
   - Tra cứu `MerchantModel` đã bọc `try-catch` an toàn (không làm hỏng toàn bộ SAGA khi lỗi kết nối phụ), nhưng ở phía PostgreSQL (`TransHisPgModel` với `QUERY_TIMEOUT_MS = 300_000`), nhánh hiện tại chưa kết nối sang `TransHisPgModel` (`F3`).

---

## Mục 4: Kiểm tra suy diễn & báo cáo láo (`Anti-Speculation & Honesty Check`)

- **Kiểm chứng chéo tránh báo lỗi giả (Zero False-Positive):**
  1. Ban đầu em đặt nghi vấn liệu ở giao dịch hoàn cọc `PARTNER_RESERVE_RELEASE` (`DRA -> DAA`), trường `item.sender` có bị đổi thành ví hệ thống thay vì thông tin người dùng hay không. Thay vì suy diễn, em đã truy vết tận gốc sang `core-trans-proxy-service` (`publish/reserve-transfer.publish.ts:46-74` và `publish/wallet.publish.ts:438-454`) và xác nhận cả hai ví `DAA` lẫn `DRA` đều thuộc sở hữu của cùng một Driver (`getWalletOwner`), do đó `item.sender` luôn mang đúng `customerId`, `phone`, `fullName` của người dùng — loại bỏ hoàn toàn cảnh báo giả.
  2. Tương tự, em đã fetch bản mới nhất của `admin-portal-web` (`0a273871`) và xác nhận FE đã dùng `toReserveMovementParams` ánh xạ `partnerReserveId -> reserveId` trước khi gọi `PartnerUserReserveExport`, tránh báo lỗi lệch tên tham số từ phiên bản FE cũ (`de2d0a04`).
- **Báo cáo trung thực trạng thái Decision Engine (`Rule #10`):**
  - Engine chính VegaML cổng `8222` phản hồi ở endpoint kiểm tra sức khỏe nhưng bị treo quá thời gian chờ (`>120s`) khi suy luận thực tế. Em đã chuyển sang engine dự phòng Laya cổng `8200` qua tham số `--url` đúng quy định Rule #10: ở `DG0` và `DG3` vòng `r2` đạt hiệu chuẩn control (`trusted_rounds: ["r2"]`), còn ở `DG1` và `DG2` không có vòng nào vượt ngưỡng hiệu chuẩn (`trusted_rounds: []` $\rightarrow$ `UNVALIDATED`, mã thoát `exit=0`). Mọi phát hiện và giải pháp đều được kiểm chứng trực tiếp bằng mã nguồn vật lý và biên lai test thật.

---

## Mục 5: Vòng lặp Phản tỉnh & Khắc phục (`Self-Improvement Loop & Lessons`)

- **Các bài học Global đã áp dụng nghiêm ngặt trong phiên:**
  - `L001` (*Reuse Existing Centralized Export Endpoint*): Đối chiếu tên `exportType = "PartnerUserReserveExport"` và luồng gọi qua endpoint xuất bất đồng bộ tập trung trên gateway (`export-auth.ts` chặn chặt tenant merchant).
  - `L029` & `L034` (*Deployed-vs-Local Diff Grounding & Branch-Aware Resolution*): Fetch đầy đủ `origin` trên cả `centralized-export-service` và `admin-portal-web`, phát hiện nhánh đích thực tế là `origin/develop` (thay vì `dev`) và phát hiện commit mới `0a273871` trên FE.
  - `L037` (*Verbatim Current-File Code Demo*): Toàn bộ trích dẫn `REF:` và khối diff trong `09_tasks_solution_gp2_4633.md` lấy nguyên văn từng dòng từ HEAD `7d9bcf1`.
  - `L039` & `L063` (*Direct Summary First & Executive Issues Summary Table*): Trình bày `Direct Summary` đầu tài liệu, sơ đồ topology tổng thể, bảng tổng hợp vấn đề theo mức độ ưu tiên và ánh xạ giải pháp 1-1 (`F1`–`F6`).
  - `L041` (*Clean CLI Args*): Gọi tách rời từng lệnh CLI kiểm định với chuỗi mô tả kiến trúc trung tính.
  - `L046` & `L048` (*Propose-Only Discipline & Git Worktree Isolation*): Working tree chính của anh đang đứng ở `feat/GP2-5081` được giữ nguyên vẹn tuyệt đối; toàn bộ kiểm định chạy trên worktree tách rời.
- **Rà soát vi phạm trong phiên:** Không vi phạm bài học nào trong kho Global Lessons.

---

## Mục 6: Kết quả QC & nhận xét từ `/laya_router` (`DG0`, `DG1/DG2`, `DG3`)

```text
DG0 (Mode A - laya_gate.py run --stage dg0)
  Summary : 06_laya_summary_dg0_20261117T115229.json
  Gate    : PASS (exit=0 sau khi resolve C-REALITY & C-CONTRA kèm REF vật lý vào 04_decisions)
  Trusted : ["r2"]

DG1 + DG2 (Mode A - laya_gate.py run --stage dg1)
  Summary : 06_laya_summary_dg1_20261117T115829.json & 06_laya_summary_dg2_20261117T115829.json
  Gate    : PASS (exit=0, 7/7 claims khớp REF vật lý evidence_error="", trusted_rounds=[] -> UNVALIDATED, kết luận bằng mã nguồn theo Rule #10)

DG3 (Mode A - laya_gate.py run --stage dg3)
  Summary : 06_laya_summary_dg3_20261117T115942.json
  Gate    : PASS (exit=0 sau khi resolve A-F1..A-F4, A-F6, A-SCOPE kèm REF vật lý vào 04_decisions; A-F5 ACCEPT trực tiếp)
  Trusted : ["r2"]

DG3 Debate Review (Mode B - laya_gate.py review)
  Summary : 06_laya_summary_dg3review_20261117T120048.json
  Gate    : BLOCK trên diff gốc của PR (GetAllPartnerUserReserveExportHandler.ts & barrel index) — khớp với các phát hiện F2/F3/F6 đang chờ lệnh APPROVE để khắc phục.
```
