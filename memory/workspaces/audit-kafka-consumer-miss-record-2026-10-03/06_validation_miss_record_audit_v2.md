# 06 — Validation: Laya `/laya_router` per-claim (v2)

> Ngày: 2026-10-04 · Validator: server local FastAPI `127.0.0.1:8200` (`POST /v1/predict`, model `convaiinnovations/laya`, PID 77549).
> Client thật (không hardcode verdict): `~/.gemini/antigravity/brain/fbec8ea4-962c-47ad-815b-93387b6bcc24/scratch/laya_claim_check.py` (r1), `.../laya_debate_rounds.py` (r2, r3).
> Raw JSON: `06_laya_raw_round1.json`, `06_laya_raw_round2_3.json` (cùng thư mục). Input claims: `scratch/claims_round1.json`.
> Giao thức: ≥3 vòng (r1 = câu hỏi 3 nhãn; r2 = thêm feature_objective/function_workflow/diff_lines; r3 = YES/NO nhị phân). Reject confidence thấp KHÔNG được tin.

## 1. Negative/positive controls (test Laya có phân biệt được không)

| ID | Loại | Nội dung | r1 | r2 | r3 | Đánh giá |
|---|---|---|---|---|---|---|
| K1 | claim SAI có chủ đích | "Flush bỏ qua commit khi có group lỗi" | CORRECT 0.54 ✗ | UNSUPPORTED 0.28 ✗ | NO 0.93 ✓ | r1 sai với confidence cao nhất bảng |
| K2 | claim SAI có chủ đích | "10s ctx cancel SQL trong db.Transaction" | INCORRECT 0.07 ✓(yếu) | CORRECT 0.27 ✗ | YES 0.09 ✗ | lật kèo |
| K3 | claim ĐÚNG | "Add append dưới mutex + non-blocking flushCh" | UNSUPPORTED 0.40 ✗ | CORRECT 0.89 ✓ | YES 0.79 ✓ | r1 sai |

**Kết luận về công cụ:** Laya là *router/classifier*, không phải verifier entailment code. Verdict **không ổn định giữa các vòng** (K1, K2, K3 đều lật). Chỉ r3 (nhị phân) đúng 2/3 control với confidence cao. ⇒ Verdict Laya chỉ là **tín hiệu tham khảo**; *ground truth = đọc đúng dòng code* (cột "Code" bên dưới).

## 2. Claims

| ID | Nhận định (rút gọn) | r1 | r2 | r3 | Code đã đọc (file:line) | Kết luận |
|---|---|---|---|---|---|---|
| C1 | Commit offset cả batch dù có group lỗi | UNSUP 0.06 | CORRECT 0.15 | NO 0.002 (non-decisive) | `batch_buffer.go:246-257, 276-291` — `err` không được đọc trước `onCommitOffsets` | **VERIFIED (code)** — Laya không quyết định |
| C2 | processEvent báo success trước khi ghi DB | UNSUP 0.04 | CORRECT 0.02 | YES 0.17 | `event_handler.go:460-462`, `kafka_consumer.go:473-483` | **VERIFIED (code)** |
| C3 | Fallback gặp lỗi retryable → return, bỏ phần còn lại chunk + chunk sau, không DLQ | UNSUP 0.17 | CORRECT 0.51 | YES 0.69 | `batch_buffer.go:474-479, 500-505, 384, 596-628` | **VERIFIED (code) + Laya r2/r3 đồng ý** |
| C4 | SQLSTATE 53300 không retryable → DLQ; DLQ fail chỉ metric; vẫn commit | UNSUP 0.10 | CORRECT 0.47 | YES 0.82 | `batch_buffer.go:596-628` (không có 53300), `:578-593`, `:279-291` | **VERIFIED (code) + Laya r2/r3 đồng ý** |
| C5 | Msg lỗi vào DLQ nhưng offset bị commit vượt qua bởi msg sau cùng partition | INCORR 0.02 | CORRECT 0.28 | YES 0.77 | `kafka_consumer.go:500-526`; `event_handler.go:288,367,418` (return trước `Add`); `batch_buffer.go:279-291` | **VERIFIED (code)**; r1 INCORRECT confidence 0.02 = non-decisive |
| C6 | Silent drop `return 0,nil` (empty/foreign/nil_after/missing_pk) | CORRECT 0.25 | CORRECT 0.52 | YES 0.86 | `kafka_consumer.go:538-555, 631-640`; `event_handler.go:343-350` | **VERIFIED (code) + Laya 3/3 đồng ý** |
| C7 | Flush không tuần tự → commit offset đảo thứ tự | UNSUP 0.16 | CORRECT 0.13 | YES 0.51 | `batch_buffer.go:217-225, 237-291`; callers `snapshot_runner_handler.go:827,889`, `recon_heal_fetch.go:91,104` | **VERIFIED (cấu trúc code)**; xảy ra thực tế chỉ khi snapshot/recon chạy cùng CDC — chưa có log prod |
| C8 | `RefreshTopics` từ NATS goroutine, `kc.batches` không lock → concurrent map | INCORR 0.05 | CORRECT 0.38 | YES 0.89 | `server_setup.go:506-508`, `topic_helper.go:43-59`, `adaptive_batcher.go:79-85,123-126` | **VERIFIED (code)**; chỉ kích hoạt khi topic-set đổi; hậu quả = crash/redelivery, KHÔNG phải mất record |
| C9 | Rebalance/rebuild reader chỉ gây trùng, không gây miss (khi flush ghi thành công) | UNSUP 0.17 | INCORR 0.16 | YES 0.53 | `kafka_consumer.go:168-181`, `batch_buffer.go:279-291` | **Code-reasoned**; Laya mâu thuẫn giữa các vòng; chưa thực nghiệm |
| C10 | Số kết nối tăng tuyến tính theo số instance (100/gorm pool + 100/pgx pool, không điều phối global) | INCORR 0.07 | CORRECT 0.28 | NO 0.08 | `config-production.yml:17-20`, `multi.go:237,257-258`, `postgres.go:34,63,108` | **Số liệu VERIFIED; hệ quả vượt `max_connections` = HYPOTHESIS (cần số liệu PG prod)** |
| C11 | `resolveDB` fallback `bb.db` (control-plane) khi key rỗng/GetShadowDB lỗi | UNSUP 0.08 | INCORR 0.12 | NO 0.49 | `batch_buffer_utils.go:41-58`, `server_setup.go:152`, `metadata_registry_service.go:254` | **Code path VERIFIED; điều kiện kích hoạt (ShadowConnectionID không có trong `connections`) = UNVERIFIED, phụ thuộc dữ liệu** |

## 3. Claim đã bị loại/hạ cấp (qua kiểm tra code, không qua Laya)

| Claim cũ | Kết quả |
|---|---|
| "10s timeout của FlushWithContext cắt SQL" (ý tưởng ban đầu) | **REFUTED** — `db.Transaction` không `WithContext(ctx)` (`batch_buffer.go:392`), `resolveDB` dùng `context.Background()` (`batch_buffer_utils.go:48`). Chính là K2. |
| F7 cũ "RefreshTopics churn liên tục mỗi 60s" | **Hạ cấp** — chỉ rebuild reader khi topic-set đổi (`topic_helper.go:43-47`). |
| F3(c) cũ "lỗi retryable bất kỳ → mất" | **Thu hẹp** — chỉ khi lỗi retryable còn tồn tại ở statement đơn lẻ (fallback), xem C3. |
| Registry reload tạo cửa sổ cache rỗng | **REFUTED** — swap dưới `Lock` (`metadata_registry_service.go:161-183`). |
| OCC gây thiếu record | **REFUTED** — OCC chỉ gây lệch (drift), không gây thiếu. |

## 4. Giới hạn
- Không có log/DB/metric production → mọi kết luận là **phân tích tĩnh**. Mức "VERIFIED (code)" nghĩa là *đường code tồn tại*, không phải *đã xảy ra ở prod*.
- Chưa chạy test (Brain không sửa code; chờ APPROVE). Không có RED test → chưa qua Gate G2/G3.
