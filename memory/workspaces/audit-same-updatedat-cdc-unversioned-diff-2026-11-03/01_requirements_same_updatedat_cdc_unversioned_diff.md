---
tier: T2
project: data-hub
workspace: audit-same-updatedat-cdc-unversioned-diff-2026-11-03
suffix: same_updatedat_cdc_unversioned_diff
mode: standard
created_at: 2026-11-03T11:26:50+07:00
---

# 01. Requirements — Audit Kịch bản Lệch `unversioned_diff` Khi 2 Lệnh Ghi Cùng Mốc `updatedAt` (Case `261029KHHDUH` — `MERCHANT_PAYMENT`)

## 1. Bối cảnh & Yêu cầu của Người dùng (User Request)
- **Phát hiện thực tế trên giao dịch `261029KHHDUH` (`MERCHANT_PAYMENT`, 7 triệu, nạp tiền cho merchant cần duyệt tay):**
  - Thứ tự lệnh ghi lúc `05:17:04Z` ngày `29/10` bị ngược: `core-trans` gọi `updateTransHisStatus` (`status: PENDING`, `error: 1172`) trước `createTransHis` `23 ms`.
    - Lệnh `update` vẫn trả `n:1, nModified:1`, tức là lúc đó Mongo đã có bản ghi. Nhưng trong 3 giờ trước đó không thấy log v1 nào tạo bản ghi này.
    - Sau đó `createTransHis` chạy `insertOne` với `status: PROCESSING`, và vẫn trả thành công.
  - **Kết quả:** `v1` (Mongo) trả `PENDING`, còn `Postgres` giữ `PROCESSING`, với cùng `updatedAt` (cùng giây `05:17:04`). Đây là nhóm `unversioned_diff`, tức cùng phiên bản mà nội dung khác nhau.
  - Admin xem danh sách `MERCHANT_PAYMENT` và gặp lệch này `34 lần`, từ `06:02Z` tới `10:56Z`. Lúc `10:59:26Z`, lệnh cập nhật `SUCCESS` (có `updatedAt` mới) mới được CDC chép sang và hết lệch.
  - **Giả thuyết / Suy luận của người dùng cần Audit:**
    > *"Theo tôi suy luận, khi hai lần ghi cùng một mốc `updatedAt`, CDC không xác định được lần nào mới hơn, nên Postgres có thể giữ nhầm bản cũ. Nếu sau đó không có lệnh ghi nào nữa, bản ghi trong Postgres sẽ sai mãi. Trong 10 ngày chỉ có đúng một giao dịch như vậy, và nó đã tự hết lệch nhờ lệnh SUCCESS sau đó. => mày audit xem vụ này có khả năng xảy ra ko"*

## 2. Mục tiêu Kiểm định (Audit Scope & Questions)
1. **Kiểm định từng mắt xích trong kịch bản `261029KHHDUH` đối chiếu với mã nguồn thực tế (`core-trans-proxy-history-service`, `wallet-transhis-service`, `centralized-data-service`):**
   - Vì sao `updateTransHisStatus` (`PENDING`, `error 1172`) chạy trước `createTransHis` `23 ms` mà lệnh `update` lại trả `n:1, nModified:1`, và sau đó `createTransHis` chạy `insertOne` (`status: PROCESSING`) vẫn thành công mà cuối cùng Mongo lại giữ `PENDING` trong khi Postgres giữ `PROCESSING`? (Hoặc bản chất thực sự của thứ tự log/thứ tự thực thi MongoDB/Mongoose/Proxy và cơ chế tạo `updatedAt` theo giây hay mili-giây là gì?).
2. **Kiểm định giả thuyết của người dùng về CDC & OCC khi 2 lần ghi có cùng mốc `updatedAt` (hoặc cùng `_source_ts`):**
   - Trong pipeline CDC (`Debezium MongoDB -> Kafka -> centralized-data-service Shadow -> Master Postgres`), trường `_source_ts` lấy từ đâu (`payload.source.ts_ms` độ phân giải giây `1000ms` của MongoDB oplog `ts` hay `updatedAt` ms?), và câu lệnh `ON CONFLICT ... DO UPDATE WHERE ...` ở cả tầng Shadow (`schema_adapter.go`) lẫn tầng Master (`transmuter.go` / `sql_generator.go`) sử dụng toán tử so sánh gì (`<` chặt hay `<=`)?
   - Suy luận *"khi hai lần ghi cùng một mốc `updatedAt` (hoặc cùng giây oplog `ts_ms`), CDC không xác định được lần nào mới hơn, nên Postgres giữ nhầm bản đầu tiên/bản cũ và bỏ qua bản ghi thứ hai; nếu sau đó không có lệnh ghi nào nữa thì bản ghi trong Postgres sẽ sai mãi"* **có khả năng xảy ra không, đúng ở điểm nào và lệch ở điểm nào**?
3. **Đề xuất giải pháp kỹ thuật duy nhất (Single Best Solution) kèm Code Demo (`Propose-Only`):**
   - Nếu phát hiện lỗ hổng thực sự trong cơ chế OCC (`<` vs `<=` hoặc độ phân giải `_source_ts` / `ord` của Debezium MongoDB oplog hoặc luồng ghi `core-trans-proxy-history-service` / `centralized-data-service`), trình bày phương án xử lý triệt để kèm Code Demo nguyên văn từ `HEAD` và dừng chờ `APPROVE`.

## 3. Ràng buộc Thực thi Trích xuất từ Bài học (Technical Constraints from Lessons `INDEX.md`)
- **`L013` (Propose-Only Discipline & Explicit Approval Gate — `LS-GLOBAL-001-propose-only-gate`):** Yêu cầu là `audit xem vụ này có khả năng xảy ra ko` $\rightarrow$ giữ chế độ **100% Read-Only (Propose-Only)** đối với mọi source code dự án, lập đầy đủ `09_tasks_solution_same_updatedat_cdc_unversioned_diff.md` và `audit_report_same_updatedat_cdc_unversioned_diff.md`, hiển thị trực tiếp **Code Demo** trên khung chat và **DỪNG CHỜ `APPROVE`**.
- **`L019` (Pre-Plan Adversarial Critique Wave DG0):** Phản biện độc lập từng giả định trong mô tả của User (về log `updateTransHisStatus` trước `createTransHis` 23ms, về `insertOne` không trùng `_id` hay khác luồng, về độ phân giải giây của `updatedAt` vs Debezium `source.ts_ms`, về điều kiện OCC `<` trên Postgres) đối chiếu với `file:line` thực tế và kiểm định qua `laya_gate.py run --stage dg0`.
- **`L004` (Verbatim Current-File Code Demo & Cross-Layer Default Parameter Check):** Mọi trích dẫn code và Code Demo bắt buộc lấy nguyên văn từ file tại `HEAD` hiện hành.
- **`L006` & `L030` (Direct-Answer Summary First, Executive Issues Summary Table & 1-to-1 Finding Mapping):** Mở đầu phản hồi bằng khối **Direct Summary** 2–4 câu trả lời trực diện câu hỏi *"vụ này có khả năng xảy ra không và tại sao"*; trình bày **Bảng Tổng Hợp Issues (Executive Summary Table)** và ánh xạ giải pháp 1-1 theo từng phát hiện `F1..Fn`.
- **`L028` (End-to-End Callchain Tracing & Writer Normalization):** Truy vết toàn trình từ `core-trans` / `core-trans-proxy-history-service` (`createTransHis`, `updateTransHisStatus`, `unversioned_diff`) $\rightarrow$ MongoDB Oplog / Debezium `source.ts_ms` & `source.ord` $\rightarrow$ `centralized-data-service` (`kafka_consumer.go`, `event_handler.go`, `schema_adapter.go`, `transmuter.go`).
- **`L039` & `L040` (Zero DB CLI Or Write-Shaped SQL):** Chỉ điều tra 100% qua source code tĩnh, không can thiệp DB.
