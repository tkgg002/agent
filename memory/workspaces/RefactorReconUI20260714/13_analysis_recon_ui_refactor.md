# Phân tích Kỹ thuật: Tối ưu hóa UI Đối Soát (Reconciliation UI Analysis)

## 1. Phân tích `ConfirmDestructiveModal.tsx`
Trong `ConfirmDestructiveModal.tsx`, chúng ta có:
- Cờ `isManualRecon` xác định modal hiển thị giao diện đối soát thủ công (có chọn chế độ, chặng, khoảng thời gian).
- Chế độ quét `checkMode` có các giá trị: `2h` (Hot mode), `7d` (Smoke), `custom` (Custom range), `deep` (Deep check).
- State:
  - `checkMode` mặc định: `'7d'` -> Cần sửa thành `'2h'`.
  - `useEffect` khi modal mở (`open` từ false -> true):
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
    -> Cần cập nhật `checkMode` thành `'2h'` và khoảng thời gian custom tương ứng lùi 2 giờ (`endTime.subtract(2, 'hour')`).
- Deep Check Radio option:
  ```tsx
  <Radio value="deep">
    <Text strong>Deep Check (Quét toàn collection)</Text>
    ...
  </Radio>
  ```
  -> Sẽ thêm `style={{ display: 'none' }}` để ẩn khỏi giao diện.
- Chọn chặng đối soát (Segment Selector):
  ```tsx
  {/* Chọn chặng đối soát */}
  <div style={{ marginBottom: 16 }}>
    <div style={{ marginBottom: 6 }}><Text strong>Chặng đối soát (Segment):</Text></div>
    <Radio.Group value={segment} onChange={(e) => setSegment(e.target.value)}>
      ...
    </Radio.Group>
  </div>
  ```
  -> Sẽ ẩn phần này bằng cách không render khi `isManualRecon` là true (hoặc comment out hoàn toàn). Do `segment` đã được khởi tạo bằng `initialSegment` trong `useEffect`, modal vẫn gửi đúng giá trị chặng ban đầu lên Backend.

## 2. Phân tích `ExecuteHealModal.tsx`
Modal chữa lành nhận prop `segment?: string` từ component cha (`DataIntegrity.tsx`).
- Mặc định, hook `useUnhealedReports` và `useTableHistory` lấy toàn bộ reports của bảng mà không phân biệt chặng.
- Dữ liệu trả về được xử lý thông qua:
  ```typescript
  const reports = (data?.data || []).filter((r: any) => !isReportFullyHealed(r));
  const totalUnhealed = reports.length;
  const healedReports = (historyData?.data || []).filter(
    (r: any) => r.check_type !== 'smoke' && r.status !== 'ok' && r.status !== 'error' && isReportFullyHealed(r)
  );
  ```
- Lọc theo `segment`:
  - Chặng A: `segment === 'source_shadow'`. Lọc `r.segment === 'source_shadow' || !r.segment` (do các bản ghi cũ của chặng A có thể có `segment` null/rỗng).
  - Chặng B: `segment === 'shadow_master'`. Lọc `r.segment === 'shadow_master'`.
  - Áp dụng logic lọc này vào cả `reports` và `healedReports` để đồng bộ hiển thị cả tab "Phiên chưa xử lý" và "Phiên đã xử lý".

## 3. Rủi ro & Giải pháp phòng ngừa
- **Rủi ro:** Khi ẩn Segment Selector, nếu `initialSegment` bị thiếu hoặc rỗng, request gửi lên Backend có thể không có chặng (mặc định đối soát cả 2 chặng).
- **Giải pháp:** Trong `ConfirmDestructiveModal.tsx`, modal mở từ button chặng nào sẽ luôn nhận đúng chặng của row đó. Đã xác nhận `initialSegment` được truyền chính xác từ component cha trong `DataIntegrity.tsx`.
