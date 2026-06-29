# Plan: fe-data-integrity-empty-db-table
**Ngày**: 2026-06-24 | **Agent**: Brain | **Status**: DRAFT — Chờ User Approve

---

## 1. Phân tích Yêu cầu

Feature `fe-data-integrity-empty-db-table` = Hiển thị **cảnh báo rõ ràng** trên trang Data Integrity khi bảng DB đích (shadow hoặc master) **rỗng hoàn toàn** (`full_dest_count == 0` hoặc count tương ứng == 0), giúp operator nhận diện ngay pipeline chưa có data thực tế.

### Vấn đề hiện tại
1. **ReconPipelineGrid**: Cột "Shadow (recs)" / "Master (recs)" hiển thị số hoặc "—" (null), nhưng **khi = 0 thì không có signal gì khác biệt** với "có data". User không thể phân biệt nhanh pipeline rỗng vs pipeline đã sync.
2. **Tab Tổng quan**: Tương tự — cột "Total Dest" = 0 không có tag/màu nào cảnh báo.
3. **overallStatus()**: Logic chỉ check `drift/error/lag`, **không check empty-table** → pipeline rỗng vẫn hiện "Khớp" (green), misleading.

---

## 2. Phân tích Data Flow

```
Backend API → /api/reconciliation/report
  ↓ join với cdc_table_registry
  ↓ trả full_source_count, full_dest_count (daily aggregated)
  ↓ trả source_count, dest_count (7d window)

Frontend ReconReport type:
  full_source_count?: number | null   ← TỔNG record thực (từ aggregator, daily)
  full_dest_count?: number | null     ← TỔNG record thực (dest/shadow)
  source_count: number | null         ← 7d window
  dest_count: number                  ← 7d window
```

**Chú ý**: `total_source_count` / `total_dest_count` trong `ReconRow` (line 20-21 useReconStatus.ts) là **tại thời điểm recon** (migration 084), khác với `full_source_count` / `full_dest_count` từ table registry (daily aggregator). PipelineRow dùng `total_source_count` / `total_dest_count`.

### Logic xác định "bảng rỗng"
Một pipeline được coi là **empty-table** khi:
- `masterCount == 0` → Master table rỗng (trạm cuối) — serious signal
- `shadowCount == 0 && masterCount == null` → Shadow rỗng, chưa có master
- `sourceCount == 0` → Source rỗng (DB gốc trống)

Điều kiện phân biệt với `null`:
- `null` = chưa có data / chưa recon → hiện "—"
- `=== 0` = đã recon, đếm được, NHƯNG rỗng → cần cảnh báo

---

## 3. Giải pháp (Minimal Impact — Simplicity First)

### Thay đổi 1: `ReconPipelineGrid.tsx` — 2 điểm sửa

#### A. Hàm `overallStatus()` — thêm case `empty`
```tsx
// TRƯỚC (line 56-61):
function overallStatus(p: PipelineRow): { label: string; color: string } {
  if (p.statusA === 'error' || p.statusB === 'error') return { label: 'Cảnh báo', color: 'volcano' };
  if (p.statusA === 'drift' || p.statusB === 'drift' || (p.driftBC ?? 0) !== 0) return { label: 'Lệch', color: 'red' };
  if ((p.ingestLagMs ?? 0) > LAGGING_MS || (p.transmuteLagMs ?? 0) > LAGGING_MS) return { label: 'Lagging', color: 'gold' };
  return { label: 'Khớp', color: 'green' };
}

// SAU:
function overallStatus(p: PipelineRow): { label: string; color: string } {
  if (p.statusA === 'error' || p.statusB === 'error') return { label: 'Cảnh báo', color: 'volcano' };
  if (p.statusA === 'drift' || p.statusB === 'drift' || (p.driftBC ?? 0) !== 0) return { label: 'Lệch', color: 'red' };
  if ((p.ingestLagMs ?? 0) > LAGGING_MS || (p.transmuteLagMs ?? 0) > LAGGING_MS) return { label: 'Lagging', color: 'gold' };
  // Cảnh báo bảng rỗng: recon đã chạy (không phải null) nhưng count = 0
  if (p.masterCount === 0 || (p.shadowCount === 0 && p.masterName == null)) return { label: 'Bảng rỗng', color: 'cyan' };
  return { label: 'Khớp', color: 'green' };
}
```

#### B. Cột "Shadow (recs)" / "Master (recs)" — format số 0 nổi bật
```tsx
// Thêm helper:
function fmtCount(v: number | null | undefined, warnIfZero = false) {
  if (v == null) return <Text type="secondary">—</Text>;
  if (warnIfZero && v === 0) return <Tag color="cyan" style={{ fontFamily: 'monospace' }}>0 (rỗng)</Tag>;
  return v.toLocaleString();
}

// Cột Shadow (recs) — line 603-604:
render: (_: unknown, p) => fmtCount(p.shadowCount, true)

// Cột Master (recs) — line 610-611:
render: (_: unknown, p) => fmtCount(p.masterCount, p.masterName != null)
```

### Thay đổi 2: `DataIntegrity.tsx` — Tab "Tổng quan"

Column "Total Dest" (line 475-479) — thêm cảnh báo khi = 0:
```tsx
// TRƯỚC:
render: (v: number | null | undefined) =>
  v == null ? <Text type="secondary">—</Text> : v.toLocaleString(),

// SAU:
render: (v: number | null | undefined) => {
  if (v == null) return <Text type="secondary">—</Text>;
  if (v === 0) return <Tooltip title="Bảng dest đang rỗng — chưa có dữ liệu hoặc pipeline chưa chạy">
    <Tag color="cyan">0 (rỗng)</Tag>
  </Tooltip>;
  return v.toLocaleString();
},
```

Column "Total Source" (line 462-464) — tương tự:
```tsx
// SAU:
render: (v: number | null | undefined) => {
  if (v == null) return <Text type="secondary">—</Text>;
  if (v === 0) return <Tooltip title="Source collection rỗng — không có bản ghi nào ở nguồn">
    <Tag color="orange">0 (rỗng)</Tag>
  </Tooltip>;
  return v.toLocaleString();
},
```

---

## 4. Files thay đổi

| File | Thay đổi | Số dòng ước tính |
|------|----------|-----------------|
| `cdc-cms-web/src/components/ReconPipelineGrid.tsx` | Thêm `fmtCount()`, sửa `overallStatus()`, sửa 2 cột render | ~15 dòng |
| `cdc-cms-web/src/pages/DataIntegrity.tsx` | Sửa render 2 cột Total Source/Dest | ~12 dòng |

**Tổng**: ~27 dòng thay đổi. Không có file mới. Không sửa backend. Không thay đổi API contract.

---

## 5. Verification Plan (Definition of Done — Rule #16)

- G3: `npm run dev` chạy không lỗi compile
- G4: Edge cases: `masterCount=null` → hiện "—" (không bị gắn tag "rỗng"), `masterCount=0` → hiện cyan tag
- G5: Không phá vỡ các status `drift`, `error`, `ok` hiện có (regression)
- G6: Verify UI trên browser — pipeline có count=0 hiện đúng tag và màu cyan
- G8: Ghi bằng chứng vào `06_validation.md`

---

## 6. Cam kết không làm

- ❌ KHÔNG sửa backend / SQL
- ❌ KHÔNG thêm API endpoint mới
- ❌ KHÔNG thay đổi type/interface contract
- ❌ KHÔNG "fix bẩn" (workaround dùng hardcode)
