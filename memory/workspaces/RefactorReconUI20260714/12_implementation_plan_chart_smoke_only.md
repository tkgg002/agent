# Kế hoạch Triển khai: Chỉ hiển thị Smoke Check trên Biểu đồ (Reconciliation Chart Smoke Only Plan)

Kế hoạch này hướng dẫn cách sửa đổi biểu đồ trong `ReconPipelineGrid.tsx` chỉ vẽ dữ liệu Smoke Check.

---

## 1. Thay đổi đề xuất (Proposed Changes)

### Component: `ReconPipelineGrid.tsx`
Đường dẫn: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx`

#### Thay đổi 1: Lọc `chartData`
Lọc dòng dữ liệu lịch sử đối soát từ `history?.data` sao cho chỉ giữ lại các bản ghi có loại kiểm tra (`check_type`) là `'smoke'` hoặc `'segment_b_smoke'`.
```typescript
  const chartData = useMemo(() => {
    const smokeRows = (history?.data || []).filter(
      (r) => r.check_type === 'smoke' || r.check_type === 'segment_b_smoke'
    );
    const rows = smokeRows.slice().reverse();
    return rows.map((r) => ({
      time: new Date(r.checked_at).toLocaleTimeString('vi-VN', { hour: '2-digit', minute: '2-digit' }),
      [r.segment === 'shadow_master' ? 'shadow' : 'source']: r.source_count,
      [r.segment === 'shadow_master' ? 'master' : 'shadow']: r.dest_count,
    }));
  }, [history]);
```

#### Thay đổi 2: Lọc `yDomain`
Cập nhật `yDomain` để cũng chỉ tính toán min/max dựa trên các phiên Smoke Check:
```typescript
  const yDomain = useMemo(() => {
    const smokeRows = (history?.data || []).filter(
      (r) => r.check_type === 'smoke' || r.check_type === 'segment_b_smoke'
    );
    if (smokeRows.length === 0) {
      return ['auto', 'auto'];
    }
    let min = Infinity;
    let max = -Infinity;
    smokeRows.forEach((r) => {
      const src = r.source_count;
      const dest = r.dest_count;
      if (src !== undefined && src !== null) {
        if (src < min) min = src;
        if (src > max) max = src;
      }
      if (dest !== undefined && dest !== null) {
        if (dest < min) min = dest;
        if (dest > max) max = dest;
      }
    });

    if (min === Infinity || max === -Infinity) {
      return ['auto', 'auto'];
    }

    const range = max - min;
    const padding = range === 0 ? 5 : Math.max(1, Math.ceil(range * 0.1));
    return [Math.max(0, Math.floor(min - padding)), Math.ceil(max + padding)];
  }, [history]);
```

---

## 2. Kế hoạch Kiểm tra (Verification Plan)
- Chạy lệnh `npm run build` trong `cdc-cms-web` để đảm bảo code biên dịch thành công 100%.
- Kiểm tra quy trình linter bằng `verify_governance.py`.
