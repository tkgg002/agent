# HỒ SƠ GIẢI PHÁP KỸ THUẬT: KHẮC PHỤC LỖI CRASH REACT HOOKS KHI CHỌN TÁC VỤ CONFLUENCE TRÊN `/activity-manager`

**Ngày lập:** 2026-10-15  
**Workspace:** `fix-recon-daily-flow-cud-2026-10-06`  
**Trạng thái:** `PROPOSED — CHỜ USER APPROVE TRƯỚC KHI SỬA CODE (Tuân thủ Rule #1 & Rule #13)`

---

## 1. HIỆN TƯỢNG & NGUYÊN NHÂN GỐC RỄ (ROOT CAUSE ANALYSIS)

### 1.1. Hiện tượng trên trình duyệt (`http://localhost:5173/activity-manager`)
Khi bấm **Tạo lịch trình** (hoặc **Sửa lịch trình**) và chuyển ô **Tác vụ (Operation)** sang **`Báo cáo Confluence Daily (Confluence Report)`** (`value = "confluence"`), toàn bộ trang `/activity-manager` lập tức bị sập và hiển thị hộp báo lỗi đỏ:
> **Không tải được dữ liệu**  
> `Rendered more hooks than during the previous render.`

### 1.2. Truy vết chính xác từng dòng code gây lỗi
1. **Tại sao hiện chữ `"Không tải được dữ liệu"`?**
   - Không phải do lỗi gọi mạng hay Backend API chết (cả 2 cổng `:8083` và `:8082` đều đang `LISTEN` bình thường).
   - Toàn bộ các route trong ứng dụng được bọc bởi component [`QueryErrorBoundary.tsx`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/QueryErrorBoundary.tsx#L11-L31). Khi có bất kỳ lỗi JavaScript/React nào ném ra trong quá trình render (`getDerivedStateFromError`), `DefaultFallback` (dòng 17) sẽ hiển thị tiêu đề cứng `title="Không tải được dữ liệu"` kèm `error.message` bên dưới (`Rendered more hooks than during the previous render.`).

2. **Tại sao React ném lỗi `Rendered more hooks than during the previous render.`?**
   - Trong file [`ActivityManager.tsx`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/ActivityManager.tsx):
     - **Vị trí lỗi 1 — Dòng 1299 (Modal Tạo lịch trình mới):**
       ```tsx
       {selectedOperation === 'confluence' ? (
         <>
           ...
           {Form.useWatch('schedule_type', form) === 'interval' ? (
       ```
       - Khi mở Modal Tạo mới lần đầu, `selectedOperation` có giá trị mặc định là `'source_count'`. Nhánh `selectedOperation === 'confluence'` **không được thực thi**, nên hook `Form.useWatch('schedule_type', form)` ở dòng 1299 **không được đăng ký** trên Fiber node của `ActivityManager`. (Ở nhánh `'source_count'`, component `<CascadingSourceSelector />` là một React component con riêng biệt nên các hook bên trong nó nằm trên Fiber node của con, không thuộc `ActivityManager`).
       - Ngay khi người dùng chọn **Tác vụ = `confluence`**, `selectedOperation` đổi thành `'confluence'`. Component `ActivityManager` re-render, đi vào nhánh `selectedOperation === 'confluence'` và **gọi thêm 1 React Hook `Form.useWatch('schedule_type', form)` tại dòng 1299**!
       - Số lượng Hooks của `ActivityManager` tăng từ $N$ lên $N+1$ giữa 2 lần render $\rightarrow$ Vi phạm trực diện **Rules of Hooks** của React (không bao giờ được gọi Hook bên trong câu lệnh điều kiện `if / ? :` của JSX) $\rightarrow$ React lập tức ném lỗi `Rendered more hooks than during the previous render.`.
     - **Vị trí lỗi 2 — Dòng 1472 (Modal Chỉnh sửa lịch trình):**
       ```tsx
       {editSelectedOperation === 'confluence' ? (
         <>
           ...
           {Form.useWatch('schedule_type', editForm) === 'interval' ? (
       ```
       - Mắc chính xác cùng một lỗi: gọi `Form.useWatch('schedule_type', editForm)` bên trong nhánh điều kiện `editSelectedOperation === 'confluence'`. Nếu mở Sửa một lịch trình khác trước rồi mở Sửa lịch trình `confluence` (hoặc ngược lại), số lượng Hooks của `ActivityManager` bị thay đổi giữa các lần render và gây crash.
     - **Vị trí lỗi tiềm ẩn 3 — Dòng 493 (Trong `CascadingSourceSelector`):**
       ```tsx
       {Form.useWatch('schedule_type', form) === 'interval' ? (
       ```
       - Gọi trực tiếp `Form.useWatch` ngay giữa khối JSX thay vì khai báo ở đầu hàm component.

---

## 2. GIẢI PHÁP KỸ THUẬT ĐỀ XUẤT (CODE SOLUTION DEMO)

Đưa toàn bộ các lời gọi `Form.useWatch('schedule_type', ...)` lên **tầng cao nhất (Top-level)** của từng Function Component, tuyệt đối không gọi `Form.useWatch` bên trong JSX có điều kiện:

### Thay đổi 1 — Trong `CascadingSourceSelector` ([`ActivityManager.tsx` dòng 122–125 & 493](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/ActivityManager.tsx#L122-L125))
```tsx
// Tại đầu hàm CascadingSourceSelector (dòng 122-126):
  const selectedLayer = Form.useWatch('layer_type', form) || 'source';
  const selectedConn = Form.useWatch('connection_code', form);
  const selectedNamespace = Form.useWatch('container_namespace', form);
  const selectedScheduleType = Form.useWatch('schedule_type', form) || 'daily';

// Tại dòng 493 (thay thế Form.useWatch inline bằng biến đã khai báo):
  {selectedScheduleType === 'interval' ? (
```

### Thay đổi 2 — Khai báo Hook ở đầu hàm `ActivityManager` ([`ActivityManager.tsx` dòng 528–538](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/ActivityManager.tsx#L528-L538))
```tsx
  // Modal Tạo Mới
  const [createVisible, setCreateVisible] = useState(false);
  const [createLoading, setCreateLoading] = useState(false);
  const [form] = Form.useForm();
  const selectedOperation = Form.useWatch('operation', form) || 'source_count';
  const createScheduleType = Form.useWatch('schedule_type', form) || 'daily';

  // Modal Chỉnh Sửa
  const [editVisible, setEditVisible] = useState(false);
  const [editLoading, setEditLoading] = useState(false);
  const [editingSchedule, setEditingSchedule] = useState<Schedule | null>(null);
  const [editForm] = Form.useForm();
  const editSelectedOperation = Form.useWatch('operation', editForm) || 'source_count';
  const editScheduleType = Form.useWatch('schedule_type', editForm) || 'daily';
```

### Thay đổi 3 — Thay thế 2 chỗ gọi `Form.useWatch` có điều kiện tại dòng 1299 và 1472 ([`ActivityManager.tsx`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/ActivityManager.tsx#L1299-L1475))
```tsx
// Tại dòng 1299 (trong nhánh selectedOperation === 'confluence' của Modal Tạo mới):
- {Form.useWatch('schedule_type', form) === 'interval' ? (
+ {createScheduleType === 'interval' ? (

// Tại dòng 1472 (trong nhánh editSelectedOperation === 'confluence' của Modal Chỉnh sửa):
- {Form.useWatch('schedule_type', editForm) === 'interval' ? (
+ {editScheduleType === 'interval' ? (
```
*(Đồng thời khi chuyển đổi `operation` trong Modal Tạo mới sang `'confluence'`, tự động cập nhật `run_at_time` mặc định thành `'06:00'` hoặc `'00:05'` tương ứng).*
