# Hồ sơ Giải pháp Kỹ thuật: Tối ưu hóa UI Đối Soát (Reconciliation UI Refactor Solution)

Hồ sơ này mô tả chi tiết các phần code cần thay đổi cho hai file frontend trong `cdc-cms-web`.

---

## 1. File: `ConfirmDestructiveModal.tsx`
**Đường dẫn:** `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ConfirmDestructiveModal.tsx`

### Khối thay đổi 1: Mặc định `checkMode` là `'2h'`
**Target Content (dòng 53):**
```typescript
  const [checkMode, setCheckMode] = useState<'2h' | '7d' | 'custom' | 'deep'>('7d');
```
**Replacement Content:**
```typescript
  const [checkMode, setCheckMode] = useState<'2h' | '7d' | 'custom' | 'deep'>('2h');
```

### Khối thay đổi 2: Khởi tạo state trong `useEffect` khi modal mở
**Target Content (dòng 71-80):**
```typescript
  useEffect(() => {
    if (open) {
      setReason('');
      setSubmitting(false);
      setCheckMode('7d');
      const endTime = getRoundedEndTime();
      setCustomRange([endTime.subtract(7, 'day'), endTime]);
      setSegment(initialSegment || '');
    }
  }, [open, initialSegment]);
```
**Replacement Content:**
```typescript
  useEffect(() => {
    if (open) {
      setReason('');
      setSubmitting(false);
      setCheckMode('2h');
      const endTime = getRoundedEndTime();
      setCustomRange([endTime.subtract(2, 'hour'), endTime]);
      setSegment(initialSegment || '');
    }
  }, [open, initialSegment]);
```

### Khối thay đổi 3: Ẩn "Chọn chặng đối soát" bằng CSS
**Target Content (dòng 168-178):**
```typescript
          {/* Chọn chặng đối soát */}
          <div style={{ marginBottom: 16 }}>
            <div style={{ marginBottom: 6 }}><Text strong>Chặng đối soát (Segment):</Text></div>
            <Radio.Group value={segment} onChange={(e) => setSegment(e.target.value)}>
              <Space direction="vertical">
                <Radio value="">Cả 2 chặng (A & B)</Radio>
                <Radio value="source_shadow">Chặng A (Source ➔ Shadow)</Radio>
                <Radio value="shadow_master">Chặng B (Shadow ➔ Master)</Radio>
              </Space>
            </Radio.Group>
          </div>
```
**Replacement Content:**
```typescript
          {/* Chọn chặng đối soát */}
          <div style={{ marginBottom: 16, display: 'none' }}>
            <div style={{ marginBottom: 6 }}><Text strong>Chặng đối soát (Segment):</Text></div>
            <Radio.Group value={segment} onChange={(e) => setSegment(e.target.value)}>
              <Space direction="vertical">
                <Radio value="">Cả 2 chặng (A & B)</Radio>
                <Radio value="source_shadow">Chặng A (Source ➔ Shadow)</Radio>
                <Radio value="shadow_master">Chặng B (Shadow ➔ Master)</Radio>
              </Space>
            </Radio.Group>
          </div>
```

### Khối thay đổi 4: Ẩn Deep Check bằng CSS
**Target Content (dòng 203-208):**
```typescript
                <Radio value="deep">
                  <Text strong>Deep Check (Quét toàn collection)</Text>
                  <div style={{ fontSize: 12, color: '#8c8c8c', marginLeft: 24 }}>
                    So khớp chi tiết từng thuộc tính trên toàn bộ dữ liệu.
                  </div>
                </Radio>
```
**Replacement Content:**
```typescript
                <Radio value="deep" style={{ display: 'none' }}>
                  <Text strong>Deep Check (Quét toàn collection)</Text>
                  <div style={{ fontSize: 12, color: '#8c8c8c', marginLeft: 24 }}>
                    So khớp chi tiết từng thuộc tính trên toàn bộ dữ liệu.
                  </div>
                </Radio>
```

---

## 2. File: `ExecuteHealModal.tsx`
**Đường dẫn:** `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ExecuteHealModal.tsx`

### Khối thay đổi 1: Lọc `reports` và `healedReports` theo `segment` prop
**Target Content (dòng 59-63):**
```typescript
  const reports = (data?.data || []).filter((r: any) => !isReportFullyHealed(r));
  const totalUnhealed = reports.length;
  const healedReports = (historyData?.data || []).filter(
    (r: any) => r.check_type !== 'smoke' && r.status !== 'ok' && r.status !== 'error' && isReportFullyHealed(r)
  );
```
**Replacement Content:**
```typescript
  const rawReports = (data?.data || []).filter((r: any) => !isReportFullyHealed(r));
  const reports = rawReports.filter((r: any) => {
    if (segment === 'source_shadow') {
      return r.segment === 'source_shadow' || !r.segment;
    }
    if (segment === 'shadow_master') {
      return r.segment === 'shadow_master';
    }
    return true;
  });
  const totalUnhealed = reports.length;

  const rawHealedReports = (historyData?.data || []).filter(
    (r: any) => r.check_type !== 'smoke' && r.status !== 'ok' && r.status !== 'error' && isReportFullyHealed(r)
  );
  const healedReports = rawHealedReports.filter((r: any) => {
    if (segment === 'source_shadow') {
      return r.segment === 'source_shadow' || !r.segment;
    }
    if (segment === 'shadow_master') {
      return r.segment === 'shadow_master';
    }
    return true;
  });
```
