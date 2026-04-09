# Plan
1. [x] Khởi tạo workspace và RC Analysis.
2. [ ] Kiểm tra hàm `getAll()` của `merchantBankAccountModel`. Xem structure trả về format array hay object.
3. [x] Kiểm tra cơ chế mapping trong `merchant-export.pure.ts`. Nếu query trả về array có duplicate merchantId (vd 1 merchant có nhiều bank account), thì hàm map tạo Map sẽ đè data lên nhau theo thứ tự xuất hiện cuối.
4. [x] Khắc phục: Sửa handler / logic mapping để lấy đúng bankAccount chính hoặc group và get bank account phù hợp nhất.
5. [x] Verify lại data.
