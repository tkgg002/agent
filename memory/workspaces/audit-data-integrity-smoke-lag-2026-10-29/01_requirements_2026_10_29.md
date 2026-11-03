---
tier: T2
workspace: audit-data-integrity-smoke-lag-2026-10-29
suffix: 2026_10_29
mode: standard
created_at: 2026-10-29T14:07:45+07:00
---

# 01. Requirements: Audit `/data-integrity` Big Data Live Smoke Recon Lag & Transient Drift

## 1. Bối cảnh & Yêu cầu của Người dùng (User Request)
- Trên trang `/data-integrity` với dữ liệu lớn (Big Data) trên môi trường Live của người dùng:
  - Dữ liệu hiển thị kết quả đối soát tại các bảng lớn (như `trans_his` ~4,463,652 bản ghi, `payment_bills` ~6,517,390 bản ghi):
    - `core-trans-proxy-history-service` / `trans_his` -> `cdc_transaction_history_payload`: Source `4,463,652`, Shadow `4,463,652`, Master `4,463,650` (`ingest: 0`, `transmute: -2 (thiếu)`, trạng thái `Lệch`).
    - `payment-bill-service` / `payment_bills` -> `master_payment_bill_service.payment_bills`: Source `6,517,390`, Shadow `6,517,398`, Master `6,517,398` (`ingest: +8 (thừa)`, `transmute: 0`, trạng thái `Lệch`).
  - Dữ liệu Trace thực tế bổ sung từ người dùng trên bảng `bank_requests_bvb` (~15,370,051 bản ghi, chạy lúc `29/10/2026 14:24:17` / UTC `07:24:17`):
    - `14:24:38 29/10/2026 | 800302…bbfd | Smoke Check | Source → Shadow | LỆCH | 15.00s : 15,370,045 → 15,370,051 (+6)`
    - `14:24:23 29/10/2026 | 800302…bbfd | Smoke Check | Shadow → Master | KHỚP | 6ms : 15,370,051 → 15,370,051 (0)`
    - `14:04:44 29/10/2026 | 88879b…9ce7 | Smoke Check | Source → Shadow | LỆCH | 15.00s : 15,369,840 → 15,369,846 (+6)`
    - `14:04:29 29/10/2026 | 88879b…9ce7 | Smoke Check | Shadow → Master | LỆCH | 5ms : 15,369,846 → 15,369,845 (-1)`
    - Cây OTel Trace của `pipeline:bank_requests_bvb`:
      - `prefetch.master` (`pg.count_rows`, `pg.count_deleted_rows`)
      - `prefetch.shadow` (`pg.count_rows`, `pg.count_deleted_rows`)
      - `source_shadow` (`recon.smoke.mongo_estimate` -> `recon.source.estimated_count`, `recon.source.hash_window: bank-requests [05:19:17 -> 07:19:17]`, `pg.hash_window: shadow_banvietbank_connector_service.bank_requests_bvb [05:19:17 -> 07:19:17]`)
      - `shadow_master` (`shadow.max_ts` -> `pg.max_window_ts`, `master.max_ts` -> `pg.max_window_ts`)
  - Phản ánh của người dùng:
    1. *"nó vẫn bị lag kết quả. bản chất nó là khớp nhug chạy smoke lại vị lệch 1,2. sau đó lại đều lại. audit lại cho anh chỗ này"*
    2. *"mọi thay đổi bắt buộc phải đặt đúng layer theo Design Pattern gốc của bạn và tách file/module theo đúng trách nhiệm đơn, tuyệt đối không nhồi nhét thêm vào các file lớn. a thấy nhận xét bị gap do đang có transmute nó k thuyết phục... anh nhớ là logic của anh là quét lùi time để ko bị gap mà"*
- Yêu cầu: Audit toàn diện luồng `/data-integrity` và cơ chế chạy Smoke Recon / lấy số đếm `Source (recs)`, `Shadow (recs)`, `Master (recs)`, `ingest`, `transmute` trên dữ liệu lớn đang nhảy số realtime (in-flight CDC) để tìm chính xác nguyên nhân gốc rễ vì sao bản chất dữ liệu khớp nhưng chạy Smoke Recon lại báo lệch vài bản ghi (`-2`, `+8`, `1, 2`) rồi lần chạy sau lại đều lại, và đề xuất 1 phương án tối ưu nhất kèm Code Demo (Propose-Only chờ `APPROVE`).

## 2. Ràng buộc Thực thi Trích xuất từ Bài học (Technical Constraints from Lessons)
- **L003 (Propose-Only Discipline & Explicit Approval Gate):** Yêu cầu có từ khóa `audit lại` -> giữ nguyên chế độ Read-Only (Propose-Only), lập `09_tasks_solution_2026_10_29.md` qua `DG0` & `DG1`, hiển thị trực tiếp **Code Demo** trên khung chat và dừng chờ lệnh `APPROVE` tường minh trước khi sửa source code.
- **L006 (Single-Responsibility Module Split):** Không nhồi thêm hàm lớn vào God-files; nếu tách logic mới phải đặt đúng file/layer đơn trách nhiệm.
- **L009 (Pre-Plan Adversarial Critique Wave DG0):** Chạy `laya_gate.py run --stage dg0` kiểm định thực tế codebase trước khi chốt phương án trong `09_tasks_solution`.
- **L015 (Targeted Lookup & Prior Workspace Reuse):** Đối chiếu các workspace trước đó về `data-integrity` / `recon` và đọc sâu source code thật của `cdc-cms-web`, `cdc-cms-service`, `centralized-data-service`.
- **L018 (End-to-End Callchain Tracing):** Truy vết toàn trình từ FE (`cdc-cms-web` trang `/data-integrity`) -> API (`cdc-cms-service` / `centralized-data-service`) -> Worker thực thi Smoke Recon (`recon_worker`, `smoke_recon`, query đếm Source/Shadow/Master) -> DB lưu `reconcile_reports`.
- **L019 (Honest Process State & Environment Lock):** Khóa rõ phạm vi phân tích (hiện tượng trên Live Big Data khi chạy Smoke Recon), dẫn chứng chính xác `file:line`.
- **L020 (Direct-Answer Summary First & 1-to-1 Finding Mapping):** Mở đầu phản hồi bằng khối **Direct Summary** 2–4 câu trả lời trực diện câu hỏi; ánh xạ giải pháp 1-1 theo từng phát hiện `F1..Fn`.
- **L029 (Zero Runtime DDL & No Full-Table Scans on Large Databases):** Trên bảng lớn hàng triệu bản ghi (`4.4M`, `6.5M`), cấm quét toàn bảng không giới hạn hoặc đếm không có mốc chặn thời gian đồng nhất (cutoff/watermark).
- **L030 (Full Multi-Tenant Identity Tuple & 1-N-M Lineage):** Bảo toàn truy vết theo đúng `source_object_registry (1) -> shadow_binding (N) -> master_binding (M)` (như `trans_his` có 1 Shadow `trans_his` nhưng gắn nhiều Master: `cdc_transaction_history`, `trans_his`, `cdc_transaction_history_payload`).
- **L042 (Recon Window Alignment & Phantom Drift):** Giữ bất biến trục thời gian đối soát giữa Source $\leftrightarrow$ Shadow (Segment A) và Shadow $\leftrightarrow$ Master (Segment B); nhận diện độ lệch ảo (in-flight / phantom drift) do lệch thời điểm chụp cắt lớp hoặc lệch cutoff khi luồng CDC realtime đang ghi liên tục.
