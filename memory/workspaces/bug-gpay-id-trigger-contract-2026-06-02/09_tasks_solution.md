# 09_tasks_solution.md — Solution Detail & Reasoning

> Tổng hợp solution thiết kế. Đọc cùng `03_implementation.md` (patch spec) + `04_decisions.md` (ADR).

---

## 1. Root cause (1 dòng)

> **3 lớp claim contract `_gpay_id auto-fill` (comment / Go DDL / migration trigger) — không lớp nào implement thực sự.** Bug ẩn từ design V2 shadow v1.25, bóc lộ khi fix vòng trước (event_handler PK lookup + batch_buffer remap) đẩy flow vào đến INSERT.

---

## 2. Solution chốt (Single Source of Truth)

### Một implementation duy nhất
**DB-side DEFAULT `cdc_internal.sf_nextval()`** trên cột `_gpay_id`.

### Áp dụng ở 2 đường tạo table
| Đường tạo table | Cách áp dụng |
|---|---|
| Migration mới (fresh DB) | `019_sonyflake_default_fill.sql` tạo function + ALTER existing |
| Go runtime `createShadowTable` | DDL emit cột với `DEFAULT cdc_internal.sf_nextval()` |

### 1 sửa nhỏ tài liệu
Comment trong `batch_buffer.go` rewrite chỉ đúng tới function `sf_nextval()` (không còn nói dối "trigger fills").

---

## 3. Tại sao chọn DB-side DEFAULT (vs trigger / Go-side)

| Tiêu chí | DB DEFAULT | BEFORE INSERT trigger | Go-side fill |
|---|---|---|---|
| Cover mọi đường INSERT | ✅ | ✅ | ❌ (chỉ go) |
| Performance | ✅ (no extra fire) | ⚠ (trigger per row) | ✅ |
| Caller có thể override `_gpay_id` | ✅ | ⚠ phức tạp | ✅ |
| Tích hợp với fencing trigger hiện hữu | ✅ độc lập | ⚠ phải chain | ✅ độc lập |
| Patch tối thiểu | ✅ (1 migration + 1 dòng Go) | ❌ (thêm trigger + attach mỗi table) | ❌ (sửa builder + maintain ID source 2 nơi) |
| Single source of truth | ✅ | ⚠ (rủi ro lệch giữa DDL và trigger) | ❌ (Go + DB cùng có cơ chế) |

→ **DB DEFAULT thắng** mọi tiêu chí trừ "Override caller" (cũng OK với DEFAULT vì caller chỉ định giá trị sẽ overide).

---

## 4. Tại sao `sf_nextval()` đọc session var thay vì hard-code

| Cách | Pros | Cons |
|---|---|---|
| Hard-code `machine_id = 0` | Đơn giản | Tất cả pod cùng ID → collision khi scale-out |
| Random per call | Đơn giản, không collision | Mất tính phân tích (track ID về machine không được) |
| **Session var `app.fencing_machine_id`** | Reuse infra fencing, mỗi pod ID riêng, decode được | Phụ thuộc Go bootstrap set var (đã làm sẵn cho fencing) |

→ Reuse session var: 0 thêm infra, mỗi pod sink có machine ID riêng (đã được sonyflake init từ k8s downward API trong Go).

---

## 5. Vì sao chọn `clock_timestamp()` + chia 10 thay vì `nextval(sequence)`

Sonyflake design intent: time-based ID, 10ms granularity, decode về thời gian được.
- `nextval()` cho ID monotonic nhưng KHÔNG encode time.
- `clock_timestamp()` → time-based, decode được, tương thích với `sonyflake.Decompose()` trên Go side.

---

## 6. Trade-off thừa nhận

| Trade-off | Tác động | Mitigation |
|---|---|---|
| 8-bit random sequence collision rate ~ 1/256 trong cùng 10ms cùng machine | RARE — DEFAULT chỉ gọi khi caller không chỉ định, sink batch upsert tốc độ vừa | UNIQUE constraint PK sẽ catch + retry |
| DB function chậm hơn pure Go (~µs vs ns) | Negligible — batch size 5000 = ~5ms overhead | NFR-5 cho phép +5%, dự kiến < 1% |
| Phụ thuộc session var → ad-hoc psql không INSERT được | Đúng intent (chỉ sink mới được ghi V2 shadow) | Doc rõ trong comment function |

---

## 7. Defense-in-depth (đã cân nhắc bỏ)

Ban đầu thiết kế A+B (Go fill + DB fallback). ADR-06 quyết định **CHỈ B** sau khi đánh giá:
- B (DB DEFAULT) đã cover 100% INSERT
- A (Go fill) thêm bề mặt bug đồng bộ machine ID
- Patch tối thiểu (CLAUDE.md §6)

→ Defense thực sự đến từ:
1. CI test AC-1 + AC-2 chạy mỗi PR
2. Grep gate AC-5 chống comment lệch lần nữa
3. ADR-07 dùng testcontainers Postgres thật

---

## 8. Pending Questions cho User

1. **Q1:** Có chấp nhận DB-side DEFAULT (sf_nextval random 8-bit seq) hay yêu cầu monotonic per-machine? (Nếu yêu cầu monotonic → cần advisory lock + per-machine sequence — complexity +1)
2. **Q2:** Migration `019` áp dụng cho **mọi schema** có cột `_gpay_id` — có muốn whitelist schema cụ thể không?
3. **Q3:** Deploy window prod — có cần maintenance window hay deploy live? (Migration metadata-only nên live OK)
4. **Q4:** Có cần backport migration cho service nào khác đang dùng V2 shadow không? (Hiện chỉ centralized-data-service)
5. **Q5:** Lesson global ghi vào `agent/memory/global/lessons.md` — User confirm format pattern abstract?

---

## 9. Lesson candidate (Brain draft, chờ User confirm để append global)

```
Pattern [Layer A claims contract X, Layer B claims contract Y, Layer C claims contract Z — but no layer implements X∩Y∩Z]
→ Result: bug hidden until upstream refactor exposes the missing layer.

Đúng:
  1. Trước mỗi `comment claim "trigger fills"` / `code claim "default sets"`, grep verify symbol/function tồn tại.
  2. CI add gate: grep comment claims vs grep symbol implementations — fail nếu mismatch.
  3. Khi design contract qua nhiều layer (Go DDL ↔ Migration SQL ↔ Comment), pick 1 layer làm canonical source, các layer khác chỉ link tới.

Áp dụng:
  - Dự án CDC (workspace bug-gpay-id-trigger-contract-2026-06-02)
  - Mọi schema migration chéo Go+SQL (web app dùng GORM AutoMigrate + raw SQL migration)
  - Mọi config flag claim ở YAML + code default — drift điển hình
```

---

## 10. Quick reference

| Q | A |
|---|---|
| Tại sao local OK, prod fail? | Local có thể đã chạy migration 003 (V1 sequence DEFAULT) hoặc dev manual patch; prod fresh + sink `createShadowTable` lazy → không có DEFAULT |
| Có cần revert fix vòng trước? | **KHÔNG.** Fix vòng trước đúng intent — chỉ bóc lộ bug ẩn |
| Có đụng V1 (id BIGINT DEFAULT nextval)? | **KHÔNG.** Out-of-scope (ADR-08) |
| Có cần đổi `_source_id` UNIQUE / ON CONFLICT logic? | **KHÔNG.** Fix vòng trước đã xử |
| Migration có rewrite table không? | **KHÔNG.** `ALTER SET DEFAULT` chỉ update `pg_attribute` catalog |
| Sau fix, caller có thể tự chỉ định `_gpay_id` không? | **CÓ.** DEFAULT chỉ kick in khi không chỉ định |
