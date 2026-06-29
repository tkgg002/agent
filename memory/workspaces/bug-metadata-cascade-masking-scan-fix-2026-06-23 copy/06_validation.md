# Validation Plan

## Test Cases

### TC 1: Toggle Active Source
1. Vào UI Source Registry.
2. Click toggle Active cho 1 source.
3. Kiểm tra DB: bảng `cdc_system.shadow_binding` tương ứng có thay đổi `is_active` không. Kỳ vọng: KHÔNG thay đổi.

### TC 2: Snapshot Monitor Shadow Name
1. Chạy snapshot cho 1 binding cụ thể.
2. Vào màn hình `/snapshot-monitor`.
3. Kiểm tra cột Shadow của dòng snapshot đó. Kỳ vọng: Hiển thị đúng tên bảng shadow của binding đó.

### TC 3: Sensitive Masking Strategy
1. Cấu hình Masking cho một cột nhạy cảm trên một shadow binding.
2. Chạy snapshot hoặc upstream.
3. Kiểm tra data trong table đích. Kỳ vọng: Cột nhạy cảm được mask đúng định dạng.

### TC 4: Scan Fields empty table polling stop
1. Đảm bảo shadow table rỗng.
2. Bấm "Scan fields" trên UI.
3. Kỳ vọng: Nút quay loading một lát, sau đó dừng lại và báo lỗi rõ ràng. Không quay vô hạn.
