---
description: Advanced Orchestration — 1 Brain điều phối nhiều Muscle chạy song song (Fan-out)
---

# Parallel Muscle Workflow v3

> Quy tắc #1 (Separation) + #4 (Sub-agent Strategy)
> Brain = Orchestrator. Muscle [1..N] = Workers.

## Khi nào dùng
Trigger: `/parallel-muscle`, hoặc khi task có thể chia nhỏ độc lập (e.g. fix 3 services khác nhau, hoặc chạy Test + Lint song song).

## Workflow Steps

### 1. Task Decomposition (Phân rã Task)
Brain chia nhỏ task thành các đơn vị độc lập (Atomic Tasks).
- **Quy tắc**: Không có 2 Muscle nào được phép sửa cùng 1 file đồng thời.

### 2. Dispatch Muscles
Khởi tạo các session Muscle riêng biệt:
- **Muscle A**: Chịu trách nhiệm Service 1.
- **Muscle B**: Chịu trách nhiệm Service 2.
- **Muscle C**: Chạy verification (QA/Security).

### 3. Model Assignment (Rule #3)
Tối ưu cost:
- Muscles chạy Lint/Audit → Dùng `gemini-3-flash`.
- Muscles chạy Complex Logic Fix → Dùng `gemini-3-pro-high`.

### 4. Conflict Resolution & Sync
- Mỗi Muscle báo cáo DoD riêng.
- Brain tổng hợp và thực hiện Final Merge/Validation.

### 5. Unified Progress Logging
Tất cả Muscle ghi log vào `05_progress.md` của workspace chung:
- `[Muscle-A][Model-Flash] Action description`
- `[Muscle-B][Model-Pro] Action description`

## Anti-patterns
- ❌ Cho 2 Muscle sửa chung 1 file cùng lúc.
- ❌ Không load `models.env` cho từng Muscle instance.
- ❌ Brain can thiệp code của Muscle đang chạy.

## Definition of Done
- [ ] Task đã được chia nhỏ thành các atomic units.
- [ ] Từng Muscle instance đã báo cáo kết quả (DoD).
- [ ] Không xảy ra conflict trong quá trình thực thi.
- [ ] `05_progress.md` ghi nhận đầy đủ Model/Action của từng Muscle.
