# 🛡️ BÁO CÁO AUDIT TOÀN DIỆN & PHẢN BIỆN CHUYÊN SÂU: CONFLUENCE DAILY REPORT
**Task:** Báo cáo đối soát dữ liệu hàng ngày bắn lên Atlassian Confluence (Confluence Daily Report Runner & Client)  
**Workspace:** `fix-recon-daily-flow-cud-2026-10-06`  
**Ngày thực hiện:** 2026-10-13  
**Audit Entity:** Brain & Chief QA Architect (/laya_router perspective)  
**Tiêu chuẩn áp dụng:** Hiến pháp GEMINI.md (Rule #0, Rule #4, Rule #5, Rule #12, Rule #14 Gate G1–G8)

---

## I. TỰ SOI XÉT TRUNG THỰC & RÀ SOÁT BÁO CÁO (TRUTH IN TELEMETRY)

### 1. Hiện trạng kiểm thử thực tế (No Mockup - No Hallucination)
* **Những gì ĐÃ CHẠY THẬT:**
  1. `centralized-data-service`: Unit test `pkgs/confluence/client_test.go` $\rightarrow$ PASS (Chỉ test hàm `html.EscapeString` của Go).
  2. `centralized-data-service`: Unit test `internal/service/confluence/confluence_runner_test.go` $\rightarrow$ PASS (Chỉ test output chuỗi XHTML sinh ra từ `buildConfluenceXHTML`).
  3. `centralized-data-service`: Biên dịch `go build -v -o /dev/null ./cmd/worker` $\rightarrow$ PASS Exit Code 0.
  4. `cdc-cms-service`: Unit test `MaskSensitiveScheduleParams` $\rightarrow$ PASS (Test regex mask token).

* **Những gì CHƯA CHẠY (Thừa nhận trung thực 100%):**
  1. **CHƯA test live với Atlassian Confluence Cloud:** Chưa bao giờ gửi một HTTP request thật nào đến `https://goopay.atlassian.net/wiki/rest/api/content` với Email và API Token thật của User.
  2. **CHƯA test tích hợp Client với mock HTTP server:** Chưa có test kịch bản API Confluence trả về 401 Unauthorized, 403 Forbidden, 429 Rate Limit, hay 400 Bad Request.
  3. **CHƯA kích hoạt Confluence Runner trên scheduler runtime:** Chưa từng để scheduler tự động trigger hoặc bấm "Run Now" một job Confluence thật trên môi trường chạy để kiểm tra luồng đọc DB `daily_source_counts` $\rightarrow$ `recon_hourly_pipeline_stats` $\rightarrow$ Bắn API.
  4. **KẾT LUẬN:** Nếu tuyên bố Confluence Daily Report "đã hoàn tất ngon lành" là **HOÀN TOÀN SAI LỰC VÀ BÁO CÁO LÁO**. Thực tế mới chỉ dừng ở mức: Code đã biên dịch xong và test chuỗi HTML in-memory. **Bắt buộc phải phối hợp cùng User để trace runtime value với credentials thật!**

---

## II. AUDIT CHI TIẾT TỪNG DÒNG MÃ NGUỒN CỦA CONFLUENCE MODULE

### 1. File `centralized-data-service/pkgs/confluence/client.go`

#### A. Hàm `CreateOrUpdateChildPage` (Dòng 58-75)
```go
func (c *Client) CreateOrUpdateChildPage(ctx context.Context, req CreateChildPageRequest) (string, error) {
    ...
    existingPage, err := c.findPageByTitle(ctx, req.BaseURL, req.SpaceKey, req.Title, authHeader)
    if err == nil && existingPage != nil {
        return c.updatePage(ctx, req, existingPage.ID, existingPage.Version.Number+1, authHeader)
    }

    return c.createPage(ctx, req, authHeader)
}
```
* **🚨 TỬ HUYỆT 1 (Nuốt lỗi tìm kiếm dẫn đến lỗi 400 Duplicate Title):**
  - Nếu `findPageByTitle` gặp lỗi mạng (timeout), lỗi 5xx của Atlassian, hoặc 429 Rate Limit $\rightarrow$ `err != nil`.
  - Điều kiện `if err == nil && existingPage != nil` bị bỏ qua!
  - Luồng code tự động rơi xuống `c.createPage(ctx, req, authHeader)`.
  - Lúc này, nếu trang **thực sự đã tồn tại** trên Confluence, lệnh `POST` sẽ bị Atlassian từ chối với lỗi:  
    `confluence api error (status 400): A page with this title already exists in the given space`.
  - **Khắc phục chuẩn:**
    ```go
    existingPage, err := c.findPageByTitle(ctx, req.BaseURL, req.SpaceKey, req.Title, authHeader)
    if err != nil {
        return "", fmt.Errorf("check existing page by title failed: %w", err)
    }
    if existingPage != nil {
        return c.updatePage(ctx, req, existingPage.ID, existingPage.Version.Number+1, authHeader)
    }
    return c.createPage(ctx, req, authHeader)
    ```

#### B. Hàm `createPage` (Dòng 109-164)
```go
payload := map[string]interface{}{
    "type":  "page",
    "title": req.Title,
    "ancestors": []map[string]interface{}{
        {"id": req.ParentPageID},
    },
    ...
}
```
* **🚨 TỬ HUYỆT 2 (`ParentPageID` rỗng gây 400 Bad Request):**
  - Nếu người dùng cấu hình không có `parent_page_id` hoặc để trống `""`, payload gửi lên sẽ là `ancestors: [{"id": ""}]`.
  - Confluence REST API v1 sẽ trả ngay lỗi `400 Bad Request` ("Invalid ancestor id").
  - **Khắc phục chuẩn:** Chỉ bổ sung key `"ancestors"` vào payload khi `strings.TrimSpace(req.ParentPageID) != ""`.

---

### 2. File `centralized-data-service/internal/service/confluence/confluence_runner.go`

#### A. Truy vấn `daily_source_counts` (Dòng 136-154)
```go
var sourceCount int64
countQuery := r.dbControl.Table("cdc_system.daily_source_counts").
    Select("creation_count").
    Where("object_name = ? AND report_date = ?", tbl.ObjectName, reportDateStr)

if tbl.LayerType != "" {
    countQuery = countQuery.Where("layer_type = ?", tbl.LayerType)
}
if tbl.ConnectionCode != "" {
    countQuery = countQuery.Where("connection_code = ?", tbl.ConnectionCode)
}
if tbl.ContainerNamespace != "" {
    countQuery = countQuery.Where("container_namespace = ?", tbl.ContainerNamespace)
}
```
* **🚨 TỬ HUYỆT 3 (Lệch pha Metadata giữa UI và Runner làm mất sạch Source Count):**
  - Ở Frontend `ActivityManager.tsx` (dòng 508-510 & 639-641), khi người dùng chọn danh sách bảng báo cáo Confluence:
    ```tsx
    return {
        layer_type: 'master',
        connection_code: 'default_master',
        container_namespace: found?.source_db || 'public',
        object_name: tblName,
        ...
    };
    ```
    Frontend gán cứng `layer_type: 'master'` và `connection_code: 'default_master'`.
  - TRONG KHI ĐÓ, ở Giai đoạn 1 (`source_count`), số đếm bản ghi ngày tạo được lưu vào bảng `daily_source_counts` với:
    `layer_type = 'source'` (hoặc layer thực tế của nguồn dữ liệu), và `connection_code` là mã kết nối của DB nguồn (ví dụ: `core_trans_proxy_history_service`)!
  - HẬU QUẢ: Câu query của Confluence Runner lọc `WHERE layer_type = 'master' AND connection_code = 'default_master'`. **Nó sẽ KHÔNG BAO GIỜ tìm thấy bản ghi đếm nguồn!**
  - Báo cáo Confluence luôn hiển thị cột Source Count là: `"Chưa đếm"` và trạng thái `"🟡 CDC ACTIVE (Chưa có Source Count)"`, dù job đếm nguồn 00:05 đã chạy thành công trước đó!
  - **Khắc phục chuẩn:**
    - Không lọc cứng theo `tbl.LayerType` hay `tbl.ConnectionCode` khi đối chiếu số đếm nguồn, HOẶC:
    - Query `daily_source_counts` theo `object_name = ? AND report_date = ? AND layer_type = 'source'` (hoặc order by `calculated_at DESC LIMIT 1`).

#### B. Đánh giá trạng thái Lệch (Dòng 166-182)
```go
if hasSourceCount && sourceCount == row.MasterUpsert {
    row.Status = "🟢 MATCH (Khớp 100%)"
} else if !hasSourceCount {
    ...
} else {
    diff := sourceCount - row.MasterUpsert
    if diff > 0 {
        row.Status = fmt.Sprintf("🔴 SOURCE > MASTER (+%d)", diff)
    } else {
        row.Status = fmt.Sprintf("🔴 MASTER > SOURCE (%d)", diff)
    }
}
```
* **🚨 TỬ HUYỆT 4 (Hiểu sai bản chất chỉ số CDC: Insert vs Update):**
  - `sourceCount` là số bản ghi **sinh mới trong ngày hôm qua** (dựa vào `created_at`).
  - `MasterUpsert` từ `recon_hourly_pipeline_stats` là **tổng toàn bộ các sự kiện ghi vào Master**, bao gồm cả INSERT (bản ghi mới) và UPDATE (cập nhật trạng thái trên các bản ghi cũ đã tạo từ tuần trước, tháng trước).
  - Nếu trong ngày hôm qua có 1.000 đơn hàng tạo mới (`sourceCount = 1000`), nhưng có thêm 500 đơn hàng cũ được cập nhật trạng thái (`MasterUpsert = 1500`), thuật toán sẽ phán ngay: `🔴 MASTER > SOURCE (-500)`.
  - Người vận hành nhìn vào sẽ hoảng hốt tưởng CDC bị nhân đôi dữ liệu (phantom duplicates)!
  - **Khắc phục chuẩn:**
    - Cần bóc tách rõ trong báo cáo: Cột `Master Inserts` (đối soát trực tiếp với `Source Creation Count`) và Cột `Master Updates` (sự kiện biến động).
    - Câu so sánh trạng thái: Đối chiếu `Source Count` với `Master Inserts`. Nếu `Master Inserts == Source Count` $\rightarrow$ `🟢 MATCH (Tạo mới khớp 100%)`.

#### C. Xử lý khi không có bảng nào được cấu hình (Dòng 106-116)
* Nếu `params.SelectedTables` rỗng và `sched.TargetTable` cũng rỗng $\rightarrow$ `tables` rỗng $\rightarrow$ Runner sinh ra một trang Confluence có bảng 0 dòng và vẫn gửi lên Atlassian!
* **Khắc phục:** Fail-fast nếu không có bảng nào được chỉ định, cập nhật `last_error = "no tables selected"`.

---

### 3. File `cdc-cms-service/internal/api/scheduler/schedule_handler.go`

```go
func MaskSensitiveScheduleParams(params json.RawMessage) json.RawMessage {
    ...
    if conf, ok := m["confluence"].(map[string]interface{}); ok {
        if token, ok := conf["api_token"].(string); ok && len(token) > 8 {
            conf["api_token"] = token[:6] + "...****"
        }
    }
    ...
}
```
* **Tử huyệt liên đới đã phát hiện ở turn trước:** Hàm mask này hoạt động tốt ở đầu ra `GET`, nhưng vì đầu vào `PATCH` chưa có bộ lọc giữ lại token cũ, nên khi UI gửi lại form có token mask, backend sẽ ghi đè token mask vào DB làm tê liệt runner vĩnh viễn.

---

## III. TỔNG HỢP CÁC ĐIỂM CẦN VÁ NGAY CHO CONFLUENCE MODULE

| STT | Vị trí file | Lỗ hổng phát hiện | Giải pháp kỹ thuật khắc phục |
| :---: | :--- | :--- | :--- |
| **1** | `pkgs/confluence/client.go` | Nuốt lỗi `findPageByTitle` dẫn đến cố tình `createPage` gây lỗi 400 Duplicate Title. | Kiểm tra `if err != nil { return "", err }` ngay sau khi tìm kiếm. |
| **2** | `pkgs/confluence/client.go` | `ParentPageID` rỗng vẫn truyền vào `ancestors` gây lỗi 400. | Chỉ gán `ancestors` nếu `ParentPageID != ""`. |
| **3** | `confluence_runner.go` | Query `daily_source_counts` bị lệch `layer_type: 'master'` làm không bao giờ lấy được số đếm nguồn. | Query theo `object_name = ? AND report_date = ? AND layer_type = 'source'`. |
| **4** | `confluence_runner.go` | Đánh đồng `MasterUpsert` (gồm cả update) với `Source Creation Count` gây báo động giả `MASTER > SOURCE`. | Chú thích rõ hoặc bóc tách `master_insert` đối chiếu với creation count. |
| **5** | `confluence_runner.go` | Cho phép chạy và bắn trang Confluence rỗng khi `tables` = 0. | Bổ sung check fail-fast: nếu không có table nào thì ngắt sớm và log error. |
| **6** | `confluence/client_test.go` | Test hời hợt, chỉ test `html.EscapeString`, không test HTTP flow. | Viết unit test mock HTTP Server cho các kịch bản Get/Post/Put Confluence. |

---

## IV. KẾ HOẠCH PHỐI HỢP TRACE RUNTIME VALUE VỚI USER

Để nghiệm thu trung thực, không suy diễn:
1. **Vá dứt điểm 6 điểm kỹ thuật trên.**
2. **Cùng User chuẩn bị thông tin thật:**
   - Atlassian Confluence Base URL: `https://goopay.atlassian.net`
   - Space Key: `~712020a3dcebcbe93646fea463f119dd38b08c`
   - Parent Page ID: `463833042` (Core Trans Proxy History Service)
   - User Email & Atlassian API Token thật.
3. **Thực hiện Live Trigger qua "Run Now":**
   - Kích hoạt job Confluence $\rightarrow$ Quan sát Worker log xem HTTP POST/PUT gọi sang Atlassian có trả về 200/201 không.
   - Nhận Page URL trả về và mở trực tiếp trên trình duyệt xem trang Confluence thật được tạo ra có format đẹp mắt, đúng số liệu và không bị lỗi XHTML không.
