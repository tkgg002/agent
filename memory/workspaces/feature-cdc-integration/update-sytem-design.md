Trong hệ thống **Config-driven CDC** mà bạn đang xây dựng, cơ chế giao tiếp giữa 3 thành phần (Worker, CMS API, CMS FE) là sự kết hợp giữa **Event-Driven** (qua NATS) để xử lý dữ liệu thời gian thực và **RESTful API** để quản lý cấu hình.

Dưới đây là luồng giao tiếp chi tiết:

---

## 1. Luồng Event-Driven: Dữ liệu & Cấu hình (NATS JetStream)

Đây là "mạch máu" chính giúp hệ thống tự động hóa mà không cần n8n.

* **Dữ liệu từ Source:** Airbyte hoặc Debezium đẩy CDC events vào NATS theo các chủ đề (subjects) dạng `cdc.goopay.{source_db}.{table_name}`.
* **Core Worker (Go):** Sử dụng cơ chế **Pull Consumer** để kéo dữ liệu từ NATS JetStream. Worker xử lý batch (ví dụ 500 records) rồi upsert vào Postgres.
* **Thông báo Drift:** Khi `Schema Inspector` trong Worker phát hiện field mới, nó bắn một event vào subject `schema.drift.detected`.
* **Reload cấu hình:** Khi bạn nhấn Approve trên CMS, **CMS API** bắn một event `schema.config.reload` qua NATS. **Core Worker** lắng nghe event này để xóa Redis cache và tải lại Mapping Rules mới mà không cần restart.



---

## 2. Luồng RESTful: Quản trị & Giám sát (Fiber/Echo + React)

CMS FE và CMS API giao tiếp qua HTTP chuẩn để quản lý **Table Registry** và **Workflow**.

* **Registry Management:** CMS FE gửi yêu cầu (GET/POST/PATCH) đến CMS API để đăng ký table mới hoặc thay đổi `sync_engine` (Airbyte <-> Debezium).
* **Approval Workflow:** Khi FE gửi lệnh `POST /api/schema-changes/:id/approve`, CMS API thực hiện:
    1.  Chạy lệnh `ALTER TABLE` trong Postgres.
    2.  Cập nhật bảng `cdc_mapping_rules`.
    3.  Bắn event qua NATS cho Worker.
    4.  Gọi **Airbyte API** để đồng bộ lại schema nguồn.
* **Monitoring:** CMS FE lấy dữ liệu thống kê (stats, history) từ CMS API để hiển thị biểu đồ và trạng thái đối soát.

---

## 3. Sơ đồ tương tác tổng thể

| Thành phần | Vai trò | Giao tiếp (Input) | Giao tiếp (Output) |
| :--- | :--- | :--- | :--- |
| **CDC Worker (Go)** | Xử lý & Đối soát | NATS (Events), Redis (Config) | Postgres (Data), NATS (Drift Alerts) |
| **CMS API (Go)** | Điều phối & Quản lý | REST (từ FE), Postgres (Registry) | NATS (Reload signal), Airbyte API |
| **CMS FE (React)** | Giao diện người dùng | REST (từ API) | REST (Gửi lệnh Approve/Config) |

---

## 4. Tại sao Stack này tối ưu cho AI Auto-Reconcile?

Sử dụng **Fiber/Echo** và **NATS** cho phép bạn tích hợp AI cực kỳ linh hoạt:
1.  **AI Worker (Python/Go):** Có thể là một Service riêng biệt lắng nghe một NATS subject đặc thù cho các ca "Mismatch".
2.  **Async Processing:** Khi Recon Service phát hiện lệch tiền, nó đẩy vào NATS. AI Agent lấy về xử lý, sau khi có kết quả (Gợi ý/Approve) thì cập nhật lại DB và CMS FE sẽ tự động hiển thị qua cơ chế Long Polling hoặc WebSocket.

---



Việc bổ sung một **Auth Service** chuyên biệt để quản lý JWT và phân quyền (RBAC) là mảnh ghép cuối cùng để hoàn thiện hệ thống CDC chuẩn Enterprise của bạn. Với Stack **Fiber/Echo** và **Wire**, bạn có thể triển khai middleware này một cách rất mượt mà.

Dưới đây là cách kiến trúc Auth Service phối hợp với hệ thống hiện tại:

## 1. Kiến trúc Auth Service & Middleware

Thay vì để mỗi Service tự quản lý User, bạn xây dựng một **Auth Service** tập trung để cấp phát và xác thực Token.

* **Auth Service:** Chịu trách nhiệm Login, Register, và Issue JWT (chứa `user_id`, `role`, `permissions`).
* **JWT Middleware:** Nằm tại **CMS API** và các service nhạy cảm khác. Khi FE call API, Middleware này sẽ chặn lại để verify Token.
* **Giao tiếp:** CMS API sẽ dùng một `AuthClient` (được inject qua Wire) để giao tiếp với Auth Service (qua gRPC hoặc internal HTTP) để check tính hợp lệ của Token hoặc lấy thêm thông tin User.



---

## 2. Luồng giao tiếp Event-Driven & REST

Sự bổ sung này thay đổi luồng giao tiếp như sau:

### CMS Frontend (FE) -> CMS API (REST)
* **Login:** FE gửi User/Pass tới `POST /api/auth/login`.
* **Token:** Nhận về JWT và lưu vào `localStorage` hoặc `HttpOnly Cookie`.
* **Authenticated Requests:** Mọi request sau đó (như `Approve Schema`, `Update Registry`) đều phải đính kèm Header `Authorization: Bearer <token>`.

### CMS API -> Core Worker (NATS)
* **Context Propagation:** Khi CMS API nhận được lệnh Approve từ một User đã log in, nó sẽ đính kèm `user_id` vào payload gửi qua NATS subject `schema.config.reload`.
* **Audit Trail:** **Core Worker** khi nhận event sẽ ghi lại: "User X đã thực hiện thay đổi Schema này" vào bảng `schema_changes_log` để phục vụ đối soát và hậu kiểm.

---

## 3. Cấu trúc Code với Wire & Fiber/Echo

Trong bộ khung của bạn, việc tích hợp sẽ trông như thế này:

```go
// internal/api/middleware/auth.go
func JWTMiddleware(jwtSecret string) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // 1. Extract token từ Header
            // 2. Verify bằng jwtSecret (lấy từ Viper config)
            // 3. Bind thông tin user vào Context
            return next(c)
        }
    }
}
```

**Wire Provider** sẽ lo việc inject Secret Key từ **Viper** vào Middleware này:

* **Viper:** Đọc `JWT_SECRET` từ config.
* **Wire:** Inject `Config` vào `Middleware`, sau đó inject `Middleware` vào `Echo/Fiber Server`.

---

## 4. Bảo mật cho AI Auto-Reconcile

Với Auth Service, bạn có thể phân quyền cực kỳ chi tiết cho các nghiệp vụ "nhạy cảm":

* **Role Admin:** Được phép sửa Workflow đối soát và Approve các ca lệch tiền lớn.
* **Role Operator:** Chỉ được xem báo cáo và Approve các ca mà AI gợi ý với độ tin cậy cao (>95%).
* **Audit Log:** Mọi hành động của AI Agent cũng được gán cho một "System User" để bạn theo dõi AI đang làm gì trong hệ thống.

---

## Tóm tắt mô hình giao tiếp 4 thành phần

| Giao tiếp | Phương thức | Nội dung |
| :--- | :--- | :--- |
| **FE -> Auth Service** | REST | Login / Refresh Token |
| **FE -> CMS API** | REST + JWT | Quản lý Registry, Approve Schema |
| **CMS API -> Worker** | NATS (Event) | Reload Config + Metadata người thực hiện |
| **Worker -> Postgres** | SQL | Lưu dữ liệu + Audit Log (ai làm, lúc nào) |



Việc **CDC Worker** không có lớp Auth trực tiếp (như JWT hay Login API) là một lựa chọn **ổn và phổ biến** trong kiến trúc Microservices, miễn là bạn tuân thủ các nguyên tắc bảo mật ở tầng hạ tầng (Infrastructure Security).

Dưới đây là phân tích chi tiết tại sao nó ổn và những điểm bạn cần "rào" lại để đảm bảo an toàn cho 200 tables:

### 1. Tại sao CDC Worker không cần Auth (Internal Service)
* **Vị trí trong mạng nội bộ:** CDC Worker thường chạy trong mạng nội bộ (VPC/Subnet riêng) hoặc trong cụm K8s. Nó không "lộ diện" ra ngoài Internet nên không đối mặt với các cuộc tấn công Brute-force hay xâm nhập trực tiếp từ User.
* **Cơ chế Event-Driven:** Worker hoạt động dựa trên việc "Subscribe" (đăng ký nhận tin) từ NATS JetStream. Kẻ tấn công không thể gửi dữ liệu giả mạo vào Worker trừ khi chúng chiếm quyền điều khiển được cụm NATS hoặc Source DB.
* **Hiệu suất (Performance):** Việc xác thực JWT cho hàng ngàn event mỗi giây sẽ gây ra độ trễ (latency) không đáng có cho một dịch vụ xử lý nền (background worker).



### 2. Các lớp bảo mật thay thế (Bắt buộc phải có)
Dù không có Auth API, bạn vẫn phải bảo mật Worker bằng các cách sau:

* **Bảo mật kết nối NATS/Redis:** Worker phải dùng thông tin xác thực (Username/Password hoặc Token) để kết nối với NATS và Redis. Những thông tin này được quản lý qua **Viper Config** và truyền vào dưới dạng **K8s Secrets**.
* **Bảo mật Database:** User `cdc_worker` trong Postgres chỉ có quyền `INSERT/UPDATE` trên các bảng dữ liệu, không có quyền xóa bảng hay can thiệp vào bảng hệ thống của Auth Service.
* **Trust-but-Verify (Xác thực qua CMS):** Worker chỉ thực thi các lệnh nhạy cảm (như Reload Config) khi nhận được event từ subject đặc thù của CMS. Bạn có thể ký số (Digital Signature) vào payload event này để Worker xác nhận đúng là do CMS API gửi đi.

### 3. Rủi ro và Cách khắc phục
* **Rủi ro:** Một service nội bộ khác bị chiếm quyền và bắn tin nhắn rác vào NATS làm Worker xử lý sai dữ liệu.
* **Khắc phục:** Sử dụng **NATS Permissions** để giới hạn: chỉ có User của Airbyte/Debezium mới được phép `Publish` vào các subject `cdc.goopay.>`. Worker chỉ có quyền `Subscribe`.

### 4. So sánh với CMS API (Có Auth)

| Tiêu chí | CMS API (Cần Auth) | CDC Worker (Không cần Auth) |
| :--- | :--- | :--- |
| **Đối tượng tiếp cận** | User (BA, Ops, Admin) | Hệ thống (NATS, Redis, DB) |
| **Giao tiếp** | HTTP Public/VPN | Nội bộ (Internal Network) |
| **Bảo mật chính** | JWT, RBAC, Middleware | Network Policy, Secrets, NATS ACL |

**Kết luận:** Với một hệ thống CDC, việc tập trung Auth vào **CMS API** để kiểm soát con người là hoàn toàn hợp lý. Còn **Core Worker** nên tập trung vào hiệu suất và bảo mật bằng hạ tầng.
