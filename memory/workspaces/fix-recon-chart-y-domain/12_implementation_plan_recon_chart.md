# Kế hoạch triển khai: Tối ưu biểu đồ biến động số lượng phiên recon

## 1. Phân tích kỹ thuật & Đề xuất giải pháp
Hiện tại, biểu đồ sử dụng trục Y `<YAxis>` mặc định mà không cấu hình miền dữ liệu (`domain`). Điều này khiến Recharts tự động scale hoặc bắt đầu từ `0`. Khi bảng có hàng triệu record và chênh lệch chỉ 1-2 record, tỉ lệ lệch so với toàn bộ trục là cực kì nhỏ (ví dụ: `2/2.000.000 = 0.0001%`), dẫn tới các đường vẽ trùng khít lên nhau và phẳng lỳ.

Để khắc phục, chúng ta cần phóng to (zoom-in) trục Y vào đúng khoảng dao động thực tế của dữ liệu trong lịch sử phiên đối soát.

### Thuật toán xác định miền trục Y động (Dynamic Domain Calculation):
1. Quét qua lịch sử `history.data` của phiên đối soát để xác định giá trị tối thiểu `min` và tối đa `max` của cả 2 cột `source_count` và `dest_count`.
2. Tính dải dao động (range): `range = max - min`.
3. Tính toán khoảng đệm (padding) hợp lý để biểu đồ không bị sát mép trên/dưới:
   - Nếu `range === 0` (dữ liệu hoàn toàn bằng nhau qua các phiên): đặt `padding = 5` (hoặc `padding = 1` nếu muốn sát hơn).
   - Nếu `range > 0`: đặt `padding = Math.max(1, Math.ceil(range * 0.1))` (lấy 10% dải dao động, tối thiểu là 1 đơn vị).
4. Thiết lập miền trục Y là: `[Math.max(0, min - padding), max + padding]`. (Luôn giới hạn biên dưới `>= 0`).

Giải pháp này hoàn toàn xử lý trong phần logic React (qua `useMemo` tính toán `yDomain`), sau đó truyền trực tiếp mảng số `domain={yDomain}` cho Recharts `<YAxis>`. Điều này tránh các lỗi không tương thích phiên bản của Recharts callback, đồng thời đảm bảo tính toán chính xác và trực quan.

---

## 2. Chi tiết thay đổi code

### [MODIFY] [ReconPipelineGrid.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx)

1. Thêm `yDomain` vào trong component `ReconPipelineGrid` (ngay sau block tính toán `chartData` ở dòng 277):
```typescript
  const yDomain = useMemo(() => {
    if (!history?.data || history.data.length === 0) {
      return ['auto', 'auto'];
    }
    let min = Infinity;
    let max = -Infinity;
    history.data.forEach((r) => {
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

2. Cập nhật thẻ `<YAxis>` ở dòng 463:
```tsx
              <YAxis fontSize={11} width={80} domain={yDomain} tickFormatter={(v: number) => v.toLocaleString()} />
```

---

## 3. Kế hoạch xác minh

### Tự động:
* Chạy build dự án frontend: `npm run build` để đảm bảo không lỗi TypeScript/JSX.

### Thủ công:
* Truy cập trang Data Integrity, mở chi tiết luồng Pipelines có lượng record lớn và có lệch 1-2 record.
* Xác nhận các đường Source, Shadow, Master được vẽ tách biệt, uốn lượn rõ ràng và trục Y đã được zoom tương ứng với vùng dữ liệu thay vì bắt đầu từ 0.
