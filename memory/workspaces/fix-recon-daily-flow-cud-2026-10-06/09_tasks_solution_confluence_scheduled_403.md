# Phân tích Gốc rễ & Giải pháp: Lỗi `search content returned status 403` trên Confluence (`cdc_transaction_history`)

## 1. Bối cảnh & Triệu chứng
- Trên môi trường Testing (`10.200.186.80:5432` — `cdc_cms_database`), bảng `cdc_system.cdc_activity_log` ghi nhận dòng lỗi:
  - `ID = 386489` | `operation = confluence` | `target_table = cdc_transaction_history` | `status = error` | `triggered_by = scheduler` | `started_at = 2026-10-15 23:00:02.812979+00` (`06:00:02` sáng 16/10/2026 giờ VN)
  - `error_message`: `check existing confluence page failed: search content returned status 403`

---

## 2. Bằng chứng Thực tế từ Atlassian API & Testing DB (Zero-Guessing)

### 2.1. Kiểm tra trực tiếp bằng chứng từ Atlassian API (`goopay.atlassian.net`) lúc `09:17` – `09:24` sáng 16/10/2026
Khi dùng đúng `user_email = "trai.nguyen@goopay.vn"` và `api_token` (192 ký tự) đang lưu trong Schedule `#16` (`cdc_transaction_history`) gọi trực tiếp tới Atlassian Cloud:

1. **Gọi `GET https://goopay.atlassian.net/wiki/rest/api/user/current` và `GET /wiki/rest/api/content?...`:**
   - Mã trả về: **`HTTP 403 Forbidden`**
   - Body trả về từ Atlassian Confluence:
     ```json
     {
       "statusCode": 403,
       "message": "com.atlassian.confluence.mvc.rest.common.exception.StacklessResponseStatusException: 403 FORBIDDEN \"Request rejected because caller cannot access Confluence\""
     }
     ```
2. **Gọi `GET https://goopay.atlassian.net/rest/api/3/myself?expand=groups,applicationRoles`:**
   - Mã trả về: **`HTTP 200 OK`**
   - Danh sách `groups` hiện tại của `trai.nguyen@goopay.vn`:
     ```json
     "groups": {
       "size": 1,
       "items": [
         {
           "name": "jira-users-goopay",
           "groupId": "344d62b7-e1c3-4759-bb0f-79a127ab2c00"
         }
       ]
     }
     ```
   - Trong khi đó, danh sách các group Confluence trên `goopay.atlassian.net` (`/rest/api/3/groups/picker`) gồm `cf_user_gp`, `cf_def_GP`, `cf_view`. Tài khoản `trai.nguyen@goopay.vn` hiện **không nằm trong bất kỳ group Confluence nào** (chỉ còn duy nhất group `jira-users-goopay`).

### 2.2. Giải thích diễn biến thời gian (Timeline)
- **Hôm qua 15/10 lúc `14:21`:** Tài khoản `trai.nguyen@goopay.vn` chỉ có group `jira-users-goopay` -> bị lỗi `403 "Request rejected because caller cannot access Confluence"`.
- **Hôm qua 15/10 lúc `14:51` – `17:07`:** Sau khi nhờ Admin cấp quyền, `trai.nguyen@goopay.vn` vào được Confluence (`200 OK`), chạy tay tạo thành công Page `500007065` (trên Testing lúc `16:50`) và Page `500138125` (trên môi trường chính lúc `17:07`).
- **Sáng nay 16/10 lúc `06:00:02`:** Job tự động (`ID=386489`, `triggered_by = scheduler`) trên Testing chạy và bị `403`.
- **Sáng nay 16/10 lúc `08:33:54` – `09:02:55`:** Quyền Confluence hoạt động tạm thời (`200 OK`), tạo được Page `501088257` lúc `08:33:54`.
- **Sáng nay 16/10 từ `09:17:02` đến hiện tại (`09:24:01`):** Quyền sản phẩm Confluence của `trai.nguyen@goopay.vn` trên `goopay.atlassian.net` **lại bị thu hồi/mất group Confluence** (chỉ còn `jira-users-goopay`), nên mọi lệnh gọi vào `/wiki/rest/api/...` ngay lúc này đều bị Atlassian từ chối với lỗi:
  `403 FORBIDDEN "Request rejected because caller cannot access Confluence"`.

---

## 3. Nguyên nhân Gốc rễ (Root Cause)
1. **Về quyền tài khoản trên Atlassian (`goopay.atlassian.net`):**
   - Tài khoản `trai.nguyen@goopay.vn` đang bị mất quyền sản phẩm Confluence (`"Request rejected because caller cannot access Confluence"`), hiện tại trong thông tin user chỉ còn duy nhất 1 group `"jira-users-goopay"`, không có mặt trong các group Confluence (`cf_user_gp` / `cf_def_GP` / `cf_view`).
   - Nguyên nhân quyền lúc có lúc mất: Do trên Atlassian Organization (`admin.atlassian.com`) có cơ chế đồng bộ Directory/IdP định kỳ (hoặc Admin/script quản lý seat license Confluence gỡ group khi đồng bộ lại), hoặc quyền chưa được gắn cố định vào group Confluence chuẩn (`cf_user_gp` / `cf_def_GP`).
2. **Về phía Code (`centralized-data-service/pkgs/confluence/client.go`):**
   - Hàm [`findPageByTitle`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/confluence/client.go#L97-L99) không đọc `resp.Body` khi `resp.StatusCode != http.StatusOK`, khiến thông báo lỗi thật sự của Atlassian (`"Request rejected because caller cannot access Confluence"`) bị nuốt mất, chỉ hiển thị `search content returned status 403` trên UI.
   - Chưa có cơ chế Retry/Backoff khi gặp lỗi tạm thời.

---

## 4. Giải pháp Đề xuất
1. **Giải pháp Cấu hình / Quyền Atlassian (Xử lý ngay không cần chờ sửa code):**
   - **Phương án A (Ổn định nhất cho Job chạy tự động):** Sử dụng `user_email` + `api_token` của tài khoản cố định sở hữu Space `GooPay Technical Team` (ví dụ `tech@goopay.vn` — chủ sở hữu Space `~712020a3dcebcbe93646fea464f119dd38b08c`) hoặc Service Account có license Confluence cố định.
   - **Phương án B (Nếu tiếp tục dùng `trai.nguyen@goopay.vn`):** Nhờ Admin Atlassian kiểm tra mục **Product Access / Groups** của `trai.nguyen@goopay.vn` trên `admin.atlassian.com`, đảm bảo tài khoản được thêm cố định vào group Confluence (`cf_user_gp` hoặc `cf_def_GP`) và không bị Directory Sync / License rule tự động gỡ ra.
2. **Giải pháp Code trong [`centralized-data-service/pkgs/confluence/client.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/confluence/client.go) (Chờ User `APPROVE`):**
   - Đọc đầy đủ `resp.Body` trong `findPageByTitle` khi `StatusCode != 200` để ghi rõ message từ Atlassian vào `last_error` và `cdc_activity_log`.
   - Bổ sung Retry với Exponential Backoff + Jitter và chuẩn hóa HTTP Headers.
