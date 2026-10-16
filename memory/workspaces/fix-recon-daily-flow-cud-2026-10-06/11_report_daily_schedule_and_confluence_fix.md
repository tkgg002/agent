# 📑 BÁO CÁO TỔNG KẾT BẢN VÁ SCHEDULER & CONFLUENCE DUAL-MODE
**Workspace:** `fix-recon-daily-flow-cud-2026-10-06`  
**Ngày thực hiện:** 2026-10-13  
**Người thực hiện:** Muscle (Chief Engineer)  
**Tiêu chuẩn chất lượng:** Hiến pháp GEMINI.md (Rule #0, Rule #4, Rule #5, Rule #12, Rule #14 Gate G1–G8)

---

## I. TỔNG QUAN CÁC THAY ĐỔI THEO MỆNH LỆNH CỦA USER

### 1. Vấn đề 1: Lỗi "Lần chạy kế" bị nhảy `+ 1440` khi bấm "Chạy ngay"
- **Hiện tượng cũ:** Khi người dùng bấm nút "Chạy ngay" trên trang `/activity-manager`, cột "Lần chạy kế" không hiện mốc giờ định kỳ tiếp theo (ví dụ `00:05` ngày mai hoặc `06:00` ngày mai) mà lại cộng thêm 1440 phút tính từ thời điểm bấm (ví dụ bấm lúc 14:50 chiều thì lần chạy kế hiện 14:50 chiều mai, sai lệch hoàn toàn).
- **Gốc rễ:** Trong `server_scheduler.go`, khi job có `interval_minutes = 1440` nhưng chưa được bóc tách mode rõ ràng, nhánh fallback rơi vào `now.Add(intervalDur)`, lấy `now + 1440m`.
- **Giải pháp:** Sửa hàm `calculateNextRun`:
  - Đối với các job có `run_at_time` (Mode 1 - Daily Fixed Time):
    - Khi nhận lệnh "Chạy ngay" (`NextRunAt <= now`): Nếu hiện tại chưa tới mốc giờ hôm nay $\rightarrow$ `nextRun = targetToday`; nếu hiện tại đã qua mốc giờ hôm nay $\rightarrow$ `nextRun = targetToday.AddDate(0, 0, 1)` (mốc giờ đó của ngày mai).
    - Lần chạy kế tiếp luôn luôn là mốc giờ chuẩn (`00:05:00` hoặc `06:00:00`), triệt tiêu hoàn toàn lỗi `+ 1440`.

### 2. Vấn đề 2: Hỗ trợ 2 cơ chế lịch trình độc lập (Mốc giờ vs Chu kỳ phút)
- **Yêu cầu từ User:** "Thời điểm bắn báo cáo hằng ngày (HH:MM) cái này cho set up 2 loại đó là chọn mốc và chạy lại sau bao nhiêu phút."
- **Giải pháp:**
  - **Mode 1 - ⏰ Mốc giờ hằng ngày (HH:mm):** Cho phép nhập mốc giờ cố định (mặc định gợi ý `00:05` cho đếm nguồn, `06:00` cho báo cáo Confluence). Lưu `run_at_time = "HH:mm"`, `interval_minutes = 1440`.
  - **Mode 2 - 🔄 Chạy lại sau bao nhiêu phút (phút):** Cho phép nhập số phút lặp lại (ví dụ: mỗi 5, 15, 30, 60 phút...). Lưu `run_at_time = nil`, `interval_minutes = X`.
  - Áp dụng trên cả Modal Tạo mới và Modal Chỉnh sửa của `ActivityManager.tsx` cho cả tác vụ Confluence và Source Count.

### 3. Vấn đề 3: Vá các lỗ hổng kỹ thuật từ 2 Báo cáo Audit
- **Bảo toàn Token Confluence (Token Masking Write-Back Protection):** Khi chỉnh sửa lịch trình Confluence trên UI/Backend, nếu `api_token` để trống hoặc chứa chuỗi mask `****`, backend tự động giữ nguyên token thật cũ trong CSDL, không ghi đè làm mất token.
- **Xử lý TargetTable chuỗi rỗng:** Gán thành `nil` để lưu `NULL` trong CSDL, tránh lỗi so khớp chuỗi rỗng.
- **Khắc phục Confluence API Client:** Không nuốt lỗi trong `findPageByTitle` để tránh lỗi 400 Duplicate Title, chỉ thêm key `"ancestors"` khi `ParentPageID != ""`.
- **Khắc phục Confluence Runner Query:** Loại bỏ filter cứng `layer_type == 'master'` khi query `daily_source_counts`, fail-fast khi không có bảng nào được cấu hình, làm rõ trạng thái `Master >= Source` khi có updates.

---

## II. DANH SÁCH TẬP TIN ĐÃ THAY ĐỔI & CHI TIẾT DÒNG CODE

| Repository | Đường dẫn tập tin | Số dòng thay đổi | Tóm tắt thay đổi |
| :--- | :--- | :---: | :--- |
| **centralized-data-service** | `internal/server/server_scheduler.go` | ~45 dòng | Viết lại `calculateNextRun` chuẩn cho 2 mode, xử lý Run Now tính mốc ngày mai, loại bỏ ép cứng Daily nếu Interval < 1440. |
| **centralized-data-service** | `internal/server/server_scheduler_test.go` | +115 dòng (mới) | Viết 4 unit test tự động bao quát kịch bản Run Now, Daily Fixed Time và Interval Preserved. |
| **centralized-data-service** | `pkgs/confluence/client.go` | ~25 dòng | Check `err != nil` khi tìm page theo title, chỉ add `"ancestors"` khi `ParentPageID != ""`. |
| **centralized-data-service** | `internal/service/confluence/confluence_runner.go` | ~30 dòng | Query ưu tiên `layer_type = 'source'`, fail-fast khi `len(tables) == 0`, escape text XHTML. |
| **cdc-cms-service** | `internal/app/commands/scheduler/update_schedule.go` | ~35 dòng | Token Masking Write-Back Protection, `target_table == ""` $\rightarrow$ `nil`. |
| **cdc-cms-service** | `internal/app/commands/scheduler/delete_worker_schedule.go` | ~10 dòng | Thay `fmt.Sprintf` bằng `json.Marshal`, dọn import `fmt`. |
| **cdc-cms-web** | `src/pages/ActivityManager.tsx` | ~120 dòng | Bổ sung Radio Group 2 mode lịch trình cho Modal Tạo mới & Modal Sửa (Confluence & Source Count); sửa map `selectedTables` lấy `source_connection_code` và `source_type`; hiển thị cột Lịch chạy trực quan. |

---

## III. BẰNG CHỨNG KIỂM THỬ THỰC TẾ (VERIFICATION EVIDENCE)

1. **`cdc-cms-web` (Frontend):**
   - Lệnh: `npm run build` (`tsc -b && vite build`)
   - Kết quả: **PASS (Exit Code 0)**, toàn bộ các chunk bundle được sinh ra đầy đủ, không còn lỗi TypeScript typing.

2. **`cdc-cms-service` (Backend):**
   - Lệnh: `go test -v ./internal/app/commands/scheduler/...`
   - Kết quả: **PASS (Exit Code 0)**
   - Lệnh: `go build -v -o /dev/null ./cmd/server`
   - Kết quả: **PASS (Exit Code 0)**

3. **`centralized-data-service` (Worker Core):**
   - Lệnh: `go test -v -run "TestCalculateNextRun" ./internal/server`
   - Kết quả: **PASS 4/4 bài test (Exit Code 0)**
     - `TestCalculateNextRun_DailyFixedTime_RunNow_AfterTarget`: PASS
     - `TestCalculateNextRun_DailyFixedTime_RunNow_BeforeTarget`: PASS
     - `TestCalculateNextRun_DailyFixedTime_NormalTrigger`: PASS
     - `TestCalculateNextRun_Confluence_Interval_Preserved`: PASS
   - Lệnh: `go test -v ./pkgs/confluence/... ./internal/service/confluence/...`
   - Kết quả: **PASS (Exit Code 0)**
   - Lệnh: `go build -v -o /dev/null ./cmd/worker`
   - Kết quả: **PASS (Exit Code 0)**

---

## IV. BƯỚC KẾ TIẾP ĐỀ XUẤT CÙNG USER TRACE RUNTIME VALUE

Theo đúng Hiến pháp Rule #14 Gate G3, G6 và yêu cầu của User:
> *"Đặc biệt : ko test bằng mockup và Hallucination kết quả, nếu cần phải nói phối hợp với anh trace runtime value."*

Hệ thống đã chuẩn bị sẵn sàng toàn bộ mã nguồn sạch. Các bước phối hợp kiểm tra runtime:
1. User khởi động frontend (`npm run dev`) và truy cập `http://localhost:5173/activity-manager`.
2. Kiểm tra Modal Tạo mới & Chỉnh sửa tác vụ Confluence và Source Count:
   - Chuyển đổi giữa 2 mode "⏰ Mốc giờ hằng ngày (HH:mm)" và "🔄 Chạy lại sau bao nhiêu phút (phút)".
   - Bấm nút "Chạy ngay" trên một schedule hằng ngày $\rightarrow$ Xác nhận cột "Lần chạy kế" hiển thị chính xác mốc giờ tiếp theo (ví dụ `00:05` ngày mai hoặc `06:00` ngày mai), không còn hiện `+ 1440`!
3. Nhập Atlassian API Token thật vào lịch trình Confluence và kích hoạt "Chạy ngay" để quan sát log gửi trang báo cáo đối soát thực tế lên Confluence Cloud của công ty.
