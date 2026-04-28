# Phase 37 — Tasks Solution

## Quyết định chốt

**Không thêm lớp đệm.** Sau 21 phase tích lũy, debt prepared-without-caller đã thành noise. Phase 37 phân loại 3 nhóm và xử lý dứt điểm:

| Nhóm | Tình trạng | Quyết định | Action |
|------|-----------|-----------|--------|
| `TransformService` | Dead code | Prune | Xóa `transform_service.go` (-112) |
| Methods raw SQL `registry_repo` | No caller | Prune | Xóa 3 methods (-49) |
| `EventBridge` | Có test, có thể fallback | Reserve | Comment header đánh dấu rõ |

## Code change summary

```
cdc-system/cdc-cms-service/internal/repository/registry_repo.go        +0  -49
cdc-system/centralized-data-service/internal/handler/event_bridge.go   +11 -3
cdc-system/centralized-data-service/internal/service/transform_service.go  DELETED  -112
```

Total: 3 files, +11/-164.

## Lessons học được trong Phase 37

1. **"Prepared path" mà 1 phase sau vẫn không có caller → prune luôn**, đừng giữ với hy vọng tương lai dùng.
2. **Phân loại 3 trạng thái rõ**: dead code (xóa), compatibility reserve (giữ + đánh dấu), wrapper (giữ + nhấc lên caller V2).
3. **Đánh dấu compatibility reserve** bằng comment header để future maintainer không nhầm với runtime chính.

## Follow-up Phase 38+

1. **Phase 38** — Tạo hồi tố bộ docs Phase 37 (✅ phiên này).
2. **Phase 38b** — Re-verify `go test` cả cms + worker sau prune.
3. **Phase 39** — Cắt `is_table_created` dual-write, để `shadow_binding.ddl_status` thành SoT.
4. **Phase 40** — Tách write-path V2 khỏi `cdc_table_registry`.

## Skills đã sử dụng

API contract audit, golang refactor, dead-code identification, compatibility analysis, file deletion, comment annotation, workspace documentation (hồi tố).
