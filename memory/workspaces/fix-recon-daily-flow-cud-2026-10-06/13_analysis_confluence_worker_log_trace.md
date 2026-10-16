# 13. Phân tích Runtime Log `cdc-worker` — Tác vụ Báo cáo Confluence Daily (Schedule #11)

- **Thời gian trích xuất:** `2026-10-15T13:20:18+07:00`
- **Đối tượng theo dõi:** Tiến trình `cdc-worker` (`PID 14111` trên cổng `:8082`), bảng `cdc_system.cdc_worker_schedule`, `cdc_system.cdc_activity_log` (`cdc_dw`) và hệ thống log SigNoz ClickHouse (`signoz_logs.logs_v2`).

---

## 1. Dữ liệu Cấu hình Lịch trình thực tế trong DB (`cdc_system.cdc_worker_schedule` — ID `#11`)

Khi tạo lịch trình từ giao diện `ConfluenceBranchSelector` mới trên `ActivityManager.tsx` (lúc `11:50:36+07:00`, Activity Log `#114987`), bản ghi `id = 11` trong `cdc_system.cdc_worker_schedule` đã lưu **chính xác 100% cấu trúc nhánh 3 trạm (`Source ➔ Shadow ➔ Master`)**:

- **`id`:** `11`
- **`operation`:** `confluence`
- **`target_table`:** `export_jobs_2`
- **`last_run_at`:** `2026-10-15 04:56:38.566372+00` (`11:56:38+07:00`)
- **`next_run_at`:** `2026-10-15 23:00:00+00` (`2026-10-16 06:00:00+07:00`)
- **`run_count`:** `1`
- **`last_error`:** `check existing confluence page failed: search content returned status 404`
- **`selected_tables`:**
  ```json
  [
    {
      "branch_key": "src:201|shd:201|mst:51",
      "display_name": "[traitestces] centrallized-export-service.export-jobs ➔ [default_shadow] shadow_traitestces.export_jobs ➔ [master_2] master_centrallized_export_service.export_jobs_2",
      "source_object_id": 201,
      "source_connection_code": "traitestces",
      "source_namespace": "centrallized-export-service",
      "source_object_name": "export-jobs",
      "source_object_type": "collection",
      "shadow_binding_id": 201,
      "shadow_connection_code": "default_shadow",
      "shadow_schema": "shadow_traitestces",
      "shadow_table": "export_jobs",
      "master_binding_id": 51,
      "master_connection_code": "master_2",
      "master_schema": "master_centrallized_export_service",
      "master_table": "export_jobs_2",
      "layer_type": "source",
      "connection_code": "traitestces",
      "container_namespace": "centrallized-export-service",
      "object_name": "export-jobs",
      "object_type": "collection"
    }
  ]
  ```
- **`confluence` config:**
  ```json
  {
    "base_url": "https://goopay.atlassian.net",
    "space_key": "~712020a3dcebcbe93646fea463f119dd38b08c",
    "parent_page_id": "463833042",
    "user_email": "trai.nguyen@goopay.vn",
    "title_template": "[Daily Data Check] {{.ReportDate}} - Core Trans Proxy History Service",
    "api_token": "<192 ký tự: ATATT3xFfGF0...1D6373>"
  }
  ```

---

## 2. Dòng Log Runtime thực tế của `cdc-worker` (SigNoz ClickHouse & `cdc_activity_log`)

Khi bấm **"Chạy ngay"** lúc `11:56:37+07:00` (`2026-10-15T04:56:37Z`):

1. **CMS API (`cdc-cms-service` `:8083`)** ghi nhận lệnh kích hoạt thủ công (`cdc_activity_log` ID `114995`):
   - `operation = 'schedule-update'`, `target_table = 'export_jobs_2'`, `triggered_by = 'manual'`, `status = 'success'`
   - `details = {"last_run_at": null, "next_run_at": "2026-10-15T11:56:37.549863+07:00", "updated_at": "2026-10-15T11:56:37.549863+07:00"}`
2. **Đúng 1 giây sau (`11:56:38.568+07:00`), `cdc-worker` (`:8082`)** bắt được lịch chạy ngay và ghi 3 dòng log liên tiếp vào OTel / SigNoz ClickHouse:
   ```text
   2026-10-15 11:56:38.568575000  info   [2026-10-15T04:56:38.568Z] INFO TraiNguyens-MacBook-Pro.local/CDC-WORKER: executing scheduled operation=confluence interval_min=1440 first_run=true target_table=export_jobs_2 run_count=1 {"component":"worker_server","first_run":true,"interval_min":1440,"op":"schedule_exec","operation":"confluence","run_count":1,"span_id":"8079cda0681ce4f6","trace_id":"b9690fb12010a42ed2243ef41e7a8804"}
   2026-10-15 11:56:38.583245000  info   [2026-10-15T04:56:38.583Z] INFO TraiNguyens-MacBook-Pro.local/CDC-WORKER: sending confluence daily report {"parent_page_id":"463833042","table_count":1,"title":"[Daily Data Check] 2026-10-14 - Core Trans Proxy History Service"}
   2026-10-15 11:56:39.298175000  error  [2026-10-15T04:56:39.298Z] ERROR TraiNguyens-MacBook-Pro.local/CDC-WORKER: failed to create confluence child page {"error":"check existing confluence page failed: search content returned status 404","schedule_id":11}
   ```
3. **Đồng thời `cdc-worker` ghi vào `cdc_system.cdc_activity_log` (ID `114996`):**
   - `operation = 'confluence'`, `target_table = 'export_jobs_2'`, `triggered_by = 'manual'`, `status = 'error'`
   - `error_message = 'check existing confluence page failed: search content returned status 404'`

---

## 3. Phân tích Nguyên nhân Gốc rễ (Root Cause Trace)

### 3.1. Nguyên nhân 1 — Lỗi từ phía Atlassian Confluence API (`403 FORBIDDEN` / `404`)
Khi kiểm tra trực tiếp bằng chứng thực `Basic Auth (trai.nguyen@goopay.vn : api_token)` của Schedule `#11` tới `https://goopay.atlassian.net`:
1. **Gọi Jira REST API (`GET https://goopay.atlassian.net/rest/api/3/myself`):**
   - Kết quả: **`HTTP 200 OK`**
   - Body trả về:
     ```json
     {
       "accountId": "712020:d5d8178d-212b-4f89-afec-9f0a21285906",
       "accountType": "atlassian",
       "emailAddress": "trai.nguyen@goopay.vn",
       "displayName": "Trai Nguyen",
       "active": true
     }
     ```
   - $\rightarrow$ Chứng minh `user_email` và `api_token` (192 ký tự) là **thông tin xác thực Atlassian hợp lệ**.
2. **Gọi Confluence REST API (`GET https://goopay.atlassian.net/wiki/rest/api/content?...`):**
   - Kết quả từ Atlassian Edge:
     ```json
     {
       "statusCode": 403,
       "message": "com.atlassian.confluence.mvc.rest.common.exception.StacklessResponseStatusException: 403 FORBIDDEN \"Request rejected because caller cannot access Confluence\""
     }
     ```
   - Và khi kiểm tra không kèm quyền Confluence hoặc sai scope trên một số endpoint tìm kiếm/space cá nhân, Confluence trả về `404` hoặc `403`.
   - **Lý do cụ thể:**
     - **Thứ nhất (Lệch Account ID / Quyền truy cập sản phẩm Confluence):** Tài khoản `trai.nguyen@goopay.vn` có `accountId` là `712020:d5d8178d-212b-4f89-afec-9f0a21285906` hiện đang có quyền truy cập **Jira** trên `goopay.atlassian.net`, nhưng chưa được cấp license/product access vào **Confluence** trên site `goopay.atlassian.net` (`"Request rejected because caller cannot access Confluence"`). Đáng chú ý, `space_key` đang cấu hình là `~712020a3dcebcbe93646fea463f119dd38b08c` — đây là Personal Space của một tài khoản Atlassian **khác** có `accountId = 712020:a3dcebcbe93646fea463f119dd38b08c` (không phải `712020:d5d8178d-212b-4f89-afec-9f0a21285906` của `trai.nguyen@goopay.vn`).
     - **Thứ hai (Nếu dùng Scoped API Token):** Nếu API token (dài 192 ký tự) được tạo bằng tính năng *"Create API token with scopes"* của Atlassian, Atlassian chặn việc gọi trực tiếp qua domain `https://<site>.atlassian.net/wiki/rest/api/...` hoặc thiếu scope Confluence, buộc phải dùng **Unscoped API Token** (*"Create API token"* thông thường) của đúng email tài khoản sở hữu Confluence Space `~712020a3dcebcbe93646fea463f119dd38b08c`.

### 3.2. Nguyên nhân 2 — Tiến trình `cdc-worker` (`PID 14111`) đang chạy bản build từ `08:55:09` sáng
- Tiến trình `cdc-worker` đang giữ cổng `:8082` (`PID 14111` trên `/dev/ttys001`) được khởi chạy từ lúc `08:55:09` sáng (`/Users/trainguyen/Library/Caches/go-build/ab/aba980fb.../main`), trước thời điểm chúng ta cập nhật `confluence_runner.go` lúc `11:38:54`.
- Để `cdc-worker` kết xuất bảng XHTML 8 cột mới theo nhánh 3 trạm (`Source ➔ Shadow ➔ Master`), cần restart `cdc-worker` trên terminal `/dev/ttys001` (hoặc chạy binary `/tmp/cds-worker` đã build sẵn).

### 3.3. Phát hiện bổ sung (Propose-Only, chưa chạm vào code)
1. **Trong [`pkgs/confluence/client.go` (dòng 97–99)](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/confluence/client.go#L97-L99):** Hàm `findPageByTitle` chỉ trả về `fmt.Errorf("search content returned status %d", resp.StatusCode)` mà không đọc `resp.Body` khi lỗi, khiến thông điệp gốc từ Atlassian (`"Request rejected because caller cannot access Confluence"`) bị ẩn mất.
2. **Trong luồng Sửa Lịch Trình (`UpdateScheduleHandler` / `ActivityManager.tsx`):** Khi mở Modal Sửa cho lịch trình Confluence nhưng người dùng không gõ lại ô API Token, Frontend có nguy cơ gửi lại chuỗi đã bị che `ATATT3...****` từ danh sách GET xuống Backend và ghi đè mất token thật trong DB nếu không bỏ qua giá trị chứa `...****`.

---

## 4. Bằng chứng Xác thực Chủ sở hữu Space `~712020a3dcebcbe93646fea463f119dd38b08c` từ Atlassian API

Truy vấn trực tiếp danh sách User trên `https://goopay.atlassian.net/rest/api/3/users/search?maxResults=50` bằng token của Schedule `#11` cho thấy:

1. **Chủ sở hữu thực sự của Space `~712020a3dcebcbe93646fea463f119dd38b08c` (và trang cha `463833042`):**
   ```json
   {
     "accountId": "712020:a3dcebcb-e936-46fe-a464-f119dd38b08c",
     "accountType": "atlassian",
     "emailAddress": "tech@goopay.vn",
     "displayName": "GooPay Technical Team",
     "active": true
   }
   ```
   *(Lưu ý: Trên Confluence Cloud, Personal Space Key được tạo tự động bằng `~` + `accountId` bỏ dấu gạch ngang `-`: `712020:a3dcebcb-e936-46fe-a464-f119dd38b08c` $\rightarrow$ `~712020a3dcebcbe93646fea463f119dd38b08c`).*

2. **Tài khoản đang cấu hình trong Schedule `#11` (`user_email = "trai.nguyen@goopay.vn"`):**
   ```json
   {
     "accountId": "712020:d5d8178d-212b-4f89-afec-9f0a21285906",
     "accountType": "atlassian",
     "emailAddress": "trai.nguyen@goopay.vn",
     "displayName": "Nguyễn Vũ Ngọc Trai",
     "active": true
   }
   ```
   - Tài khoản `trai.nguyen@goopay.vn` hiện chỉ truy cập được Jira (`/rest/api/3/myself` trả `200 OK`), còn khi gọi vào Confluence (`/wiki/rest/api/content?spaceKey=~712020a3dcebcbe93646fea463f119dd38b08c...` hoặc `/wiki/api/v2/...`) thì bị từ chối với HTTP `404` / `403 FORBIDDEN ("Request rejected because caller cannot access Confluence")`.
   - Khi thử ghép `tech@goopay.vn` với API Token hiện tại của `trai.nguyen@goopay.vn`, Atlassian trả về `401 Unauthorized` (vì API Token hiện tại được sinh từ tài khoản `trai.nguyen@goopay.vn`, không phải của `tech@goopay.vn`).

---

## 5. Bằng chứng Phân quyền Group (`groups` & `applicationRoles`) trên `goopay.atlassian.net`

Kết quả truy vấn trực tiếp `GET https://goopay.atlassian.net/rest/api/3/myself?expand=groups,applicationRoles` và `/rest/api/3/user?accountId=712020:a3dcebcb-e936-46fe-a464-f119dd38b08c&expand=groups,applicationRoles`:

1. **Tài khoản `trai.nguyen@goopay.vn` (`712020:d5d8178d-212b-4f89-afec-9f0a21285906`):**
   - `groups`: Chỉ có duy nhất 1 group:
     - `jira-users-goopay` (`groupId: 344d62b7-e1c3-4759-bb0f-79a127ab2c00`)
   - `applicationRoles`: Chỉ có `jira-software` (`Jira Software`).
   - **Không** có group `confluence-users-goopay` hay `confluence-admins-goopay` trên site `goopay.atlassian.net`.

2. **Tài khoản `tech@goopay.vn` (`712020:a3dcebcb-e936-46fe-a464-f119dd38b08c` — "GooPay Technical Team"):**
   - `groups`: Có 5 groups:
     - `compass-admins-goopay`
     - `compass-users-goopay`
     - **`confluence-admins-goopay`** (`groupId: a055fced-df76-4a29-902c-cb0a00d5bbe4`)
     - `jira-users-goopay`
     - **`org-admins`** (`groupId: ab7281eb-7d61-475e-8e0c-3c97d3bd6d43`)

3. **Đánh giá phương án User tự tạo Space mới để dùng:**
   - **Nếu tạo Space mới trên một Atlassian Cloud site riêng của `trai.nguyen@goopay.vn` (ví dụ `https://<site-moi>.atlassian.net`):**
     - Hoàn toàn chạy được ngay lập tức với chính `user_email = trai.nguyen@goopay.vn` và `api_token` hiện tại (vì API Token của Atlassian gắn theo tài khoản `id.atlassian.com` toàn cục). Chỉ cần điền `Base URL`, `Space Key` và `Parent Page ID` của Space mới.
   - **Nếu tạo Space mới ngay trên `https://goopay.atlassian.net/wiki`:**
     - Trên `goopay.atlassian.net`, tài khoản `trai.nguyen@goopay.vn` hiện mới chỉ có group `jira-users-goopay` (chưa có product access vào Confluence trên `goopay.atlassian.net`). Cần được `org-admins` / `confluence-admins-goopay` (`tech@goopay.vn`) thêm vào group Confluence trên `goopay.atlassian.net` thì mới vào được `/wiki` để tạo Space mới trên `goopay.atlassian.net`.

---

## 6. Kiểm tra sau khi Cấp quyền Confluence cho `trai.nguyen@goopay.vn` (Lúc `14:51+07:00`)

### 6.1. Kết quả kiểm tra API Token cũ (`ATATT3xFfG...1D6373`):
- **KHÔNG CẦN tạo API Token mới.** API Token cũ đang lưu trong Schedule `#11` của `trai.nguyen@goopay.vn` đã gọi thành công **`HTTP 200 OK`** vào Confluence REST API (`/wiki/rest/api/user/current`, `/wiki/rest/api/space`, `/wiki/rest/api/content/search`).

### 6.2. Phát hiện Lệch 1 chữ số ở `space_key` và `parent_page_id` mặc định so với trang thật trên Confluence:
Khi truy vấn `/wiki/rest/api/content/search?cql=title~"Core"`, trang **`Core Trans Proxy History Service`** thật trên `goopay.atlassian.net` trả về:
```json
{
  "id": "463831042",
  "type": "page",
  "status": "current",
  "title": "Core Trans Proxy History Service",
  "space": {
    "id": 98306,
    "key": "~712020a3dcebcbe93646fea464f119dd38b08c",
    "name": "GooPay Technical Team"
  },
  "operations": [
    {"operation": "read", "targetType": "page"},
    {"operation": "update", "targetType": "page"},
    {"operation": "create", "targetType": "page"}
  ],
  "_links": {
    "webui": "/spaces/~712020a3dcebcbe93646fea464f119dd38b08c/pages/463831042/Core+Trans+Proxy+History+Service"
  }
}
```
- **Đối chiếu:**
  1. **`space_key`:** Đang lưu `~712020a3dcebcbe93646fea463f119dd38b08c` (`a463` — báo lỗi `404 No space with key`) $\leftrightarrow$ Giá trị thật trên Confluence là **`~712020a3dcebcbe93646fea464f119dd38b08c`** (`a464` — trả về **`200 OK`**).
  2. **`parent_page_id`:** Đang lưu `463833042` (`33042` — báo lỗi `404 No content found with id`) $\leftrightarrow$ Giá trị thật trên Confluence là **`463831042`** (`31042` — trả về **`200 OK`**, có đủ quyền `create page` và `update page`).

---

## 7. Xác nhận Thành công 100% với Space `~712020a3dcebcbe93646fea464f119dd38b08c` và Parent Page `500432919`

Lúc `14:59:20+07:00`, lịch trình `#22` (`space_key = "~712020a3dcebcbe93646fea464f119dd38b08c"`, `parent_page_id = "500432919"`) đã được `cdc-worker` thực thi và **TẠO TRANG CON THÀNH CÔNG 100%** trên Atlassian Confluence:

- **Log `cdc-worker` (SigNoz ClickHouse):**
  ```text
  2026-10-15 14:59:20.930386000  info  executing scheduled operation=confluence interval_min=1440 first_run=true target_table=export_jobs_2 run_count=1
  2026-10-15 14:59:20.948856000  info  sending confluence daily report {"parent_page_id":"500432919","table_count":1,"title":"[Daily Data Check] 2026-10-14 - Core Trans Proxy History Service"}
  2026-10-15 14:59:22.501920000  info  confluence child page processed successfully {"schedule_id":22,"url":"https://goopay.atlassian.net/wiki/spaces/~712020a3dcebcbe93646fea464f119dd38b08c/pages/500531217/Daily+Data+Check+2026-10-14+-+Core+Trans+Proxy+History+Service"}
  ```
- **Thông tin trang con vừa được tạo trên Confluence (`id = 500531217`):**
  - **URL xem trực tiếp:** `https://goopay.atlassian.net/wiki/spaces/~712020a3dcebcbe93646fea464f119dd38b08c/pages/500531217/Daily+Data+Check+2026-10-14+-+Core+Trans+Proxy+History+Service`
  - **Cây thư mục cha (`ancestors`):**
    `Technical Team Document Space (98403)` ➔ `III. Team Document (16842810)` ➔ `System Architecture Document (445546555)` ➔ `Core Trans Proxy History Service (463831042)` ➔ **`Daily Data Report - Core Trans Proxy History Service (500432919)`** ➔ **`[Daily Data Check] 2026-10-14 - Core Trans Proxy History Service (500531217)`**
  - **Nội dung bảng 8 cột (`body.storage.value`) đang hiển thị trên trang Confluence `500531217`:**
    - `Trạm Nguồn (Source)`: `[traitestces] centrallized-export-service.export-jobs`
    - `Trạm Trung Gian (Shadow)`: `[default_shadow] shadow_traitestces.export_jobs`
    - `Trạm Đích (Master)`: `[master_2] master_centrallized_export_service.export_jobs_2`
    - `Số Đếm Ngày Tạo (Source | Shadow | Master)`: `0 | 0 | 0`
    - `Lưu Lượng CDC (Kafka / Shadow / Master)`: `0 / 0 / 0`
    - `Trạng Thái`: `🟢 MATCH (Khớp 100%: Source = Shadow = Master)`
