# Hồ Sơ Giải Pháp Kỹ Thuật - Tùy chỉnh Thời gian Đối soát

Thiết kế chi tiết thay đổi trong component frontend.

## 1. Frontend (ConfirmDestructiveModal.tsx)

Sử dụng helper để lấy thời điểm `date to` lùi 5 phút và làm tròn giây về `00s`:
```typescript
const getRoundedEndTime = () => {
  return dayjs().subtract(5, 'minute').second(0).millisecond(0);
};
```

Cập nhật các hàm xử lý:
- **`handleCheckModeChange`**:
  ```typescript
  const handleCheckModeChange = (mode: '2h' | '7d' | 'custom' | 'deep') => {
    setCheckMode(mode);
    const endTime = getRoundedEndTime();
    if (mode === '2h') {
      setCustomRange([endTime.subtract(2, 'hour'), endTime]);
    } else if (mode === '7d') {
      setCustomRange([endTime.subtract(7, 'day'), endTime]);
    } else if (mode === 'custom') {
      setCustomRange([endTime.subtract(30, 'day'), endTime]);
    } else if (mode === 'deep') {
      setCustomRange(null);
    }
  };
  ```

- **`useEffect` (on open)**:
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

- **`handleOk`**:
  ```typescript
  const handleOk = async () => {
    if (!isFormValid || isBusy) return;
    try {
      setSubmitting(true);
      if (isManualRecon) {
        let startMs: number | null = null;
        let endMs: number | null = null;
        let typeRecon = 'hash_window';

        const endTime = getRoundedEndTime();
        if (checkMode === '2h') {
          startMs = endTime.subtract(2, 'hour').valueOf();
          endMs = endTime.valueOf();
        } else if (checkMode === '7d') {
          startMs = endTime.subtract(7, 'day').valueOf();
          endMs = endTime.valueOf();
        } else if (checkMode === 'custom') {
          startMs = customRange?.[0] ? customRange[0].valueOf() : null;
          endMs = customRange?.[1] ? customRange[1].valueOf() : null;
        } else if (checkMode === 'deep') {
          typeRecon = 'deep_check';
        }
  ```
