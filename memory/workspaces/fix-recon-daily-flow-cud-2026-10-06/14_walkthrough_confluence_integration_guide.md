# Hướng Dẫn Chi Tiết: Xem Báo Cáo Trên Confluence & Các Bước Tích Hợp (Confluence Integration Guide)

- **Ngày khởi tạo:** 2026-10-12
- **Tác giả:** Brain (Architect)
- **Mục tiêu:** Hướng dẫn chi tiết vị trí xem báo cáo trên `atlassian.net/wiki`, cấu trúc cây thư mục, cách lấy API Token, Space Key, Parent Page ID và luồng tích hợp kỹ thuật.

---

## PHẦN 1: TRÊN ATLASSIAN.NET/WIKI THÌ XEM Ở ĐÂU?

### 1. Cấu Trúc Cây Thư Mục Trang (Page Tree Structure)
Trên Confluence, các trang được tổ chức theo cấu trúc phân cấp (Cha - Con) nằm trong một **Space** (ví dụ Space: `Data Platform` hoặc `Data Engineering` có key là `DATA`):

```text
📁 Space: DATA (Data Platform)
│
└── 📄 [Trang Cha] Daily Data Quality & Reconciliation Reports
    │   (Trang tổng quan: chứa Macro tự động liệt kê tất cả các ngày)
    │
    ├── 📄 Daily Report - 2026-10-12 [🟢 HEALTHY]
    ├── 📄 Daily Report - 2026-10-11 [🟢 HEALTHY]
    ├── 📄 Daily Report - 2026-10-10 [🔴 DRIFT DETECTED]
    └── 📄 Daily Report - 2026-10-09 [🟢 HEALTHY]
```

### 2. Đường Dẫn (URL) Xem Trang
- **URL trang danh mục tổng (Parent Page):**
  `https://<your-company>.atlassian.net/wiki/spaces/<SPACE_KEY>/pages/<PARENT_PAGE_ID>`
  *(Ví dụ: `https://company.atlassian.net/wiki/spaces/DATA/pages/123456789/Daily+Data+Quality+Reports`)*
- **URL trang báo cáo ngày cụ thể:**
  `https://<your-company>.atlassian.net/wiki/spaces/<SPACE_KEY>/pages/<PAGE_ID>/Daily+Report+-+2026-10-12`

### 3. Trang Cha Hiển Thị Bằng Macro Nào Cho Đẹp & Dễ Theo Dõi?
Trên trang cha `Daily Data Quality & Reconciliation Reports`, chúng ta gắn 1 trong 2 macro có sẵn của Confluence:
1. **Macro `Children Display`**: Tự động liệt kê danh sách tất cả các ngày kèm liên kết click xem ngay.
2. **Macro `Page Properties Report`**: Tự động tạo bảng tổng hợp tất cả các ngày:
   | Ngày | Tổng Records | Tỷ Lệ Toàn Vẹn | Trạng Thái | Link Chi Tiết |
   | :--- | :--- | :--- | :--- | :--- |
   | 2026-10-12 | 1,250,400 | 100% | 🟢 HEALTHY | [Xem báo cáo](#) |
   | 2026-10-11 | 980,200 | 99.8% | 🔴 DRIFT | [Xem báo cáo](#) |

---

## PHẦN 2: CÁCH TÍCH HỢP VÀO ATLASSIAN CONFLUENCE (STEP-BY-STEP)

Để hệ thống (`cdc-cms-service`) có thể tự động bắn bài lên Confluence, cần hoàn thành 4 bước cấu hình:

### Bước 1: Tạo Atlassian API Token
API Token đóng vai trò là "mật khẩu ứng dụng" để bot/backend gọi Confluence REST API mà không cần gõ mật khẩu thật:
1. Đăng nhập tài khoản Atlassian (tài khoản cá nhân hoặc email tài khoản bot của team: vd `cdc-bot@company.com`).
2. Truy cập trực tiếp link quản lý bảo mật của Atlassian:
   👉 **https://id.atlassian.com/manage-profile/security/api-tokens**
3. Bấm nút **"Create API token"**.
4. Đặt tên (Label): ví dụ `cdc-daily-report-bot`.
5. Bấm **Create** $\rightarrow$ Hệ thống sinh ra chuỗi mã (ví dụ: `ATATT3xFfGF0...`).
6. **Copy và lưu lại ngay** (chuỗi này chỉ hiển thị 1 lần duy nhất).

---

### Bước 2: Xác Định `Space Key` và `Parent Page ID` trên Confluence
Để bot biết **viết vào đâu**, cần 2 thông số:

#### 1. Lấy `Space Key`:
- Mở Space muốn chứa báo cáo trên Confluence (ví dụ Space tên là "Data Team").
- Nhìn lên thanh URL trình duyệt:
  `https://company.atlassian.net/wiki/spaces/DATA/overview` $\rightarrow$ **Space Key** chính là chữ `DATA`.
- Hoặc vào menu trái: **Space Settings** $\rightarrow$ xem dòng **Key**.

#### 2. Tạo Trang Cha & Lấy `Parent Page ID`:
- Trong Space đó, bấm nút **Create** tạo một trang mới đặt tên là:
  `Daily Data Quality & Reconciliation Reports`.
- Bấm **Publish** trang đó.
- Nhìn lên URL trình duyệt:
  `https://company.atlassian.net/wiki/spaces/DATA/pages/654321098/Daily+Data+Quality`
  $\rightarrow$ Dãy số `654321098` chính là **Parent Page ID** (Tất cả các báo cáo ngày sẽ được tạo tự động làm con của trang này).
- *(Mẹo khác: Bấm dấu ba chấm `...` ở góc phải trên của trang $\rightarrow$ chọn **Page Information** $\rightarrow$ xem dòng **Page ID**).*

---

### Bước 3: Đảm Bảo Tài Khoản Có Quyền Ghi Trong Space
- Vào Space Settings $\rightarrow$ **Space Permissions**.
- Kiểm tra tài khoản (email ở Bước 1) đã có quyền:
  - `Add` (Tạo trang mới).
  - `Delete` (Chỉnh sửa/Update trang).

---

### Bước 4: Khai Báo Biến Môi Trường Vào Backend Service
Trong file cấu hình `.env` của `cdc-cms-service`:
```env
# Confluence Integration Config
CONFLUENCE_ENABLED=true
CONFLUENCE_BASE_URL=https://your-company.atlassian.net
CONFLUENCE_USER_EMAIL=cdc-bot@your-company.com
CONFLUENCE_API_TOKEN=ATATT3xFfGF0...
CONFLUENCE_SPACE_KEY=DATA
CONFLUENCE_PARENT_PAGE_ID=654321098
```

---

## PHẦN 3: CƠ CHẾ GỌI REST API CONFLUENCE TỪ BACKEND

Confluence Cloud sử dụng **REST API v1** hoặc **v2**:
- Endpoint tạo trang: `POST https://<company>.atlassian.net/wiki/rest/api/content`
- Headers:
  - `Content-Type: application/json`
  - `Authorization: Basic <base64(email:api_token)>`
- Body JSON:
```json
{
  "type": "page",
  "title": "Daily Data Quality Report - 2026-10-12",
  "space": {
    "key": "DATA"
  },
  "ancestors": [
    {
      "id": "654321098"
    }
  ],
  "body": {
    "storage": {
      "value": "<h2>Tóm tắt luồng dữ liệu 2026-10-12</h2><table>...</table>",
      "representation": "storage"
    }
  }
}
```

Nếu trong ngày đó báo cáo được chạy lại (Re-run), API Client sẽ:
1. `GET /wiki/rest/api/content?title=...&spaceKey=DATA` để lấy `page_id` và `version.number`.
2. `PUT /wiki/rest/api/content/{page_id}` với `version.number + 1` để cập nhật đè nội dung mới nhất, không sinh ra trang rác bị trùng lặp.
