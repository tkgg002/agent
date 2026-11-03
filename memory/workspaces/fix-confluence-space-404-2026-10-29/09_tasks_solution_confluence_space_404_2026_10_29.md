# 09 Tasks Solution: Audit & Xử lý Lỗi Confluence 404 (`No space with key : ~712020a3dcebcbe93646fea464f119dd38b08c`) khi dùng `sre@goopay.vn`

- **Workspace**: `fix-confluence-space-404-2026-10-29`
- **Suffix**: `confluence_space_404_2026_10_29`
- **Tier**: `T2`

---

## 1. Phân tích Nguyên nhân Gốc rễ (`S2 Investigate` — `100% VERIFIED`)

### Phát hiện `F1` (`VERIFIED` trên Atlassian Confluence Cloud API): Tài khoản `sre@goopay.vn` chưa thuộc group `confluence-users-goopay` và chưa được cấp quyền `View` (`read:space`) + `Add Page` (`create:page`) trên Space `~712020a3dcebcbe93646fea464f119dd38b08c` (`GooPay Technical Team`)
- **Bằng chứng thực tế từ Atlassian Confluence REST API (`https://goopay.atlassian.net`):**
  1. **Trạng thái xác thực (`authorized: true`):**
     - Trong lỗi trả về: `{"statusCode":404,"data":{"authorized":true,"valid":true,...},"message":"...No space with key : ~712020a3dcebcbe93646fea464f119dd38b08c"}`.
     - Tra cứu `GET /wiki/rest/api/user?accountId=712020:fd809a67-b541-4346-9aa7-7a482cf14324&expand=operations`:
       - Email: `sre@goopay.vn` (`displayName: "sre"`, `accountId: "712020:fd809a67-b541-4346-9aa7-7a482cf14324"`, `accountStatus: "active"`).
       - Email + API Token của `sre@goopay.vn` **hoàn toàn hợp lệ** và đã đăng nhập thành công (`authorized: true`).
  2. **Thông tin Space chứa trang báo cáo (`ParentPageID = 500432919`):**
     - Tra cứu `GET /wiki/rest/api/content/500432919?expand=space`:
       - Trang cha `500432919` (*"Daily Data Report - Core Trans Proxy History Service"*) và trang báo cáo con `502661151` (*"[Daily Data Check] Core Trans Proxy History Service"*) nằm trong Space:
         - **Space Key:** `~712020a3dcebcbe93646fea464f119dd38b08c`
         - **Space Name:** `GooPay Technical Team` (`type: personal`, owner: `tech@goopay.vn`).
  3. **Đối chiếu Group Membership & Space Permissions (`trai.nguyen@goopay.vn` vs `sre@goopay.vn`):**
     - Tra cứu `GET /wiki/rest/api/user/memberof`:
       - `trai.nguyen@goopay.vn` (`712020:d5d8178d-212b-4f89-afec-9f0a21285906`) thuộc: **`['jira-users-goopay', 'confluence-users-goopay']`**.
       - `sre@goopay.vn` (`712020:fd809a67-b541-4346-9aa7-7a482cf14324`) chỉ thuộc: **`['jira-users-goopay', 'cf_def_GP']`** — **KHÔNG thuộc group `confluence-users-goopay`!**
     - Tra cứu `GET /wiki/rest/api/space/~712020a3dcebcbe93646fea464f119dd38b08c?expand=permissions`:
       - Quyền `read:space` (`View`) và `create:page` (`Add Page`) trên Space `GooPay Technical Team` (`~712020a3dcebcbe93646fea464f119dd38b08c`) chỉ được cấp cho:
         - **Groups:** `confluence-users-goopay`, `confluence-admins-goopay`, `org-admins` (không có `cf_def_GP`).
         - **Users:** `tech@goopay.vn`, `product@goopay.vn`, `vu.ton@goopay.vn`, `trai.nguyen@goopay.vn`, `tech_fe@goopay.vn`, `tech_be@goopay.vn`, `qc@goopay.vn` (không có `sre@goopay.vn`).
       - Tương tự, trên **Space `SRE`** (`key: SRE`), quyền `read:space` / `create:page` cũng chỉ cấp cho `confluence-users-goopay` và `cf_user_gp` (không có `cf_def_GP` hay `sre@goopay.vn`).
  4. **Cơ chế trả mã `404` của Atlassian Confluence REST API:**
     - Khi một user đã xác thực (`authorized: true`) gọi `GET /wiki/rest/api/content?spaceKey=~712020a3dcebcbe93646fea464f119dd38b08c&title=...` vào một Space mà user đó **không có quyền `View` (`read:space`)**, Confluence Cloud chủ động trả về **`404 NotFoundException: No space with key : ~712020a3dcebcbe93646fea464f119dd38b08c`** (thay vì 403) để ẩn sự tồn tại của Space đối với user chưa được phân quyền.

---

### Phát hiện `F2` (`VERIFIED` tại `pkgs/confluence/client.go:210-264`): `CreateOrUpdateChildPage` chưa tự động đồng bộ `SpaceKey` từ `ParentPageID` và chưa giải mã lỗi phân quyền `404 (authorized: true)`
- **Bằng chứng mã nguồn (`file:line`):**
  - `centralized-data-service/pkgs/confluence/client.go:219-222`:
    ```go
    existingPage, err := c.findPageByTitle(ctx, req.BaseURL, req.SpaceKey, req.Title, authHeader)
    if err != nil {
        return "", fmt.Errorf("check existing confluence page failed: %w", err)
    }
    ```
  - `centralized-data-service/internal/service/confluence/confluence_runner.go:633-638` và `cdc-cms-web/src/pages/ActivityManager.tsx:1215, 1304, 1377, 1744`:
    - Mặc định gán `SpaceKey = "~712020a3dcebcbe93646fea464f119dd38b08c"`.
    - **Hệ quả 1:** Nếu người vận hành đổi `ParentPageID` sang một trang cha thuộc Space khác (ví dụ trang trong Space `SRE` như `31555830`) mà quên sửa ô `Space Key` trên UI (vẫn để mặc định `~712020a3dcebcbe93646fea464f119dd38b08c`), `CreateOrUpdateChildPage` sẽ tìm/tạo trang sai Space hoặc văng lỗi 404/400.
    - **Hệ quả 2:** Khi Confluence trả về `404` kèm `"authorized":true` và `"No space with key"`, `findPageByTitle` (`client.go:247-253`) chỉ in chuỗi JSON thô của Java exception (`NotFoundException: No space with key`), khiến người vận hành tưởng nhầm là sai Space Key hoặc hỏng API Token thay vì biết ngay là **tài khoản `UserEmail` thiếu quyền `View / Add Page` trên Space đó**.

---

## 2. Thiết kế Giải pháp Duy nhất Ánh xạ 1-1 (`S3 Solution`)

### Giải pháp cho `F1` (Cấp quyền cho `sre@goopay.vn` trên Atlassian Confluence Cloud)
Để tài khoản `sre@goopay.vn` push được báo cáo vào trang cha `500432919` (*"Daily Data Report - Core Trans Proxy History Service"* thuộc Space `GooPay Technical Team` — `~712020a3dcebcbe93646fea464f119dd38b08c`), thực hiện **1 trong 2 thao tác phân quyền** trên Atlassian (`https://goopay.atlassian.net`):
1. **Cách chuẩn nhất (Cấp theo Group ở Atlassian Admin):**
   - Dùng tài khoản Org Admin (`org-admins` / `confluence-admins-goopay`) vào `admin.atlassian.com` → **Users** → chọn **`sre@goopay.vn`** → **Add to group: `confluence-users-goopay`**.
   - Ngay khi nằm trong group `confluence-users-goopay`, `sre@goopay.vn` tự động có đầy đủ quyền `View Space` + `Add/Update Page` trên cả Space `GooPay Technical Team` (`~712020a3dcebcbe93646fea464f119dd38b08c`) lẫn Space `SRE`!
2. **Hoặc cấp trực tiếp trên Space `GooPay Technical Team`:**
   - Đăng nhập Confluence bằng một trong các tài khoản đang có quyền **Space Admin (`administer:space` / `manage_users:space`)** trên Space `GooPay Technical Team`:
     - `tech@goopay.vn` (*GooPay Technical Team*)
     - `product@goopay.vn` (*Product GooPay*)
     - `vu.ton@goopay.vn` (*Vũ Tôn Thất*)
     - `tech_be@goopay.vn` (*Backend Team*)
     - `tech_fe@goopay.vn` (*Tech FE GP*)
     *(Lưu ý: `trai.nguyen@goopay.vn` chỉ có quyền `View` + `Add Page`, không có quyền `Space Admin` để mở trang Space Settings -> Space Permissions).*
   - Truy cập: `https://goopay.atlassian.net/wiki/spaces/~712020a3dcebcbe93646fea464f119dd38b08c/settings/permissions` → mục **Individual Users** → bấm **Edit Permissions** → thêm user **`sre` (`sre@goopay.vn`)** với quyền **All (`View` Space + `Add` Page)**.

---

### Giải pháp cho `F2` (Tự động đồng bộ `SpaceKey` từ `ParentPageID` + Chẩn đoán tường minh lỗi phân quyền Space `404 (authorized: true)` tại `pkgs/confluence/client.go`)
1. **Tự động phân giải `SpaceKey` từ `ParentPageID` (`resolveParentPageSpaceKey`):**
   - Trong `CreateOrUpdateChildPage` (`centralized-data-service/pkgs/confluence/client.go:210-232`), nếu `strings.TrimSpace(req.ParentPageID) != ""`, gọi nhẹ `c.resolveParentPageSpaceKey(ctx, req.BaseURL, req.ParentPageID, authHeader)` (`GET /wiki/rest/api/content/{ParentPageID}?expand=space`):
     - Nếu lấy được `space.key` hợp lệ từ `ParentPageID` (ví dụ khi người dùng đổi `ParentPageID` sang một trang thuộc Space `SRE` nhưng quên đổi `SpaceKey` mặc định `~712020...`), tự động cập nhật `req.SpaceKey = parentSpaceKey`.
     - Nếu `GET /wiki/rest/api/content/{ParentPageID}` trả về `404` kèm `"authorized":true`, trả về ngay thông báo lỗi chẩn đoán rõ: tài khoản `req.UserEmail` đã xác thực thành công (`authorized=true`) nhưng **không có quyền `View` đối với `ParentPageID` / `SpaceKey`** (cần thêm `req.UserEmail` vào group `confluence-users-goopay` hoặc cấp quyền `View + Add Page` trong Space Permissions).
2. **Giải mã lỗi `404 No space with key` trong `findPageByTitle` (`formatConfluenceSearchError`):**
   - Khi `findPageByTitle` nhận `resp.StatusCode == http.StatusNotFound` mà body chứa `"authorized":true` và `"No space with key"`, kèm thêm chẩn đoán tiếng Việt rõ ràng vào chuỗi lỗi để hiển thị trực tiếp trên bảng `Activity Log` / `Worker Schedule` của CMS UI.

---

### Bảng Ánh xạ Điểm Gọi Thực thi & Thời điểm Kích hoạt (Lesson `L002`)

| Hàm / Phương thức | File & Dòng (`file:line`) | Caller (`file:line`) | Thời điểm kích hoạt (Lifecycle / Event Trigger) |
| :--- | :--- | :--- | :--- |
| `(*Client).resolveParentPageSpaceKey` | `centralized-data-service/pkgs/confluence/client.go:234-268` | `(*Client).CreateOrUpdateChildPage` (`client.go:219`) | Kích hoạt đầu tiên trong `CreateOrUpdateChildPage` khi `ConfluenceReportRunner.Run` (`confluence_runner.go:695`) chạy theo lịch trình hoặc bấm "Chạy ngay" trên CMS. |
| `formatConfluenceSpaceError` | `centralized-data-service/pkgs/confluence/client.go:270-285` | `(*Client).findPageByTitle` (`client.go:305`) | Kích hoạt khi `GET /wiki/rest/api/content?spaceKey=...` trả về `HTTP 404` (`No space with key` + `authorized: true`). |

---

## 3. Code Demo (`centralized-data-service/pkgs/confluence/client.go:209-315`)

```go
// CreateOrUpdateChildPage tạo mới hoặc cập nhật (append ngày tiếp theo phía dưới) trang con dưới ParentPageID
func (c *Client) CreateOrUpdateChildPage(ctx context.Context, req CreateChildPageRequest) (string, error) {
	req.BaseURL = strings.TrimRight(req.BaseURL, "/")
	if req.BaseURL == "" {
		req.BaseURL = "https://goopay.atlassian.net"
	}

	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(req.UserEmail+":"+req.APIToken))

	// 0. Nếu có ParentPageID, tự động đồng bộ SpaceKey thực tế từ trang cha (tránh lệch SpaceKey khi đổi ParentPageID)
	if strings.TrimSpace(req.ParentPageID) != "" {
		if parentSpaceKey, err := c.resolveParentPageSpaceKey(ctx, req.BaseURL, strings.TrimSpace(req.ParentPageID), authHeader); err == nil && parentSpaceKey != "" {
			req.SpaceKey = parentSpaceKey
		}
	}

	// 1. Kiểm tra xem trang có tiêu đề này đã tồn tại trong Space chưa (lấy kèm version + body.storage)
	existingPage, err := c.findPageByTitle(ctx, req.BaseURL, req.SpaceKey, req.Title, req.UserEmail, authHeader)
	if err != nil {
		return "", fmt.Errorf("check existing confluence page failed: %w", err)
	}
	if existingPage != nil {
		// Trang đã tồn tại -> Append khối ngày mới xuống dưới nội dung cũ (hoặc cập nhật tại chỗ nếu trùng ngày)
		mergedReq := req
		mergedReq.BodyXHTML = MergeDailySectionXHTML(existingPage.Body.Storage.Value, req.SectionDate, req.BodyXHTML)
		return c.updatePage(ctx, mergedReq, existingPage.ID, existingPage.Version.Number+1, authHeader)
	}

	// 2. Trang chưa tồn tại -> Tạo mới dưới ParentPageID
	return c.createPage(ctx, req, authHeader)
}

func (c *Client) resolveParentPageSpaceKey(ctx context.Context, baseURL, parentPageID, authHeader string) (string, error) {
	queryURL := fmt.Sprintf("%s/wiki/rest/api/content/%s?expand=space", baseURL, url.PathEscape(parentPageID))
	resp, err := c.doWithRetry(ctx, authHeader, func() (*http.Request, error) {
		return http.NewRequestWithContext(ctx, http.MethodGet, queryURL, nil)
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("get parent page %s returned status %d", parentPageID, resp.StatusCode)
	}

	var payload struct {
		Space struct {
			Key string `json:"key"`
		} `json:"space"`
	}
	if err := json.Unmarshal(respBytes, &payload); err != nil {
		return "", err
	}
	return strings.TrimSpace(payload.Space.Key), nil
}

func formatConfluenceSpaceError(statusCode int, spaceKey, userEmail, bodyText string) error {
	lower := strings.ToLower(bodyText)
	if statusCode == http.StatusNotFound && strings.Contains(lower, "no space with key") && strings.Contains(lower, `"authorized":true`) {
		return fmt.Errorf(
			"search content returned status 404 (account %q authenticated successfully, but lacks View/Add Page permission on Confluence space %q or space does not exist; grant Space Permissions or add %q to group 'confluence-users-goopay'): %s",
			userEmail, spaceKey, userEmail, bodyText,
		)
	}
	if bodyText != "" {
		return fmt.Errorf("search content returned status %d: %s", statusCode, bodyText)
	}
	return fmt.Errorf("search content returned status %d", statusCode)
}
```
