# BÁO CÁO KIỂM TOÁN PHẢN BIỆN & QC TOÀN DIỆN (`audit_report_confluence_append_daily.md`)

- **Ngày kiểm toán:** `2026-10-16`
- **Workspace:** `fix-recon-daily-flow-cud-2026-10-06`
- **Chế độ thực thi:** **AUDIT & PROPOSE-ONLY** (Tuân thủ tuyệt đối Rule #1, Rule #13 & Lesson `[2026-10-14] #audit-propose-only` — **Tuyệt đối không tự ý sửa code dự án trong bước Audit khi chưa có lệnh `APPROVE` mới của User**).
- **Bằng chứng Raw JSON `/laya_router`:** [`06_laya_raw_confluence_append_audit.json`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/fix-recon-daily-flow-cud-2026-10-06/06_laya_raw_confluence_append_audit.json)

---

## I. KIỂM ĐỊNH ĐỘ TRUNG THỰC: TÁCH BẠCH RÕ "UNIT TEST MOCKUP" VÀ "RUNTIME THỰC TẾ" (KHÔNG BÁO CÁO LÁO, KHÔNG HALLUCINATION)

Trả lời thẳng và trung thực tuyệt đối theo yêu cầu *"Kiểm tra xem có suy diễn, báo cáo láo không... ko test bằng mockup và Hallucination kết quả, nếu cần phải nói phối hợp với anh trace runtime value"*:

### 1. Những gì ĐÃ ĐƯỢC kiểm chứng thật ở lượt trước
1. **Unit Test logic trong Go (`go test -v ./pkgs/confluence/... ./internal/service/confluence/...`)**:
   - Đã chạy thật lệnh `go test` và PASS 100% các kịch bản:
     - `TestMergeDailySectionXHTML_AppendAndIdempotentUpdate`: Kiểm chứng hàm `MergeDailySectionXHTML` với dữ liệu XHTML khi trang trống, khi nối ngày mới (`2026-10-15` xuống dưới `2026-10-14`), khi cập nhật đè cùng ngày (`2026-10-14`), và khi Confluence chuyển đổi tiếng Việt (`Báo cáo`) sang HTML entities (`B&aacute;o c&aacute;o`).
     - `TestResolveSingleReportPageTitle`: Kiểm chứng hàm `resolveSingleReportPageTitle` cắt bỏ biến `{{.ReportDate}}` khỏi `TitleTemplate`.
     - `TestLiveBranchLineageAndReportRow`: Chạy thật truy vấn trên DB Postgres Local (`cdc_dw:5433`) kiểm tra dòng báo cáo thuyết minh.

### 2. Những gì CHỈ MỚI TEST BẰNG MOCKUP (`httptest.NewServer`) & CHƯA CHẠY TRÊN RUNTIME THỰC TẾ
1. **`TestCreateOrUpdateChildPage_AppendFlowAndRetry403` trong [`pkgs/confluence/client_test.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/confluence/client_test.go#L41-L148) LÀ MOCKUP HTTP SERVER (`httptest.NewServer`):**
   - Bài test này dựng một HTTP server giả lập trong bộ nhớ RAM (`httptest.NewServer`) để giả lập phản hồi `403` ở lần gọi thứ nhất, `200` ở lần gọi thứ hai, `POST` tạo trang và `PUT` cập nhật trang.
   - **Sự thật:** Nó chỉ chứng minh code Go của `Client` gọi đúng thứ tự hàm trong RAM, **CHƯA HỀ gọi vào Atlassian Confluence Cloud thật (`https://goopay.atlassian.net`)** để kiểm chứng việc gộp trang thật (`501088257` dưới trang cha `500432919`).
2. **Các tiến trình Service đang chạy trên máy (`:8082` và `:8083`) CHƯA ĐƯỢC BUILD LẠI & RESTART:**
   - Kiểm tra trực tiếp bằng lệnh `lsof -iTCP -sTCP:LISTEN -P -n` và `ps -o pid,lstart,command -p 42655,44286`:
     - **`PID 42655` (`:8082` — `centralized-data-service` / `cdc-worker`)**: Vẫn đang chạy bản binary khởi động từ **`Thu Oct 15 15:52:07 2026`**. Toàn bộ code mới sửa sáng nay (`16/10/2026`) trong `pkgs/confluence/client.go` và `confluence_runner.go` **chưa hề được nạp vào tiến trình `:8082` đang chạy**.
     - **`PID 44286` (`:8083` — `cdc-cms-service`)**: Vẫn đang chạy bản binary khởi động từ **`Thu Oct 15 16:04:42 2026`**.
3. **Frontend [`cdc-cms-web`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/ActivityManager.tsx) CHƯA CHẠY `npm run build` (`tsc -b && vite build`):**
   - Sau khi sửa `ActivityManager.tsx` ở lượt trước, Agent chưa chạy `npm run build` để kiểm tra biên dịch TypeScript và bundle thực tế.
4. **Trạng thái quyền thực tế trên `https://goopay.atlassian.net`:**
   - Tài khoản cấu hình trong lịch chạy (`trai.nguyen@goopay.vn`) hiện đang chỉ có group `"jira-users-goopay"` (mất group Confluence `cf_user_gp` / `cf_def_GP`), nên Atlassian đang trả về `403 FORBIDDEN: "Request rejected because caller cannot access Confluence"`. Khi chưa được cấp lại quyền Confluence hoặc đổi sang token tài khoản có quyền Confluence, không một lệnh gọi thật nào tới `goopay.atlassian.net/wiki/rest/api/content` có thể trả về `200 OK`.

---

## II. KIỂM TOÁN QUY TRÌNH & PHẠM VI (PROCESS & SCOPE AUDIT)

### 1. Phát hiện P1 (Nghiêm trọng): Bỏ sót 2 trên 4 Hồ sơ giải pháp khi nhận lệnh `"APPROVE toàn bộ các vấn đề tìm đc cho anh"`
- **Sự thật đối chiếu trong Workspace [`fix-recon-daily-flow-cud-2026-10-06`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/fix-recon-daily-flow-cud-2026-10-06):**
  Khi anh ra lệnh `"APPROVE toàn bộ các vấn đề tìm đc cho anh"`, trong thư mục workspace có tổng cộng **4 hồ sơ `09_tasks_solution_*.md` đang ở trạng thái `PROPOSED` (chờ APPROVE)** từ các câu hỏi liên tiếp của anh trong phiên:
  1. ✅ [`09_tasks_solution_confluence_scheduled_403.md`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/fix-recon-daily-flow-cud-2026-10-06/09_tasks_solution_confluence_scheduled_403.md) *(Đã thực thi một phần trong `client.go`)*
  2. ✅ [`09_tasks_solution_confluence_append_daily.md`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/fix-recon-daily-flow-cud-2026-10-06/09_tasks_solution_confluence_append_daily.md) *(Đã thực thi trong `client.go`, `confluence_runner.go`, `ActivityManager.tsx`)*
  3. ❌ **BỊ BỎ SÓT — [`09_tasks_solution_cms_shutdown_db_closed_race.md`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/fix-recon-daily-flow-cud-2026-10-06/09_tasks_solution_cms_shutdown_db_closed_race.md):**
     - **Vấn đề của anh:** Log báo hàng loạt lỗi `sql: database is closed` tại `system_health_queries.go:106, 126, 55`, `probes/postgres.go:23`, `alert_manager.go:337` và `redis SET snapshot failed: redis: client is closed` khi `cdc-cms-service` dừng hoặc đụng port `:8083`.
     - **Hiện trạng file thật:** [`cdc-cms-service/internal/server/server.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/server/server.go#L393-L446) và [`cdc-cms-service/internal/infra/observability/system_health_collector.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/observability/system_health_collector.go#L225-L248) **hoàn toàn chưa được sửa** (`git status` tại `cdc-cms-service` trống trơn).
  4. ❌ **BỊ BỎ SÓT — [`09_tasks_solution_schedule_timezone_testing.md`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/fix-recon-daily-flow-cud-2026-10-06/09_tasks_solution_schedule_timezone_testing.md):**
     - **Vấn đề của anh:** Trên môi trường Testing, cột `Lần chạy cuối` (`15/10/2026 09:46:33`) và `Lần chạy kế` (`15/10/2026 17:05:00`) bị hiển thị lệch `-7 tiếng` (giờ UTC thay vì giờ Việt Nam `16:46:33` và `16/10/2026 00:05:00`).
     - **Hiện trạng file thật:** [`cdc-cms-web/src/utils/datetime.ts`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/utils/datetime.ts#L5-L18) và [`cdc-cms-service/internal/infra/persistence/scheduler/worker_schedule_read_repo_gorm.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/scheduler/worker_schedule_read_repo_gorm.go#L116-L147) **hoàn toàn chưa được sửa**.
- **Nguyên nhân gốc rễ (Root Cause của Agent):** Do thiên kiến gần nhất (*Recency Bias*), Agent chỉ nhớ 2 yêu cầu vừa trao đổi ngay sát trước câu `"APPROVE toàn bộ"` mà không chạy lệnh liệt kê toàn bộ các file `09_tasks_solution_*.md` đang ở trạng thái `PROPOSED` trong workspace.

### 2. Phát hiện P2: Lỗi Format dòng Log cuối trong `05_progress.md` & Thiếu chạy `verify_governance.py`
- Tại lượt trước, dòng 1106 của [`05_progress.md`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/fix-recon-daily-flow-cud-2026-10-06/05_progress.md#L1101-L1106) được ghi bằng `write_to_file(Append: true)` thiếu dấu gạch đầu dòng `- ` ở đầu dòng, và các dòng 1101–1105 ghi `[Brain]` / `[Muscle]` thay vì `[Agent:Model]`, khiến script `verify_governance.py` báo lỗi regex format nếu kiểm tra. (Sẽ được chuẩn hóa ngay trong bước cập nhật `05_progress.md` của phiên Audit này).

---

## III. KIỂM TOÁN PHẢN BIỆN TỪNG FILE, TỪNG DÒNG CODE ĐÃ UPDATE (LINE-BY-LINE ADVERSARIAL CODE AUDIT)

Dưới đây là kết quả rà soát từng dòng (line-by-line) trên 5 file đã chỉnh sửa ở lượt trước, chỉ rõ những dòng làm đúng và những lỗi tiềm ẩn / thiếu sót cần vá:

---

### 1. File [`centralized-data-service/pkgs/confluence/client.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/confluence/client.go)

#### A. Các dòng đã cài đặt đúng
- **Dòng 32–34 (`CreateChildPageRequest`):** Thêm `SectionDate string` để truyền ngày báo cáo (`YYYY-MM-DD`) tách biệt khỏi tiêu đề trang cố định.
- **Dòng 189 (`findPageByTitle`):** Đổi `expand=version` thành `expand=version,body.storage` để lấy luôn nội dung XHTML hiện có của trang trong 1 lượt `GET`.
- **Dòng 200–207 (`findPageByTitle`):** Đọc toàn bộ `resp.Body` (`io.LimitReader(resp.Body, 4096)`) khi `resp.StatusCode != http.StatusOK` và đưa vào chuỗi lỗi trả về:
  `search content returned status %d: %s` — chấm dứt tình trạng mù thông tin khi gặp `403`.

#### B. Phát hiện C1 (Nghiêm trọng - Cần sửa): Hàm `updatePage` (Dòng 283–303) thiếu trường `"ancestors"` trong Payload `PUT`
- **Code hiện tại tại [`pkgs/confluence/client.go:283-303`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/confluence/client.go#L283-L303):**
  ```go
  func (c *Client) updatePage(ctx context.Context, pageID string, nextVersion int, req CreateChildPageRequest, xhtmlBody string) (*CreateChildPageResponse, error) {
  	endpoint := c.baseURL + "/wiki/rest/api/content/" + url.PathEscape(pageID)

  	payload := map[string]any{
  		"id":    pageID,
  		"type":  "page",
  		"title": req.Title,
  		"space": map[string]any{
  			"key": req.SpaceKey,
  		},
  		"version": map[string]any{
  			"number": nextVersion,
  		},
  		"body": map[string]any{
  			"storage": map[string]any{
  				"value":          xhtmlBody,
  				"representation": "storage",
  			},
  		},
  	}
  ```
- **Tư duy phản biện chỉ ra lỗi:**
  - Hãy đối chiếu với hàm `createPage` ngay phía trên ([`pkgs/confluence/client.go:241-245`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/confluence/client.go#L241-L245)): `createPage` có gắn `"ancestors": []map[string]any{{"id": strings.TrimSpace(req.ParentPageID)}}`.
  - Trong **Atlassian Confluence REST API v1 (`PUT /wiki/rest/api/content/{id}`)**, nếu cập nhật một trang con (Child Page của `500432919`) mà **không truyền trường `"ancestors"`** trong JSON body của `PUT`, Confluence có hành vi mặc định trên nhiều phiên bản là **gỡ trang đó khỏi trang cha (`500432919`) và đẩy nó ra ngoài Root của Space (`~712020a3dcebcbe93646fea464f119dd38b08c`)**!
- **Giải pháp đề xuất (Chờ anh APPROVE):**
  Bổ sung khối `if strings.TrimSpace(req.ParentPageID) != ""` vào `updatePage` giống hệt `createPage`:
  ```go
  	if strings.TrimSpace(req.ParentPageID) != "" {
  		payload["ancestors"] = []map[string]any{
  			{"id": strings.TrimSpace(req.ParentPageID)},
  		}
  	}
  ```

#### C. Phát hiện C2 (Trung bình - Cần sửa): Hàm `doWithRetry` (Dòng 76–124) xóa mất `resp.Body` khi retry và retry vô ích khi `403` là lỗi mất quyền vĩnh viễn
- **Code hiện tại tại [`pkgs/confluence/client.go:100-112`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/confluence/client.go#L100-L112):**
  ```go
  		if !shouldRetryStatus(resp.StatusCode) || attempt == maxAttempts {
  			return resp, nil
  		}

  		// Đóng body cũ trước khi retry để tránh rò rỉ kết nối TCP.
  		_, _ = io.Copy(io.Discard, resp.Body)
  		_ = resp.Body.Close()
  ```
- **Tư duy phản biện chỉ ra 2 điểm yếu:**
  1. **Mất nguyên nhân gốc nếu lần retry sau bị timeout (`ctx.Done()`):** Khi lần 1 trả về `403` hoặc `503`, `resp.Body` bị `io.Discard` và `lastErr` vẫn là `nil`. Nếu trong lúc chờ backoff hoặc ở lần gọi thứ 2/3, context hết hạn hoặc rớt mạng, hàm trả về lỗi `context deadline exceeded` thay vì thông báo kèm mã HTTP và nội dung lỗi của lần trước.
  2. **Retry vô ích 14 giây (`2s + 4s + 8s`) khi gặp lỗi `403` do thu hồi quyền cứng (`"caller cannot access Confluence"`):**
     - Chỉ nên retry `403` khi đó là lỗi Edge/WAF/Rate-limit tạm thời. Nếu đọc nhanh `resp.Body` của `403` mà thấy chuỗi `"caller cannot access Confluence"` hoặc `"NOT_AUTHORIZED"`, đó là lỗi phân quyền IAM cố định — phải trả về `resp` ngay lập tức ở lần 1 để báo lỗi rõ ràng thay vì bắt worker ngủ chờ 14 giây vô ích.
- **Giải pháp đề xuất (Chờ anh APPROVE):**
  Đọc tối đa 4KB body vào bộ nhớ đệm trước khi quyết định retry: nếu là `403` có chứa `"caller cannot access Confluence"` thì khôi phục `resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))` và trả về ngay ở attempt 1; nếu retry thì lưu `lastErr = fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))` để không bao giờ mất dấu lỗi gốc.

#### D. Phát hiện C3 (Nhỏ - Cải thiện độ bền): Hàm `MergeDailySectionXHTML` (Dòng 131–165) xác định điểm kết thúc Section ngày
- **Code hiện tại tại [`pkgs/confluence/client.go:158-165`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/confluence/client.go#L158-L165):**
  ```go
  	afterMarker := existingXHTML[markerIdx:]
  	tableEndRel := strings.Index(afterMarker, "</table>")
  	if tableEndRel == -1 {
  		return existingXHTML[:h2StartIdx] + newDayXHTML
  	}
  	sectionEndIdx := markerIdx + tableEndRel + len("</table>")
  	return existingXHTML[:h2StartIdx] + newDayXHTML + existingXHTML[sectionEndIdx:]
  ```
- **Tư duy phản biện chỉ ra rủi ro khi mở rộng mẫu báo cáo:**
  Hiện tại mỗi ngày chỉ có đúng 1 bảng `<table>...</table>` nên tìm `</table>` đầu tiên sau `markerIdx` là chạy đúng. Tuy nhiên, nếu sau này trong 1 ngày báo cáo có thêm dòng ghi chú `<p>...</p>` phía dưới bảng hoặc có bảng phụ thứ 2, việc cắt tại `</table>` đầu tiên sẽ để sót phần đuôi cũ mỗi khi chạy lại trong ngày (gây lặp nội dung đuôi).
- **Giải pháp đề xuất (Chờ anh APPROVE):**
  Tìm vị trí thẻ `<h2` kế tiếp sau `markerIdx` (và lùi lại trước thẻ `<hr/>` / `<hr />` ngăn cách nếu có) làm ranh giới kết thúc section ngày; nếu là section cuối cùng của trang (không có `<h2` kế tiếp) thì thay thế đến hết trang (hoặc hết `</table>`).

---

### 2. File [`centralized-data-service/internal/service/confluence/confluence_runner.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/confluence/confluence_runner.go)

- **Dòng 157 (`resolveSingleReportPageTitle`):**
  `pageTitle, err := resolveSingleReportPageTitle(req.TitleTemplate, data)`
- **Dòng 173–179 (`CreateOrUpdateChildPage`):**
  Truyền `Title: pageTitle` (đã loại bỏ `{{.ReportDate}}`) và `SectionDate: reportDateStr` (`YYYY-MM-DD`).
- **Dòng 200–254 (`resolveSingleReportPageTitle`):**
  Xử lý đầy đủ các biến thể `{{.ReportDate}}`, `{{ .ReportDate }}`, `{{- .ReportDate -}}`, dọn sạch dấu gạch ngang thừa (`" - "` ở đầu/cuối hoặc `" - - "`), và render template với `dataSinglePage.ReportDate = ""`.
- **Dòng 626 (`defaultTitleTemplate`):**
  `const defaultTitleTemplate = "[Daily Data Check] {{.ServiceName}}"` (đã bỏ `{{.ReportDate}}`).
- **Đánh giá:** Logic trong `confluence_runner.go` **chính xác, chặt chẽ, không có lỗi logic**.

---

### 3. File [`cdc-cms-web/src/pages/ActivityManager.tsx`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/ActivityManager.tsx)

#### Phát hiện C4 (Nghiêm trọng trên UI - Cần sửa): Ngay sát các dòng vừa sửa `confluence_title_template`, UI vẫn hardcode `space_key` (`a463` sai 1 ký tự) và `parent_page_id` (`463833042` cũ) gây lỗi 404!
- **Kiểm tra từng dòng tại [`cdc-cms-web/src/pages/ActivityManager.tsx`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/ActivityManager.tsx):**
  1. **Dòng 1214–1216 (`handleOpenCreate`):**
     ```tsx
     confluence_space_key: '~712020a3dcebcbe93646fea463f119dd38b08c',
     confluence_parent_page_id: '463833042',
     confluence_title_template: '[Daily Data Check] Core Trans Proxy History Service',
     ```
  2. **Dòng 1295–1301 (`handleOpenEdit`):**
     ```tsx
     confluence_space_key:
       record.params?.confluence?.space_key ?? '~712020a3dcebcbe93646fea463f119dd38b08c',
     confluence_parent_page_id:
       record.params?.confluence?.parent_page_id ?? '463833042',
     confluence_title_template:
       record.params?.confluence?.title_template ??
       '[Daily Data Check] Core Trans Proxy History Service',
     ```
  3. **Dòng 1369–1371 (`handleSubmit`):**
     ```tsx
     space_key: values.confluence_space_key || '~712020a3dcebcbe93646fea463f119dd38b08c',
     parent_page_id: values.confluence_parent_page_id || '463833042',
     title_template:
       values.confluence_title_template || '[Daily Data Check] Core Trans Proxy History Service',
     ```
  4. **Dòng 1736–1738 (`initialValues` của Form) và các placeholder tại dòng `1767`, `1797`, `1799`, `1966`:**
     Đều đang để `'~712020a3dcebcbe93646fea463f119dd38b08c'` (`a463`) và `'463833042'`.
- **Tại sao đây là lỗi nghiêm trọng do "Điểm mù dòng kế cận" (`#adjacent-default-blindspot`):**
  - Ở đầu phiên, chính anh đã hỏi: *"~712020a3dcebcbe93646fea464f119dd38b08c và page này 500432919 thì sao"* và trong [`confluence_runner.go:634-637`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/confluence/confluence_runner.go#L634-L637) đã sửa thành:
    - `defaultSpaceKey = "~712020a3dcebcbe93646fea464f119dd38b08c"` (`a464`)
    - `defaultParentPageID = "500432919"`
  - Thế nhưng ở lượt trước, khi vào `ActivityManager.tsx` sửa dòng `confluence_title_template`, Agent lại bỏ qua ngay 2 dòng nằm sát phía trên nó (`confluence_space_key` đang là `a463` và `confluence_parent_page_id` đang là `463833042`). Hậu quả là bất kỳ ai tạo lịch mới trên giao diện CMS mà để mặc định sẽ gửi `a463` và `463833042` xuống DB, ghi đè lên default đúng của Backend và gây lỗi `404 Not Found`!
  - Ngoài ra, ở `handleOpenEdit` (dòng 1299), nếu mở một lịch cũ đang lưu `title_template: "[Daily Data Check] {{.ReportDate}} - Core Trans Proxy History Service"` trong DB và bấm Save lại, UI vẫn giữ nguyên chuỗi có `{{.ReportDate}}` trên ô input.
- **Giải pháp đề xuất (Chờ anh APPROVE):**
  1. Thay toàn bộ `'~712020a3dcebcbe93646fea463f119dd38b08c'` -> `'~712020a3dcebcbe93646fea464f119dd38b08c'` (`a464`) và `'463833042'` -> `'500432919'` tại tất cả các dòng trong `ActivityManager.tsx`.
  2. Trong `handleOpenEdit`, tự động chuẩn hóa `record.params?.confluence?.title_template` (bỏ `{{.ReportDate}} - `) để khi anh mở Modal Edit lịch cũ trên UI là thấy ngay tiêu đề chuẩn mới.

---

### 4. Hai Hồ sơ Giải pháp Bị Bỏ Sót Cần Thực Thi Bổ Sung (Sau khi anh APPROVE)

#### A. Hồ sơ 1: Vá lỗi `sql: database is closed` & `redis: client is closed` khi dừng `cdc-cms-service` ([`09_tasks_solution_cms_shutdown_db_closed_race.md`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/fix-recon-daily-flow-cud-2026-10-06/09_tasks_solution_cms_shutdown_db_closed_race.md))
1. **File [`cdc-cms-service/internal/server/server.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/server/server.go#L393-L446):**
   - Dùng `var bgWg sync.WaitGroup` bọc 4 goroutine chạy nền (`auditLogger`, `alertResolver`, `stuckJobReaper`, `healthCollector`).
   - Khi `ctx.Done()` (hoặc khi `httpServer.ListenAndServe` trả về lỗi `bind: address already in use`), sau khi `cancel()`, **bắt buộc gọi `bgWg.Wait()`** (với timeout bảo vệ 5s) để toàn bộ truy vấn SQL/Redis đang dở của `healthCollector` kết thúc hẳn rồi mới gọi `s.redis.Close()` và `sqlDB.Close()`.
2. **File [`cdc-cms-service/internal/infra/observability/system_health_collector.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/observability/system_health_collector.go#L225-L248):**
   - Trong `collectOnce(parent context.Context)`, ngay sau `g.Wait()`, kiểm tra:
     ```go
     if parent.Err() != nil {
         return parent.Err()
     }
     ```
     Nếu `parent` context đã bị hủy (server đang tắt), thoát ngay lập tức, **không gọi tiếp** `evalAndFireAlerts(ctx)` và `c.redis.Set(...)` bằng `context.Background()`.

#### B. Hồ sơ 2: Vá lỗi hiển thị lệch `-7h` (`Lần chạy cuối` / `Lần chạy kế`) trên môi trường Testing ([`09_tasks_solution_schedule_timezone_testing.md`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/fix-recon-daily-flow-cud-2026-10-06/09_tasks_solution_schedule_timezone_testing.md))
1. **File [`cdc-cms-web/src/utils/datetime.ts`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/utils/datetime.ts#L5-L18):**
   - Sửa `formatNaiveDateTime`: Nếu chuỗi đầu vào có hậu tố múi giờ (`Z` hoặc `[+-]\d{2}:?\d{2}`), parse bằng `new Date(trimmed)` và format theo múi giờ `'Asia/Ho_Chi_Minh'` (`Intl.DateTimeFormat('en-GB', { timeZone: 'Asia/Ho_Chi_Minh', ... })`) để `"2026-10-15T09:46:33Z"` hiển thị đúng thành **`15/10/2026 16:46:33`** và `"2026-10-15T17:05:00Z"` hiển thị đúng thành **`16/10/2026 00:05:00`**.
2. **File [`cdc-cms-service/internal/infra/persistence/scheduler/worker_schedule_read_repo_gorm.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/scheduler/worker_schedule_read_repo_gorm.go#L116-L147):**
   - Trong `toRecord`, chuyển đổi `LastRunAt`, `NextRunAt`, `CreatedAt`, `UpdatedAt` sang múi giờ `Asia/Ho_Chi_Minh` (`.In(locHCM)`) trước khi trả về JSON, đảm bảo dù container chạy ở `TZ=UTC` trên Testing thì JSON xuất ra vẫn mang offset `+07:00` đồng nhất.

---

## IV. KẾT QUẢ QC ĐỘC LẬP QUA `/laya_router` (`http://127.0.0.1:8200/v1/predict`)

Đã thực thi script [`run_laya_confluence_append_audit.py`](file:///Users/trainguyen/.gemini/antigravity/brain/d0a1f679-e07d-4cf3-8545-0ca5db1e83f1/scratch/run_laya_confluence_append_audit.py) gọi trực tiếp `/laya_router` (dữ liệu thô lưu tại [`06_laya_raw_confluence_append_audit.json`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/fix-recon-daily-flow-cud-2026-10-06/06_laya_raw_confluence_append_audit.json)):

| Mã Kịch Bản QC | Nội Dung Đánh Giá | R2 Verdict (`prob` / `conf`) | R3 Decision (`prob` / `conf`) | Phân Tích Phản Biện Độc Lập |
| :--- | :--- | :--- | :--- | :--- |
| **`NEG_CONTROL_MOCK_TEST_AS_PRODUCTION_DONE`** *(Negative Control)* | Nhận định sai cố ý: *"Chỉ cần pass unit test với `httptest.NewServer` là đủ kết luận tính năng chạy 100% trên thực tế, không cần restart worker `:8082`"* | `CORRECT` (`0.6403` / **`conf=0.2600`**) | `YES` (`0.8140` / **`conf=0.3070`**) | **Cảnh báo quan trọng:** Dù `confidence` sụt giảm mạnh xuống mức rất thấp (`0.26`), `/laya_router` vẫn bị đánh lừa bởi văn phong tự tin của mệnh đề giả (False Positive). Điều này chứng minh **tuyệt đối không được lấy `/laya_router` hay Unit Test Mock làm cái cớ thay thế cho kiểm chứng Runtime thật**. |
| **`AUDIT_C1_CONFLUENCE_SINGLE_PAGE_APPEND_AND_403_LOGGING`** | Kiến trúc gom báo cáo ngày vào 1 trang cố định (`resolveSingleReportPageTitle` + `expand=version,body.storage` + `MergeDailySectionXHTML` theo mốc `(YYYY-MM-DD)</h2>`) | **`CORRECT`** (`0.8302` / `conf=0.4785`) | **`YES`** (`0.7700` / `conf=0.2220`) | Hướng thiết kế gom 1 trang và định vị bằng mốc ASCII `(YYYY-MM-DD)</h2>` là đúng đắn, nhưng `conf` ở mức trung bình vì còn thiếu các chốt chặn ở `AUDIT_C2` và `AUDIT_C3`. |
| **`AUDIT_C2_UPDATE_PAGE_ANCESTORS_AND_RETRY_ERROR_PRESERVATION`** | Phát hiện `updatePage` thiếu `"ancestors"` trong payload `PUT` (nguy cơ làm rớt trang con khỏi `500432919` ra Space Root) và `doWithRetry` cần giữ `lastErr` | **`CORRECT`** (**`0.9780`** / **`conf=0.8904`**) | **`YES`** (**`0.9724`** / **`conf=0.8179`**) | **Độ đồng thuận cao nhất (`97.8%` CORRECT, `97.24%` YES, `conf=0.89`):** Xác nhận phát hiện lỗi thiếu `"ancestors"` trong `updatePage` và bảo toàn lỗi trong `doWithRetry` là hoàn toàn chính xác và bắt buộc phải vá. |
| **`AUDIT_C3_FRONTEND_TYPO_DEFAULTS_IN_ACTIVITY_MANAGER`** | Phát hiện `ActivityManager.tsx` vẫn giữ default typo `a463` và `463833042` ngay sát dòng `confluence_title_template` vừa sửa | `INCORRECT` (`0.5784` / `conf=0.1168`) *(R2 chấm đoạn code chứa `a463` là sai)* | **`YES`** (**`0.8920`** / `conf=0.5062`) | R2 đánh giá đoạn `code_context` của `ActivityManager.tsx` chứa `a463`/`463833042` là `INCORRECT`, và R3 đồng thuận **`89.2% YES`** phải sửa dứt điểm các dòng default typo này trên Frontend. |
| **`AUDIT_C4_OMITTED_APPROVED_TASKS_SHUTDOWN_RACE_AND_TIMEZONE`** | Phát hiện bỏ sót 2 hồ sơ `09_tasks_solution_cms_shutdown_db_closed_race.md` và `09_tasks_solution_schedule_timezone_testing.md` khi thực thi lệnh `"APPROVE toàn bộ"` | **`CORRECT`** (`0.7524` / `conf=0.3349`) | **`YES`** (**`0.9349`** / **`conf=0.6527`**) | Đồng thuận **`93.49% YES`**: Xác nhận phải báo cáo trung thực việc bỏ sót 2 hồ sơ này và thực thi đầy đủ ngay khi anh xác nhận. |

---

## V. VÒNG LẶP PHẢN TỈNH & KHẮC PHỤC (SELF-IMPROVEMENT LOOP)

Đã rút trích nguyên nhân gốc rễ và ghi nhận bài học mới vào [`/Users/trainguyen/Documents/work/agent/memory/global/lessons.md`](file:///Users/trainguyen/Documents/work/agent/memory/global/lessons.md#L27-L33) (số dòng tăng từ `1875 -> 1882`, tuân thủ chuẩn Canonical 5 phần và Rule #6/#17):

- **Lesson `[2026-10-16]`:** *Bỏ sót hồ sơ giải pháp khi nhận lệnh Batch `APPROVE`, thiếu trường `ancestors` khi `PUT` cập nhật trang con Confluence, bỏ sót giá trị mặc định lỗi ở các dòng kế cận trên UI, và chưa tách bạch rõ ranh giới giữa Mock Test (`httptest.NewServer`) với Runtime thực tế.*
- **4 Quy tắc khắc phục bắt buộc:**
  1. **`#batch-approve-scope-omission`**: Khi nhận lệnh `"APPROVE toàn bộ"`, bắt buộc liệt kê toàn bộ các file `09_tasks_solution_*.md` đang ở trạng thái `PROPOSED` trong workspace thành bảng đối chiếu trước khi code, không được bỏ sót bất kỳ hồ sơ nào.
  2. **`#confluence-put-ancestors`**: Khi gọi `PUT /wiki/rest/api/content/{id}` cập nhật trang con Confluence, bắt buộc truyền đầy đủ `"ancestors": [{"id": parentPageID}]` giống như lúc `POST` tạo mới để không làm trang con bị tách khỏi trang cha.
  3. **`#adjacent-default-blindspot`**: Khi sửa một trường default trong cụm config trên UI (`ActivityManager.tsx`), bắt buộc kiểm tra toàn bộ các trường anh em nằm sát nó (`space_key`, `parent_page_id`, `title_template`) để đồng bộ 100% với Backend.
  4. **`#mock-vs-runtime-boundary`**: Báo cáo rõ ràng đâu là kết quả của `httptest.NewServer` (Mock) và đâu là trạng thái của tiến trình thật đang chạy (`:8082`, `:8083`, `goopay.atlassian.net`).

---

## VI. ĐỀ XUẤT HÀNH ĐỘNG & KẾ HOẠCH PHỐI HỢP TRACE RUNTIME VALUE CÙNG ANH

Để giải quyết triệt để **100% các lỗ hổng phát hiện qua Audit** và kiểm chứng trên **Runtime thực tế (không dùng Mockup)**, em đề xuất lộ trình 2 bước rõ ràng như sau:

### Bước 1: Anh cho lệnh `APPROVE` để em thực thi dứt điểm 4 hạng mục code còn thiếu/lỗi
1. **Vá `pkgs/confluence/client.go` (Findings C1, C2, C3):**
   - Thêm `"ancestors": [{"id": req.ParentPageID}]` vào `updatePage` (`PUT`).
   - Lưu `lastErr` chứa HTTP status + body trước khi đóng `resp.Body` trong `doWithRetry`, và không retry vô ích 14s nếu `403` trả về `"caller cannot access Confluence"`.
   - Cải tiến `MergeDailySectionXHTML` nhận diện ranh giới `<h2` kế tiếp (trước `</table>`).
2. **Vá `cdc-cms-web/src/pages/ActivityManager.tsx` (Finding C4):**
   - Đổi toàn bộ default `space_key` từ `...a463...` sang `~712020a3dcebcbe93646fea464f119dd38b08c` (`a464`) và `parent_page_id` từ `'463833042'` sang `'500432919'`.
   - Tự động strip `{{.ReportDate}}` khi mở Modal Edit lịch trình cũ + chạy `npm run build` kiểm chứng.
3. **Thực thi Hồ sơ bị bỏ sót #1 ([`09_tasks_solution_cms_shutdown_db_closed_race.md`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/fix-recon-daily-flow-cud-2026-10-06/09_tasks_solution_cms_shutdown_db_closed_race.md)):**
   - Thêm `sync.WaitGroup` vào `cdc-cms-service/internal/server/server.go` và kiểm tra `if parent.Err() != nil` trong `system_health_collector.go`.
4. **Thực thi Hồ sơ bị bỏ sót #2 ([`09_tasks_solution_schedule_timezone_testing.md`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/fix-recon-daily-flow-cud-2026-10-06/09_tasks_solution_schedule_timezone_testing.md)):**
   - Sửa `cdc-cms-web/src/utils/datetime.ts` và `worker_schedule_read_repo_gorm.go` chuyển đổi chuẩn múi giờ `Asia/Ho_Chi_Minh` (`+07:00`).

### Bước 2: Phối hợp cùng anh Trace Runtime Value Thực Tế (Không Mockup)
1. **Restart & Trace trên Local (`:8082` `centralized-data-service` và `:8083` `cdc-cms-service`):**
   - Sau khi anh `APPROVE`, em sẽ build lại và restart 2 tiến trình `:8082` (hiện đang chạy bản cũ từ `15/10 15:52`) và `:8083` (hiện đang chạy bản cũ từ `15/10 16:04`).
   - Kiểm tra trực tiếp:
     - Gọi `GET http://localhost:8083/api/worker-schedule` để trace giá trị JSON thật của `last_run_at` và `next_run_at` (mang offset `+07:00`).
     - Gửi tín hiệu `SIGTERM` thử nghiệm vào tiến trình `cdc-cms-service` để trace log shutdown thật — xác nhận **0 dòng lỗi** `sql: database is closed` và `redis: client is closed`.
2. **Phối hợp cùng anh kiểm tra quyền & Trace Live Confluence (`https://goopay.atlassian.net`):**
   - Hiện tại tài khoản `trai.nguyen@goopay.vn` trên `goopay.atlassian.net` đang bị mất group truy cập sản phẩm Confluence (chỉ còn group `"jira-users-goopay"`, nên API `/wiki/rest/api/content` trả về `403 "Request rejected because caller cannot access Confluence"`).
   - **Cần anh phối hợp:**
     - **Cách A:** Anh (hoặc Admin Atlassian) cấp lại quyền truy cập Confluence (`cf_user_gp` / `cf_def_GP`) cho `trai.nguyen@goopay.vn`, **HOẶC**
     - **Cách B:** Anh cung cấp email + API token của tài khoản Service Account đang có quyền trên Space `~712020a3dcebcbe93646fea464f119dd38b08c`.
   - Ngay khi tài khoản có quyền trở lại, em sẽ kích hoạt chạy thật (`POST /api/v1/reports/confluence/daily`) cho 2 ngày liên tiếp (`2026-10-14` và `2026-10-15`) dưới trang cha `500432919` và đọc trực tiếp nội dung trang thật từ Atlassian API về để anh kiểm chứng kết quả Append thực tế!
