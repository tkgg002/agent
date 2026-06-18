# Plan: System Health Reconciliation Removal

## 1. Mục tiêu
- Loại bỏ UI hiển thị "Đối soát dữ liệu" khỏi trang SystemHealth.
- Dọn dẹp component helper và interfaces liên quan không dùng tới nữa để tối ưu và làm gọn file.

## 2. Các tệp cần chỉnh sửa
- [MODIFY] [SystemHealth.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/SystemHealth.tsx)

## 3. Các bước thực hiện chi tiết
1. Xóa component `ReconciliationBody` khỏi file `SystemHealth.tsx` (dòng 414-462).
2. Xóa interface `ReconRow` (dòng 405-412).
3. Xóa đoạn JSX hiển thị HealthSection "Đối soát dữ liệu" (dòng 669-671):
   ```tsx
   <HealthSection title="Đối soát dữ liệu" section={sections.reconciliation}>
     {(data) => <ReconciliationBody data={data} />}
   </HealthSection>
   ```

## 4. Kế hoạch kiểm thử (Verification Plan)
- Chạy npm run build trên frontend `cdc-cms-web` để đảm bảo không còn lỗi cú pháp hay thiếu import.
- Kiểm tra thủ công giao diện SystemHealth xem bảng "Đối soát dữ liệu" đã biến mất chưa.
