# Kế hoạch triển khai - Bổ sung khoảng thời gian đối soát

## 1. Thành phần thay đổi
*   **Tệp đích:** `ExecuteHealModal.tsx` (`/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ExecuteHealModal.tsx`)

## 2. Giải pháp kỹ thuật
1.  **Hàm helper `formatTimeRange`:**
    ```typescript
    const formatTimeRange = (startStr?: string | null, endStr?: string | null): string => {
      if (!startStr && !endStr) return '—';
      if (startStr && !endStr) return `Từ ${new Date(startStr).toLocaleString('vi-VN')}`;
      if (!startStr && endStr) return `Đến ${new Date(endStr).toLocaleString('vi-VN')}`;
      
      const start = new Date(startStr!);
      const end = new Date(endStr!);
      
      const startDateStr = start.toLocaleDateString('vi-VN');
      const endDateStr = end.toLocaleDateString('vi-VN');
      
      const startTimeStr = start.toLocaleTimeString('vi-VN', { hour: '2-digit', minute: '2-digit' });
      const endTimeStr = end.toLocaleTimeString('vi-VN', { hour: '2-digit', minute: '2-digit' });
      
      if (startDateStr === endDateStr) {
        return `${startTimeStr} - ${endTimeStr} (${startDateStr})`;
      }
      return `${startTimeStr} ${startDateStr} - ${endTimeStr} ${endDateStr}`;
    };
    ```
2.  **Cập nhật `reportColumns`:**
    Thêm cột "Khoảng thời gian" vào mảng `reportColumns`:
    ```typescript
    {
      title: 'Khoảng thời gian',
      key: 'time_range',
      width: 220,
      render: (record: any) => formatTimeRange(record.recon_start_time, record.recon_end_time),
    }
    ```

## 3. Kế hoạch kiểm thử
*   Chạy `npx tsc --noEmit` ở Frontend để đảm bảo kiểu dữ liệu an toàn.
*   Chạy linter quy trình: `python3 agent/tooling/verify_governance.py`.
