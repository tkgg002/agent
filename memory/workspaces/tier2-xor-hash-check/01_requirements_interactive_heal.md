# Yêu cầu — Luồng Chữa Lành Tương Tác (Interactive Heal)

## Bối cảnh
Khi bấm "Chữa lành" trên UI, gateway dispatch `ReconHealCommand` → worker nhận và **vừa chạy đối soát (RunTier2/RunSegmentBFor) vừa heal** — vi phạm SRP. Payload `ReconHealCommand` thực chất là tham số đối soát, không phải tham số thực thi heal.

## Yêu cầu từ User
1. **Tách biệt**: Recon (Check) ≠ Execution (Heal). Không nhồi 2 action vào 1 command.
2. **Batch**: Lấy TOÀN BỘ report chưa heal, không chỉ mới nhất.
3. **Granular**: 3 checkboxes riêng biệt — Mismatched / Missing Dest / Prune Src.
4. **Segment A+B**: Bao phủ cả Source↔Shadow (A) và Shadow↔Master (B).
5. **Thống kê**: 6 cột mới cho thống kê chi tiết từng checkbox (count + duration).
6. **Tuân thủ Architecture**: CQRS, NATS async bus, handler layer, GORM adapters.

## Luồng mới
```
[Bước 1] POST /reconciliation/check/:table → NATS → Worker RunTier2/RunSegmentBFor → Ghi report
[Bước 2] GET /reconciliation/report/:table/unhealed → Danh sách report chưa heal
[Bước 3] POST /reconciliation/execute-heal → NATS → Worker load report → heal/prune granular
```

## Scope
- Backend Gateway: 9 files (7 modify + 2 new)
- Backend Worker: 4 files (2 modify + 1 new + 1 deprecate)
- DB Migration: 1 file (6 cột mới)
- Frontend: 3 files (hooks + components)
