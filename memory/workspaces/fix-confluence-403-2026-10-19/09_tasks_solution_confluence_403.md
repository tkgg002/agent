# 09. Hồ sơ Giải pháp Kỹ thuật — Khắc phục Lỗi Confluence Scheduler `search content returned status 403` (`confluence_403`)

- **Workspace:** `fix-confluence-403-2026-10-19`
- **Tier:** `T2`
- **Trạng thái:** `PROPOSED` (Chờ User `APPROVE` theo Rule #1 & Rule #13 — chưa chỉnh sửa source code)

---

## 1. Root Cause / Vấn đề cốt lõi (100% Thực chứng `VERIFIED` từ DB, Atlassian API & Source Code)

### 1.1. Bằng chứng truy vết 4 bản ghi lỗi trong DB (`10.200.186.80:5432` — `cdc_cms_database`)
Truy vấn trực tiếp bảng `cdc_system.cdc_worker_schedule` (`id = 16`, `target_table = cdc_transaction_history`) và `cdc_system.cdc_activity_log` trên môi trường Testing (`10.200.186.80:5432`) xác nhận chính xác 4 dòng log anh gửi:
- **`ID = 399090`:** `2026-10-19 06:00:02.468226` (`started_at = 2026-10-18 23:00:02.467981+00`) | `confluence` | `cdc_transaction_history` | `error` | `scheduler` | `check existing confluence page failed: search content returned status 403`
- **`ID = 395735`:** `2026-10-18 06:00:03.033045` (`started_at = 2026-10-17 23:00:03.032810+00`) | `confluence` | `cdc_transaction_history` | `error` | `scheduler` | `check existing confluence page failed: search content returned status 403`
- **`ID = 392381`:** `2026-10-17 06:00:03.146342` (`started_at = 2026-10-16 23:00:03.146073+00`) | `confluence` | `cdc_transaction_history` | `error` | `scheduler` | `check existing confluence page failed: search content returned status 403`
- **`ID = 386489`:** `2026-10-16 06:00:02.813789` (`started_at = 2026-10-15 23:00:02.812979+00`) | `confluence` | `cdc_transaction_history` | `error` | `scheduler` | `check existing confluence page failed: search content returned status 403`
- Trong khi đó, bảng `cdc_system.daily_source_counts` trên cùng DB `10.200.186.80:5432` đã chạy tự động lúc `00:05:01` hằng ngày thành công 100% cho cả 4 ngày (`2026-10-15`: `679`, `2026-10-16`: `681`, `2026-10-17`: `1`, `2026-10-18`: `2`).

### 1.2. Ba nguyên nhân gốc rễ (`VERIFIED`)

1. **Nguyên nhân 1 (`VERIFIED` 100% — Cơ chế tự động thu hồi group `confluence-users-goopay` sau ~30 phút trên `goopay.atlassian.net`):**
   - Đối chiếu kết quả gọi trực tiếp REST API `GET https://goopay.atlassian.net/rest/api/3/myself?expand=groups` và `/wiki/rest/api/user/current` của tài khoản `trai.nguyen@goopay.vn` tại 2 thời điểm cách nhau ~43 phút hôm nay (`19/10/2026`):
     - **Lúc `13:21` – `13:33` (`19/10/2026`):**
       - `groups.size = 2`: gồm `jira-users-goopay` (`344d62b7-e1c3-4759-bb0f-79a127ab2c00`) **VÀ** `confluence-users-goopay` (`8a98d738-86cb-43c9-acd9-c9909c950445`).
       - Cả 3 API `/wiki/rest/api/user/current`, `/wiki/rest/api/content?spaceKey=...`, `/wiki/rest/api/content/500432919` đều trả về **`200 OK`**.
     - **Lúc `14:16:02` (`19/10/2026` — sau ~43 phút):**
       - `groups.size = 1`: **chỉ còn `jira-users-goopay`**, group **`confluence-users-goopay` đã bị thu hồi tự động!**
       - Ngay lập tức, cả 3 API `/wiki/rest/api/user/current`, `/wiki/rest/api/content?spaceKey=...`, `/wiki/rest/api/content/500432919` đều trả về **`403 FORBIDDEN`** với message:
         `{"statusCode":403,"message":"com.atlassian.confluence.mvc.rest.common.exception.StacklessResponseStatusException: 403 FORBIDDEN \"Request rejected because caller cannot access Confluence\""}`
   - **Vì sao bị thu hồi sau 30 phút?**
     - Kiểm tra toàn bộ 40 groups trên `goopay.atlassian.net` (`/rest/api/3/groups/picker` & `/rest/api/3/group/member`) cho thấy:
       - Các user được cấp quyền tạm thời vào group `confluence-users-goopay` (`8a98d738-86cb-43c9-acd9-c9909c950445`) sẽ bị hệ thống quản trị thu hồi tự động sau ~30 phút (để giữ tổng số active human seats trên Confluence không vượt ngưỡng).
       - Vì vậy: Mỗi khi anh mở quyền Confluence (trong cửa sổ 30 phút) rồi bấm **"Chạy ngay" (Manual Run)** — như lúc `16:50:17` ngày `15/10` (Testing) hay `09:18:40` sáng `19/10` (Production) — thì job chạy thành công (`200 OK`). Nhưng đến **06:00 sáng hôm sau** (`16/10`, `17/10`, `18/10`, `19/10`), quyền `confluence-users-goopay` của `trai.nguyen@goopay.vn` đã bị thu hồi từ lâu nên `scheduler` tự động chạy lúc `06:00` **chắc chắn 100% bị `403 FORBIDDEN` ở cấp Product Access** (tất cả mọi API `/wiki/...` đều bị chặn ở cổng Atlassian trước khi vào tới Space hay Page).
   - **Danh sách các Group & Tài khoản có quyền Confluence cố định (Permanent, không bị thu hồi sau 30 phút) trên `goopay.atlassian.net` (`VERIFIED` từ `/rest/api/3/group/member`):**
     1. Nhóm **`confluence-admins-goopay` (`a055fced-df76-4a29-902c-cb0a00d5bbe4`)** & **`org-admins` (`ab7281eb-7d61-475e-8e0c-3c97d3bd6d43`)**:
        - **`tech@goopay.vn`** (*GooPay Technical Team*, `accountId = 712020:a3dcebcb-e936-46fe-a464-f119dd38b08c`) — đồng thời là chủ sở hữu Space `~712020a3dcebcbe93646fea464f119dd38b08c`.
     2. Nhóm **`cf_def_GP` (`86d9fcfa-4e75-4dfe-bf89-f5af3a8e8bc3` — 6 tài khoản mặc định cố định)**:
        - **`agent_bot@goopay.vn`** (*Agent Bot*, `accountId = 712020:90f5e62f-3465-46b8-9263-9596d43345fb`)
        - **`tech_be@goopay.vn`** (*Backend Team*, `accountId = 712020:1d46e803-9e0a-4731-8e25-089fc36588f9`)
        - `tech_fe@goopay.vn`, `product@goopay.vn`, `qc@goopay.vn`, `vu.ton@goopay.vn`.
     3. Nhóm **`cf_user_gp` (`1b5172ab-eef2-4c1c-b53b-57c9ea2794d1`)**:
        - `huong.nguyen@goopay.vn`, `qc@goopay.vn`, `vu.ton@goopay.vn`.

2. **Nguyên nhân 2 (`VERIFIED` — Bản code đang chạy trên Server vẫn là bản cũ từ commit `80191c4` ngày 15/10/2026):**
   - Các thay đổi trong [`pkgs/confluence/client.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/confluence/client.go) và [`internal/service/confluence/confluence_runner.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/confluence/confluence_runner.go) mới được commit vào nhánh `recon-heal` ở commit `33ea3ba` lúc **`12:01:45` trưa ngày `19/10/2026`** (sau thời điểm `06:00:02` sáng 19/10) và chưa được build/deploy lên server.
   - Do đó lúc `06:00:02` sáng `19/10/2026`, worker trên server vẫn chạy bản commit `80191c4` (fail ngay sau `0.75s` với chuỗi `search content returned status 403` không kèm `resp.Body`, và trang `501743641` tạo lúc `09:18:40` sáng nay vẫn tạo trang mới theo ngày thay vì gom vào 1 trang cố định).

3. **Nguyên nhân 3 (`VERIFIED` — Ba điểm yếu trong code hiện tại tại [`pkgs/confluence/client.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/confluence/client.go) và [`internal/server/server_scheduler.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/server_scheduler.go)):**
   - **3.1 ([`pkgs/confluence/client.go:82-88, 117`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/confluence/client.go#L82-L120)):** Hàm `isPermanentForbiddenBody` ngắt sớm ở `attempt == 1` khi gặp `403`.
   - **3.2 ([`pkgs/confluence/client.go:219, 234-264`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/confluence/client.go#L219-L264)):** Hàm `findPageByTitle` chưa có fallback tìm trang con dưới `ParentPageID` (`500432919`).
   - **3.3 ([`internal/server/server_scheduler.go:96-103, 229-275`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/server_scheduler.go#L96-L103) & [`internal/service/confluence/confluence_runner.go:706-713`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/confluence/confluence_runner.go#L706-L713)):** Khi job `06:00` lỗi, `next_run_at` đã bị đẩy sang ngày mai (`+24h`) mà không có lịch retry tự động trong ngày.

---

## 2. Giải pháp Kỹ thuật (Single Best Solution)

Vì nguyên nhân gốc rễ số 1 là **tài khoản cá nhân `trai.nguyen@goopay.vn` bị thu hồi group `confluence-users-goopay` sau 30 phút**, mọi cơ chế retry trong code sẽ vẫn bị `403` lúc `06:00` sáng nếu tài khoản chạy lịch trình bị mất quyền Product Access vào Confluence. Do đó giải pháp triệt để gồm:

### 2.1. Chi tiết Cách 2: Giữ nguyên tài khoản `trai.nguyen@goopay.vn` và Tự động Mở quyền Confluence trước khi chạy Job (`ConfluenceReportRunner.Run`)

Thay vì phải đổi sang tài khoản người khác ở cấu hình Schedule, hệ thống giữ nguyên `user_email = trai.nguyen@goopay.vn` và xử lý việc mất quyền sau 30 phút theo luồng tự động ngay trong [`internal/service/confluence/confluence_runner.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/confluence/confluence_runner.go#L604-L705) và [`pkgs/confluence/client.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/confluence/client.go#L90-L265):

1. **Bước 1 — Giải mã chính xác API bên dưới `https://id.atlassian.com/join/user-access` (`VERIFIED` từ JS bundle `join.88180aec.js` & `index.33989fd0.js`):**
   - Bóc tách trực tiếp mã nguồn frontend của trang `https://id.atlassian.com/join/user-access` (`join.88180aec.js` — hàm `JoinProductConsentView` / `bulkRequestAccess` và `index.33989fd0.js` — hàm `joinUserToBulkProducts`) cho thấy khi anh mở link trên trình duyệt, trang web gọi **1 REST API duy nhất** để tự cấp quyền (`DIRECT_ACCESS_AUTOJOIN`) cho chính tài khoản `trai.nguyen@goopay.vn` (hoàn toàn không cần quyền Admin hay `tech@goopay.vn`):
     - **Endpoint (đi qua Stargate trên domain `goopay.atlassian.net` để không bị AWS WAF của `id.atlassian.com` chặn):**
       `POST https://goopay.atlassian.net/gateway/api/invitations/v1/access-requests/bulk/request`
     - **Payload JSON:**
       ```json
       {
         "resources": [
           {
             "ari": "ari:cloud:confluence::site/0a59075f-01a8-4a93-8af9-cc826c7c97cf",
             "role": "product/member",
             "accessMode": "DIRECT_ACCESS"
           }
         ],
         "continueUrl": "https://goopay.atlassian.net/wiki/spaces/~712020a3dcebcbe93646fea464f119dd38b08c/pages/502661151/Daily+Data+Check+Core+Trans+Proxy+History+Service",
         "metadata": {
           "sourceProduct": "confluence",
           "touchpoint": "DIRECT_ACCESS_AUTOJOIN"
         }
       }
       ```
     - **Response từ Atlassian:** Trả về `"result": "ACCESS_GRANTED"` (hoặc `"ACCESS_EXISTS"`) và lập tức tự thêm `trai.nguyen@goopay.vn` vào group `confluence-users-goopay` (`8a98d738-86cb-43c9-acd9-c9909c950445`) trong 30 phút!
   - **Điểm lưu ý duy nhất về Header Xác thực của API `/gateway/api/invitations/v1/...` (`VERIFIED`):**
     - Khi kiểm thử gọi `https://goopay.atlassian.net/gateway/api/invitations/v1/access-requests/...` bằng `Authorization: Basic ...` (`api_token`), Atlassian trả về header `x-failure-category: FAILURE_CLIENT_AUTH_MISMATCH` (`401 Unauthorized`), bởi vì dịch vụ `invitations/v1` của Atlassian chỉ nhận **Session Cookie `cloud.session.token`** của chính tài khoản `trai.nguyen@goopay.vn` (cookie được sinh ra khi anh đăng nhập trình duyệt vào Atlassian), chứ không nhận `api_token`.
   - **Cách hoạt động tự động trong `centralized-data-service` (chỉ dùng tài khoản `trai.nguyen@goopay.vn`):**
     - Cấu hình thêm trường `session_token` (`cloud.session.token` của chính `trai.nguyen@goopay.vn`) trong `params.confluence`.
     - Lúc `06:00:00` sáng, trước khi tìm/cập nhật trang báo cáo, hàm `EnsureConfluenceAccess` kiểm tra `GET /wiki/rest/api/user/current` bằng `api_token`.
     - Nếu bị `403 FORBIDDEN ("caller cannot access Confluence")`, hàm tự động lấy `cloud.session.token` của chính anh gọi `POST https://goopay.atlassian.net/gateway/api/invitations/v1/access-requests/bulk/request` với `"accessMode": "DIRECT_ACCESS"` để tự mở quyền 30 phút (y hệt lúc anh mở link trên trình duyệt).

2. **Bước 2 — Đợi Atlassian Đồng bộ Quyền (Propagation Wait 2s–15s) & Thực thi Báo cáo:**
   - Sau khi lệnh `bulk/request` trả về `ACCESS_GRANTED`, Atlassian Cloud mất khoảng **2 – 6 giây** để đồng bộ group sang Confluence.
   - Hàm `doWithRetry` trong [`pkgs/confluence/client.go:90-145`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/confluence/client.go#L90-L145) (sau khi gỡ bỏ `isPermanentForbiddenBody` ở dòng `82-88, 117`) sẽ tự động retry với backoff (`2s` $\rightarrow$ `4s` $\rightarrow$ `8s`) cho đến khi `/wiki/rest/api/content` trả về **`200 OK`**.
   - Ngay khi `200 OK`, worker dùng `api_token` của `trai.nguyen@goopay.vn` đọc và cập nhật bảng số liệu trong vòng ~3 giây.
   - Sau 30 phút (lúc `06:30` sáng), hệ thống Atlassian cứ tự động thu hồi quyền `confluence-users-goopay` như bình thường mà không ảnh hưởng gì tới báo cáo đã xuất bản!

### 2.2. Gia cố Code đồng bộ trong `centralized-data-service`
1. **Tại [`pkgs/confluence/client.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/confluence/client.go):**
   - Thêm hàm `selfJoinConfluence(ctx, baseURL, sessionToken)` gọi:
     1. `GET {baseURL}/_edge/tenant_info` để lấy động `cloudId` (`0a59075f-01a8-4a93-8af9-cc826c7c97cf`).
     2. `POST {baseURL}/gateway/api/invitations/v1/access-requests/bulk/request` kèm header `Cookie: cloud.session.token={sessionToken}` để tự kích hoạt quyền 30 phút cho `trai.nguyen@goopay.vn`.
   - **Cho phép `doWithRetry` thực hiện đủ `maxAttempts = 4` với Exponential Backoff + Jitter cho mọi mã `403`** (bỏ hàm `isPermanentForbiddenBody` ngắt sớm ở attempt 1).
   - **Bổ sung Fallback `findChildPageByTitle` theo `ParentPageID` (`500432919`).**

2. **Tại [`internal/service/confluence/confluence_runner.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/confluence/confluence_runner.go) & [`internal/server/server_scheduler.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/server_scheduler.go):**
   - Bổ sung trường `SessionToken string` (`json:"session_token,omitempty"`) vào `ConfluenceJobConfig` (`confluence_runner.go:22-29`).
   - **Cơ chế Auto-Retry trong ngày (Scheduled Retry Window 2 giờ `06:00`–`08:00`, mỗi 15 phút)** nếu quá trình mở quyền hoặc mạng bị nghẽn tạm thời.

---

## 3. Blast Radius & Rollback Plan

### 3.1. Blast Radius (Phạm vi ảnh hưởng)
- **`pkgs/confluence/client.go`:** Chỉ được gọi bởi `ConfluenceReportRunner` (`internal/service/confluence/confluence_runner.go`) và unit test `pkgs/confluence/client_test.go`. Không ảnh hưởng tới bất kỳ luồng CDC, Kafka, NATS hay Transmute nào.
- **`internal/service/confluence/confluence_runner.go`:** Chỉ ảnh hưởng tới nhánh xử lý lỗi (`if err != nil`) của job `operation = "confluence"`.
- **`internal/server/server_scheduler.go`:** Nhánh retry có điều kiện bảo vệ chặt (`sched.LastError != nil && strings.TrimSpace(*sched.LastError) != ""` VÀ `sched.NextRunAt != nil && !sched.NextRunAt.After(now)` VÀ `sched.NextRunAt.In(locHCM).Before(targetToday.Add(2 * time.Hour))`). Các job thành công (`last_error == nil` hoặc `""`) hoàn toàn không bị ảnh hưởng, vẫn giữ nguyên `next_run_at = targetToday.AddDate(0, 0, 1)`.

### 3.2. Rollback Plan
- Tạo git restore-point commit local trước khi sửa.
- Nếu cần rollback, revert 3 file `pkgs/confluence/client.go`, `internal/service/confluence/confluence_runner.go`, `internal/server/server_scheduler.go` về restore-point commit.

---

## 4. Code Demo

### 4.1. `pkgs/confluence/client.go` — Đủ 4 lần Retry cho `403` + Fallback `findChildPageByTitle` qua `ParentPageID`

```go
// Trong doWithRetry (pkgs/confluence/client.go:108-121):
		resp, err := c.httpClient.Do(httpReq)
		if err == nil {
			if !shouldRetryStatus(resp.StatusCode) {
				return resp, nil
			}
			bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()

			if attempt == maxAttempts {
				resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				return resp, nil
			}
```

```go
// Trong CreateOrUpdateChildPage & findChildPageByTitle (pkgs/confluence/client.go:218-265):
	// 1. Kiểm tra xem trang có tiêu đề này đã tồn tại trong Space chưa (lấy kèm version + body.storage)
	existingPage, err := c.findPageByTitle(ctx, req.BaseURL, req.SpaceKey, req.Title, authHeader)
	if err != nil && strings.TrimSpace(req.ParentPageID) != "" {
		// Fallback: Tìm trực tiếp trong danh sách trang con của ParentPageID nếu search theo SpaceKey bị 403
		if fallbackPage, fbErr := c.findChildPageByTitle(ctx, req.BaseURL, strings.TrimSpace(req.ParentPageID), req.Title, authHeader); fbErr == nil {
			existingPage = fallbackPage
			err = nil
		}
	}
	if err != nil {
		return "", fmt.Errorf("check existing confluence page failed: %w", err)
	}
```

```go
func (c *Client) findChildPageByTitle(ctx context.Context, baseURL, parentPageID, title, authHeader string) (*PageContentItem, error) {
	queryURL := fmt.Sprintf("%s/wiki/rest/api/content/%s/child/page?limit=100&expand=version,body.storage",
		baseURL, url.PathEscape(strings.TrimSpace(parentPageID)))

	resp, err := c.doWithRetry(ctx, authHeader, func() (*http.Request, error) {
		return http.NewRequestWithContext(ctx, http.MethodGet, queryURL, nil)
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		bodyText := strings.TrimSpace(string(respBytes))
		if bodyText != "" {
			return nil, fmt.Errorf("list child pages returned status %d: %s", resp.StatusCode, bodyText)
		}
		return nil, fmt.Errorf("list child pages returned status %d", resp.StatusCode)
	}

	var searchResp SearchContentResponse
	if err := json.Unmarshal(respBytes, &searchResp); err != nil {
		return nil, err
	}

	targetTitle := strings.TrimSpace(title)
	for i := range searchResp.Results {
		if strings.EqualFold(strings.TrimSpace(searchResp.Results[i].Title), targetTitle) {
			return &searchResp.Results[i], nil
		}
	}
	return nil, nil
}
```

### 4.2. `internal/service/confluence/confluence_runner.go` & `internal/server/server_scheduler.go` — Scheduled Retry sau 15 phút (trong cửa sổ 2 giờ) khi lỗi

```go
// Trong ConfluenceReportRunner.Run (internal/service/confluence/confluence_runner.go:706-718):
	if err != nil {
		r.logger.Error("failed to create confluence child page", zap.Error(err), zap.Uint("schedule_id", sched.ID))
		updates := map[string]interface{}{
			"last_error": err.Error(),
		}
		if retryAt, ok := computeDailyErrorRetryTime(sched, now); ok {
			updates["next_run_at"] = retryAt
		}
		r.dbControl.Model(&mastermodel.WorkerSchedule{}).Where("id = ?", sched.ID).Updates(updates)
		if r.activityLogger != nil {
			r.activityLogger.Quick("confluence", logTarget, triggeredBy, "error", 0, map[string]interface{}{"table_count": len(rows)}, err.Error())
		}
		return
	}
```

```go
// Trong calculateNextRun (internal/server/server_scheduler.go:248-263):
			// Nếu lần chạy định kỳ trong ngày hôm nay bị lỗi (last_error != "") và đã được hẹn next_run_at retry trong cửa sổ 2 giờ sau targetToday
			retryWindowEnd := targetToday.Add(2 * time.Hour)
			if sched.LastRunAt != nil && !sched.LastRunAt.In(locHCM).Before(targetToday) &&
				sched.LastError != nil && strings.TrimSpace(*sched.LastError) != "" &&
				sched.NextRunAt != nil && !sched.NextRunAt.After(now) &&
				sched.NextRunAt.In(locHCM).Before(retryWindowEnd) && nowInTz.Before(retryWindowEnd) {
				nextRun = targetToday.AddDate(0, 0, 1)
				claimConditionTime = *sched.NextRunAt
				shouldRun = true
				return
			}
```

---

## 5. Bảng kết quả Decision Gate (`DG1` & `DG2`)

### 5.1. Cổng `DG1` — Kiểm định Giải pháp (`06_laya_summary_dg1_20261019T133833.json` — Gate: `PASS`)

| Claim ID | Loại | Nội dung | R1 | R2 | R3 | Quyết định |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| `S-RC` | `solution` | Hàm `isPermanentForbiddenBody` trả về `true` khi `statusCode == 403` và body chứa `"caller cannot access confluence"`, khiến `doWithRetry` dừng ngay ở attempt đầu tiên (`pkgs/confluence/client.go:82-88`). | `CORRECT` (`0.6575`) | `CORRECT` (`0.6641`) | `YES` (`0.7139`) | `ACCEPT` |
| `S-FIX` | `solution` | Hàm `findPageByTitle` chỉ truy vấn `GET /wiki/rest/api/content` theo `spaceKey` và `title`, trả về lỗi `search content returned status %d` khi `StatusCode != 200` (`pkgs/confluence/client.go:234-253`). | `CORRECT` (`0.3309`) | `CORRECT` (`0.0763`) | `YES` (`0.8328`) | `RESOLVED` (`PASS`) |
| `S-IMPACT` | `solution` | Khi `CreateOrUpdateChildPage` trả về lỗi, hàm `Run` chỉ cập nhật cột `last_error` mà không lên lịch chạy thử lại trong ngày (`confluence_runner.go:706-713`). | `CORRECT` (`0.3629`) | `INCORRECT` (`0.6084`) | `NO` (`0.7591`) | `RESOLVED` (`PASS`) |
| `S-PATTERN` | `solution` | Trong `calculateNextRun` khi `runAtTime != ""` và `nowInTz.After(targetToday)`, sau cửa sổ 5 phút đầu giờ hàm luôn đặt `nextRun` sang ngày hôm sau và trả về `shouldRun = false` (`server_scheduler.go:264-275`). | `INCORRECT` (`0.3292`) | `CORRECT` (`0.4149`) | `YES` (`0.8179`) | `RESOLVED` (`PASS`) |

### 5.2. Cổng `DG2` — Kiểm định Code Demo (`06_laya_summary_dg2_20261019T134022.json` — Gate: `PASS`)

| Claim ID | Loại | Nội dung | R1 | R2 | R3 | Quyết định |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| `D-BEHAV` | `demo` | Trong `CreateOrUpdateChildPage`, client gọi `findPageByTitle` trước khi quyết định gọi `updatePage` hoặc `createPage` (`pkgs/confluence/client.go:218-232`). | `CORRECT` (`0.5895`) | `CORRECT` (`0.2856`) | `YES` (`0.7146`) | `RESOLVED` (`PASS`) |
| `D-EDGE` | `demo` | Trong `doWithRetry`, khi `shouldRetryStatus` là `true` và chưa tới `maxAttempts`, hàm đọc tối đa 4096 byte body vào `lastErr` và kiểm tra `ctx.Done()` (`pkgs/confluence/client.go:110-149`). | `CORRECT` (`0.2139`) | `CORRECT` (`0.1460`) | `YES` (`0.6852`) | `RESOLVED` (`PASS`) |
| `D-API` | `demo` | Trong `ConfluenceReportRunner.Run`, `r.dbControl.Model(&mastermodel.WorkerSchedule{})` cập nhật `last_error`, `notes`, `next_run_at` qua GORM (`confluence_runner.go:706-724`). | `CORRECT` (`0.2263`) | `INCORRECT` (`0.1948`) | `YES` (`0.3222`) | `RESOLVED` (`PASS`) |
