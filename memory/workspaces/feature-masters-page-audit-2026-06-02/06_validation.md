# 06_validation.md — Verify Plan vs Source (Muscle)

> **Ngày**: 2026-06-03 | **Agent**: Muscle:Claude-Opus-4.8 (ultracode workflow `verify-masters-plan`, 10 sub-agent)
> **Mục tiêu**: Kiểm chứng độc lập mọi giả định của `02_plan.md` với source thật TRƯỚC khi execute.
> **Repo base**: `/Users/trainguyen/Documents/work/data-hub/`

---

## Bảng kết quả verify

| ID | Mục kiểm | Giả định của Plan | Verdict | Blocker | Bằng chứng (file:line) |
|----|----------|-------------------|---------|---------|------------------------|
| V1 | Phase 2 SinkWorker DB | "Phải THÊM `DB *gorm.DB` vào Config + wire `worker_server.go`" | ❌ **REFUTED** | ⚠️ CÓ | `sinkworker.go:54` (Config.DB đã có), `:31` (struct.db), `:72-88` (New wire), `cmd/sinkworker/main.go:100-109` (đã pass DB) |
| V2 | approve_master cột | "Có thể sai tên cột master_table/master_name" | ✅ Đúng (lo lắng thừa) | Không | `approve_master.go:70-72` dùng `master_table`; `032_v2_master_binding.sql:13` có cột `master_table` |
| V3 | Hợp đồng API /schedules | Body `{master_table,mode,cron_expr,is_enabled,reason}` | 🟡 PARTIAL | Không | `transmute_schedule_handler.go:68-74` body khớp; `:85` mode chỉ `cron/immediate/post_ingest`; `:97` reason **≥10 ký tự**; RunNow `:166` **không đọc body** |
| V4 | G-4 scheduler mode | Scheduler chỉ tick `mode='cron'` | ✅ Confirmed | Không | `transmute_scheduler.go:109-119` `AND ts.mode='cron'` |
| V5 | G-5 fan-out vô điều kiện | SinkWorker luôn publish transmute-shadow | ✅ Confirmed | ⚠️ cần gate | `sinkworker.go:224-227` gọi `publishTransmuteTrigger` không guard; sig `:235 (shadowSchema,shadowTable,sourceID string)` |
| V6 | Phase 1 FE MasterRegistry | Thiếu Sync UI + import Radio/Tooltip/SyncOutlined/InfoCircleOutlined | ✅ Gap thật; import đúng | minor | `MasterRegistry.tsx:261-294` chỉ có Approve/Reject/Swap; MasterRow có `master_name/schema_status/is_active`; `cmsApi/humanizeApiError/qc/queryKey['master-registry']` đều đúng |
| V7 | Routes + helpers FE | Routes masters/:name, schedules/:id/run-now tồn tại | ✅ Confirmed | Không | `router.go:219-249`; `services/api.ts:11 cmsApi`; `utils/apiError.ts:110 humanizeApiError` |
| V8 | Phase 3 TransmuteSchedules | Cần thêm 3 option mode + tooltip | 🟡 đã có 3 option | Không | `TransmuteSchedules.tsx:216-220` đã có cron/immediate/post_ingest; chỉ cần thêm Tooltip + InfoCircleOutlined + wrap label |

**Adversarial re-check (V2, V3)**: cả 2 đều **CONFIRMED** lại — không lật được kết luận.

---

## Kết luận theo Phase

### ✅ Phase 1 (FE Sync Modal) — EXECUTE-READY (low risk), 1 tinh chỉnh
- Type `MasterRow`, `cmsApi`, `humanizeApiError`, `qc`, `queryKey ['master-registry']`, routes, mode values, reason≥10 validation: **tất cả khớp**.
- Import bổ sung plan ghi đúng: thêm `Radio, Tooltip` (antd) + `SyncOutlined, InfoCircleOutlined` (icons). Các import khác (Modal/Alert/Space/Input/Button/message/Select) **đã có sẵn** — KHÔNG re-import.
- ⚠️ **BUG logic cần sửa**: nhánh `run_now` của plan fetch all schedules rồi `.find(s => s.mode === 'immediate')` — KHÔNG lọc theo master → có thể run-now **nhầm master khác**. Phải sửa: `.find(s => s.mode==='immediate' && s.master_table===row.master_name)`.
- Lưu ý non-blocker: `/schedules/:id/run-now` **bỏ qua body** → field `reason` gửi kèm không được ghi audit (create-immediate đã ghi reason rồi nên chấp nhận được).

### ⚠️ Phase 2 (BE sinkworker gate) — CẦN REVISE TRƯỚC KHI EXECUTE
- Vấn đề plan nêu (G-5 fan-out vô điều kiện) **đúng** và gate logic hợp lý.
- NHƯNG bước plan "thêm `DB *gorm.DB` vào Config + wire `worker_server.go`" **SAI**: DB đã được wire đầy đủ; điểm khởi tạo SinkWorker là `cmd/sinkworker/main.go` **KHÔNG phải** `worker_server.go`. Chạy nguyên văn plan → **compile error (duplicate field)**.
- Hành động đúng: BỎ phần DB-plumbing, CHỈ thêm `hasPostIngestSchedule()` + guard trước `publishTransmuteTrigger`. Cần verify thêm tên cột join `shadow_binding`/`master_binding` lúc implement.

### ✅ Phase 3 (TransmuteSchedules tooltip) — EXECUTE-READY (trivial)
- 3 option mode đã tồn tại → KHÔNG re-add. Chỉ thêm import `Tooltip` + `InfoCircleOutlined` và wrap label `post_ingest`.

---

## Đề xuất đường thực thi
1. **Khuyến nghị**: Execute Phase 1 (kèm fix bug `.find`) + Phase 3 ngay (low risk, độc lập), đồng thời **revise** doc Phase 2 cho đúng rồi mới code.
2. Hoặc execute cả 3, Phase 2 Muscle tự bỏ phần DB sai.
3. Hoặc chỉ revise plan, chưa code.
