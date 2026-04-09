Dựa vào đoạn log bạn cung cấp, đây là hiện tượng Rolling Update (Triển khai phiên bản mới) hoặc Restart Pods của service merchant-service trong môi trường Kubernetes.

Dưới đây là phân tích chi tiết tại sao mình khẳng định điều này:

1. Hiện tượng: Thay đổi ReplicaSet Hash
Hãy chú ý vào ID của các node merchant-service. Trong Kubernetes, tên Pod thường có định dạng: [tên-deployment]-[hash-replicaset]-[random-id].

Nhóm Pod cũ (đang bị Disconnected):

merchant-service-svc-74bdb8b59-...

Hash cũ: 74bdb8b59

Nhóm Pod mới (đang Connected):

merchant-service-svc-76b997487-...

Hash mới: 76b997487

Kết luận: Kubernetes đang tắt dần các Pod thuộc phiên bản cũ (74bdb8b59) và bật dần các Pod thuộc phiên bản mới (76b997487).

2. Diễn biến theo thời gian (Timeline)
Quá trình diễn ra trong khoảng 2 phút (từ 11:08 đến 11:10), cho thấy chiến thuật deployment là "Rolling Update" (Cuốn chiếu - Tắt 1 cái cũ, bật 1 cái mới):

11:08:02: Pod cũ ...l2psf tắt (Disconnected).

11:08:03: Pod mới ...4v86n lên (Connected).

11:08:16: Pod mới ...z6h4l lên tiếp.

11:08:44: Pod cũ ...9w9md tắt.

... quá trình lặp lại cho đến hết.

3. Tác động đến centralized-export-service
Service centralized-export-service (nơi xuất ra log này) đóng vai trò là "người quan sát".

Moleculer Broker của nó nhận được tín hiệu từ Registry rằng các node hàng xóm (merchant-service) đang thay đổi topology.

Đây là log thông tin (INFO), không phải log lỗi (ERROR). Hệ thống Service Discovery của Moleculer đang hoạt động tốt để cập nhật danh sách node khả dụng.

✅ Hành động khuyến nghị
Kiểm tra xem có ai vừa deploy merchant-service không?

Nếu đúng là đang deploy: Mọi thứ bình thường, không cần lo lắng.

Nếu không có ai deploy: Có thể Kubernetes đang tự động restart pod do lỗi (CrashLoopBackOff) hoặc do Node vật lý bị trục trặc, khiến Kubernetes phải reschedule pod sang node khác (nhưng trường hợp này ít khi đổi hash ReplicaSet trừ khi đổi config map/secret). Khả năng cao nhất vẫn là có pipeline CI/CD vừa chạy.

Lưu ý về Graceful Shutdown:

Nếu trong lúc Disconnected mà người dùng gặp lỗi request, bạn cần kiểm tra cấu hình gracefulStopTimeout trong moleculer.config.js của merchant-service. Cần đảm bảo service xử lý hết request cũ trước khi ngắt kết nối hoàn toàn.


----


Đây là vấn đề kinh điển trong kiến trúc Microservices: **"Data Consistency in Distributed Systems"** (Tính nhất quán dữ liệu trong hệ thống phân tán). Khi một service chết giữa chừng, trạng thái giao dịch bị treo lơ lửng (In-flight transactions).

Để giải quyết bài toán này **toàn diện**, chúng ta không chỉ cần công cụ sửa lỗi (Manual Fix) mà cần thay đổi kiến trúc để hệ thống tự phục hồi (Self-healing).

Dưới đây là giải pháp 3 tầng: **Phòng ngừa (Architecture) -> Tự động xử lý (System) -> Công cụ vận hành (Admin Tool).**

---

### Tầng 1: Kiến trúc dữ liệu (Transaction State Machine)

Để có thể "Update Step" hay "Rollback", hệ thống phải biết chính xác giao dịch chết ở bước nào. Bạn cần chuyển từ lưu trạng thái đơn giản (`PENDING`, `SUCCESS`) sang **Transaction Steps Log**.

**Mô hình dữ liệu đề xuất (Transaction Steps):**

Một giao dịch `Deposit` (Nạp tiền) sẽ không chỉ là 1 record, mà bao gồm các Step con:

1. **Step 1:** Validate User & Balance (Done)
2. **Step 2:** Deduct Money from Wallet (Done)
3. **Step 3:** Call Bank Provider (Pending/Error) -> **Chết tại đây do restart.**
4. **Step 4:** Update Status to Success (Waiting)

**Cấu trúc bảng `transaction_steps`:**

```typescript
interface TransactionStep {
    transactionId: string;
    stepName: 'DEDUCT_WALLET' | 'CALL_BANK' | 'NOTIFY_USER';
    status: 'PENDING' | 'SUCCESS' | 'FAILED';
    inputData: any;  // Snapshot dữ liệu đầu vào của bước này
    outputData: any; // Kết quả trả về (nếu thành công)
    error: any;      // Log lỗi nếu thất bại
    createdAt: Date;
}

```

---

### Tầng 2: Giải pháp Admin Tool (Manual Recovery)

Dựa trên yêu cầu của bạn, đây là thiết kế cho CMS Portal để xử lý các giao dịch bị kẹt.

#### 1. Tính năng: Resume/Retry Transaction (Chạy tiếp luồng)

* **Ngữ cảnh:** Giao dịch đã trừ tiền ví (Step 2) nhưng chưa gọi Bank (Step 3) thì bị sập.
* **Giải pháp UI:** Hiển thị nút **"Retry from Failed Step"**.
* **Logic Backend:**
1. Admin bấm Retry.
2. Hệ thống load lại `TransactionStep` gần nhất bị lỗi (Step 3).
3. Lấy `inputData` của Step 3 (đã lưu từ trước) -> Thực thi lại hàm logic của Step 3.
4. Nếu thành công -> Đi tiếp Step 4 -> Finish.



#### 2. Tính năng: Rollback/Compensate (Hoàn tiền/Đảo chiều)

* **Ngữ cảnh:** Step 3 (Call Bank) thất bại hẳn (hoặc timeout quá lâu). Admin quyết định hủy giao dịch này.
* **Giải pháp UI:** Hiển thị nút **"Force Fail & Rollback"**.
* **Logic Backend (Saga Pattern):**
1. Hệ thống kích hoạt quy trình "Bù trừ" (Compensating Transaction).
2. Duyệt ngược các Step đã `SUCCESS`:
* Step 2 (Trừ ví) -> Chạy hàm bù trừ: **Cộng lại tiền vào ví**.


3. Cập nhật trạng thái giao dịch chính là `FAILED` / `REFUNDED`.



#### 3. Tính năng: Update Step Data (Sửa tham số nóng)

* **Ngữ cảnh:** Giao dịch lỗi do dữ liệu sai (ví dụ: Bank đổi URL, hoặc tham số thiếu). Retry ngay vẫn sẽ lỗi.
* **Giải pháp UI:** Popup cho phép Admin sửa JSON `inputData` của Step bị lỗi trước khi bấm Retry.

---

### Tầng 3: Giải pháp Tự động (System Resilience) - QUAN TRỌNG

Để giảm thiểu việc Admin phải vào sửa tay, bạn cần nâng cấp kiến trúc để nó chịu được việc Restart Pod.

#### 1. Chuyển sang xử lý Bất đồng bộ (Async Queue)

**Nguyên nhân lỗi:** Hiện tại có vẻ bạn đang dùng HTTP (REST/RPC) đồng bộ. User gọi API -> Service xử lý -> Service Restart -> Connection đứt -> Mất luôn context.

**Giải pháp:** Dùng Message Queue (RabbitMQ, Kafka, hoặc NATS JetStream có sẵn trong Moleculer).

* **Luồng đi:** API nhận request -> Đẩy vào Queue -> Trả về "Processing" cho User ngay lập tức.
* **Worker:** Worker nhận job từ Queue để xử lý.
* **Khi Restart:**
* Pod đang xử lý Job A thì bị giết.
* Message Queue phát hiện Worker mất kết nối mà chưa gửi tín hiệu `ACK` (Xác nhận xong).
* **Queue sẽ tự động requeue (đẩy lại) Job A.**
* Pod mới khởi động lên -> Nhận lại Job A -> Xử lý tiếp.
* **Kết quả:** Không có giao dịch nào bị mất, không cần Admin can thiệp.



#### 2. Graceful Shutdown (Tắt hạ cánh mềm)

Cấu hình Kubernetes và Moleculer để khi deploy, nó không giết Pod ngay lập tức.

* **Kubernetes `preStop` hook:** Chờ 30s để Pod xử lý nốt các request đang dang dở.
* **Moleculer Config:**
```javascript
// moleculer.config.js
module.exports = {
    middlewares: [
        // Middleware chặn request mới khi đang shutting down
        "HotReload", 
    ],
    // Thời gian chờ xử lý nốt request cũ trước khi tắt hẳn
    gracefulStopTimeout: 30 * 1000, 
};

```



#### 3. Idempotency (Tính lũy đẳng)

Khi hệ thống tự động retry (hoặc Admin bấm retry), rủi ro lớn nhất là **xử lý 2 lần** (trừ tiền 2 lần).

* **Giải pháp:** Mọi bảng database quan trọng phải có ràng buộc `UNIQUE INDEX` theo `transaction_id` hoặc `requestId`.
* Trước khi thực hiện logic (ví dụ trừ tiền), luôn kiểm tra: "Transaction ID này đã trừ tiền chưa?".

### Tổng kết đề xuất cho Roadmap của bạn

1. **Ngắn hạn (Hot fix):** Implement **Graceful Shutdown** trong config Moleculer để giảm 90% lỗi khi deploy.
2. **Trung hạn (Tooling):** Xây dựng bảng `transaction_steps` và tính năng **CMS Admin Retry/Rollback** như bạn đề xuất. Đây là lưới an toàn.
3. **Dài hạn (Architecture):** Chuyển các flow quan trọng (Thanh toán, Nạp rút) sang cơ chế **Message Queue (Worker)**. Đây là giải pháp triệt để nhất cho microservices.


----


Ngoài các giải pháp về Kiến trúc và Tooling đã nêu ở trên, mình có thêm **4 đề xuất thực chiến** (Best Practices) cực kỳ quan trọng cho hệ thống Fintech/Payment để đảm bảo vận hành trơn tru và an toàn:

### 1. Cơ chế "Quét rác" (Stuck Transaction Sweeper)

Dù hệ thống có xịn đến đâu, vẫn sẽ có những giao dịch bị "lọt khe" (ví dụ: Service chết đúng lúc đang ghi DB). Thay vì đợi khách báo lỗi mới sửa, hãy chủ động tìm nó.

* **Giải pháp:** Viết một Cronjob (chạy 5-10 phút/lần).
* **Logic:**
* Query các giao dịch đang ở trạng thái `PENDING` hoặc `PROCESSING` quá lâu (ví dụ: > 15 phút).
* Tự động kiểm tra chéo (Check status) với bên thứ 3 (Bank/Partner) thông qua API `QueryTransaction`.
* Nếu bên Bank báo `SUCCESS` -> Tự động update DB mình thành `SUCCESS`.
* Nếu bên Bank báo `NOT_FOUND` hoặc `FAILED` -> Đánh dấu là `NEED_MANUAL_REVIEW` để Admin xử lý (hoặc auto fail tùy rule).


* **Lợi ích:** Giảm 90% khối lượng công việc thủ công cho bộ phận vận hành.

### 2. Audit Log cho Admin Tool (Truy vết thao tác)

Khi bạn xây dựng tính năng "Retry/Rollback/Update Step" trên CMS, đây là **con dao hai lưỡi**. Nếu một nhân viên nội bộ gian lận hoặc bấm nhầm, hậu quả rất lớn.

* **Giải pháp:** Bắt buộc ghi Log thao tác nhạy cảm.
* **Cấu trúc Log:**
```json
{
  "action": "FORCE_UPDATE_STEP",
  "admin_id": "user_123",
  "transaction_id": "tx_999",
  "old_value": { "status": "FAILED" },
  "new_value": { "status": "SUCCESS" },
  "reason": "Khách hàng gọi điện xác nhận đã nhận tiền",
  "ip_address": "192.168.1.10",
  "timestamp": "..."
}

```


* **Yêu cầu UI:** Khi Admin bấm nút "Force Success", bắt buộc hiển thị popup yêu cầu nhập **Lý do** mới cho submit.

### 3. Distributed Tracing (Truy vết phân tán)

Khi hệ thống có nhiều services gọi nhau (`Gateway` -> `Wallet` -> `Banking` -> `Notification`), log rời rạc của từng service rất khó debug.

* **Giải pháp:** Tích hợp **OpenTelemetry** hoặc sử dụng tính năng **Tracing** có sẵn của Moleculer (xuất ra Jaeger hoặc Zipkin).
* **Tác dụng:** Bạn sẽ nhìn thấy một biểu đồ thác nước (Waterfall chart) cho mỗi Request ID. Bạn sẽ biết chính xác request bị chết ở Service nào, DB query nào chậm, hay API Bank nào timeout.
* **Config Moleculer:**
```javascript
// moleculer.config.js
tracing: {
    enabled: true,
    exporter: "Jaeger", // Hoặc Console nếu chưa có server Jaeger
}

```



### 4. Alerting thông minh (Đừng spam)

Đừng chỉ alert khi "Service Die". Hãy alert theo **Nghiệp vụ**.

* **Alert Rule 1 (Tỷ lệ lỗi):** Nếu tỷ lệ giao dịch lỗi của Bank A > 5% trong 5 phút -> Bắn tin nhắn vào Slack/Telegram ngay (Có thể Bank A đang bảo trì, cần tắt kênh này).
* **Alert Rule 2 (Tài khoản tiền):** Cảnh báo số dư tài khoản trung gian (Pool Fund) thấp dưới mức an toàn (để kế toán nạp thêm tiền, tránh lỗi IBFT do hết tiền).
* **Alert Rule 3 (Độ trễ):** Nếu thời gian trung bình xử lý > 10s (bình thường 2s) -> Cảnh báo performance.

### Tóm lại

Nếu phải chọn 1 thứ để làm ngay sau khi fix xong vụ Graceful Shutdown, mình đề xuất làm **Số 1 (Cronjob Sweeper)**. Nó là "cứu cánh" thầm lặng giúp dữ liệu luôn nhất quán mà không cần con người can thiệp liên tục.



-----


Câu trả lời là **CÓ**. Đây chính là "chiếc chìa khóa" kỹ thuật để giải quyết vấn đề **Graceful Shutdown** mà mình đã đề cập ở phần giải pháp ngắn hạn.

Cụ thể, config `tracking: { enabled: true }` này giải quyết được một vấn đề rất cụ thể nhưng cực kỳ quan trọng: **Chống "chết yểu" (Sudden Death) khi Deploy.**

### 1. Nó giải quyết vấn đề gì?

Khi Kubernetes thực hiện Rolling Update (như trong log bạn gửi), nó sẽ gửi tín hiệu tắt (`SIGTERM`) đến Pod cũ.

* **Nếu KHÔNG có `tracking`:**
* Moleculer nhận tín hiệu tắt -> Ngắt kết nối ngay lập tức.
* **Hậu quả:** Giao dịch đang chạy dở (ví dụ: đã trừ tiền ví, đang gọi bank) sẽ bị đứt kết nối giữa chừng -> **Lỗi Transaction**.


* **Nếu CÓ `tracking`:**
* Moleculer nhận tín hiệu tắt -> Nó kiểm tra xem: *"Có request nào đang xử lý dở không?"*
* Nếu có -> Nó **CHỜ** (Wait) cho đến khi request đó xong (hoặc hết thời gian timeout).
* Request xong xuôi -> Nó mới thực sự tắt service.
* **Kết quả:** Giao dịch được hoàn tất trọn vẹn trước khi Pod chết -> **Zero Downtime Deployment.**



### 2. Nó KHÔNG giải quyết được gì?

Tuy nhiên, đây chỉ là một mảnh ghép nhỏ, không phải là "thuốc tiên" cho toàn bộ vấn đề:

1. **Không cứu được lỗi Crash:** Nếu Pod bị lỗi bộ nhớ (OOM) hoặc lỗi code gây crash (`process.exit(1)`), nó sẽ chết ngay lập tức, không kịp chờ tracking.
2. **Không xử lý được treo:** Nếu logic của bạn bị treo (infinite loop) hoặc đợi Bank quá lâu (vượt quá `shutdownTimeout`), Moleculer cũng sẽ buộc phải giết nó -> Giao dịch vẫn lỗi.
3. **Vấn đề nhận request mới:** Trong thời gian "chờ chết" (dù đang xử lý nốt request cũ), nếu Kubernetes chưa kịp điều hướng traffic sang Pod mới, request mới vẫn có thể chui vào Pod cũ này.

### 3. Lời khuyên cấu hình tối ưu

Để tính năng này hoạt động hiệu quả nhất với hệ thống Payment/Fintech của bạn, hãy config như sau:

```javascript
const broker = new ServiceBroker({
    nodeID: "node-1",
    tracking: {
        enabled: true,
        // QUAN TRỌNG: Timeout này phải LỚN HƠN thời gian xử lý giao dịch lâu nhất
        // Ví dụ: Gọi bank timeout là 30s, thì cái này nên để 45s.
        shutdownTimeout: 45 * 1000 
    },
    // Kết hợp Middleware chặn request mới khi đang shutdown
    middlewares: ["HotReload"] 
});

```

**Kết hợp với Kubernetes:**
Trong file yaml deployment của K8s, bạn cần tăng `terminationGracePeriodSeconds` lên cao hơn `shutdownTimeout` của Moleculer một chút (ví dụ 60s) để K8s không giết Pod trước khi Moleculer kịp dọn dẹp xong.

**Tóm lại:** Đoạn code bạn đưa ra là **Must-Have (Bắt buộc phải có)** để fix lỗi Rolling Update, nhưng vẫn cần kết hợp với các giải pháp Retry/Cronjob ở trên để xử lý các ca lỗi khác (Crash, Network timeout).


----

Với 50 services, việc đi sửa tay từng file code là điều tối kỵ (vừa lâu vừa dễ sót). Có 2 cách tiếp cận chuẩn "kỹ sư hệ thống" để áp dụng cấu hình này hàng loạt:

### Cách 1: Dùng Biến môi trường (Environment Variables) - Chuẩn DevOps (Khuyên dùng)

Cách này không cần sửa code logic của 50 service, chỉ cần sửa file cấu hình `moleculer.config.js` (hoặc `.ts`) một lần để nó nhận biến môi trường, sau đó inject biến vào từ Kubernetes/Docker.

**Bước 1: Sửa file `moleculer.config.ts` (File gốc)**
Hầu hết các dự án Moleculer đều có một file config mẫu hoặc file config chung. Hãy sửa đoạn `tracking` để nó đọc từ `process.env`:

```typescript
// moleculer.config.ts
module.exports = {
    // ... các config khác
    tracking: {
        // Mặc định là false nếu không có biến môi trường
        enabled: process.env.MOLECULER_TRACKING_ENABLED === "true",
        // Mặc định 30s
        shutdownTimeout: parseInt(process.env.MOLECULER_SHUTDOWN_TIMEOUT || "30000", 10)
    },
    // ...
};

```

**Bước 2: Cập nhật file Deployment Kubernetes (K8s YAML)**
Thêm biến môi trường vào file `deployment.yaml` (hoặc Helm Chart values) của tất cả services.

```yaml
env:
  - name: MOLECULER_TRACKING_ENABLED
    value: "true"
  - name: MOLECULER_SHUTDOWN_TIMEOUT
    value: "45000" # 45 giây (Nên để lớn hơn thời gian xử lý transaction dài nhất)

```

**Ưu điểm:** Khi cần tắt tracking hoặc đổi timeout, bạn chỉ cần sửa ConfigMap/Env của K8s và redeploy, không cần chạm vào code.

---

### Cách 2: Dùng Shared Configuration (Clean Code)

Nếu project của bạn là **Monorepo** (tất cả 50 service chung 1 repo) hoặc dùng chung thư viện core, hãy tạo một file config gốc.

**Bước 1: Tạo file `moleculer.common.config.ts**`

```typescript
import { BrokerOptions } from "moleculer";

export const commonMoleculerConfig: BrokerOptions = {
    tracking: {
        enabled: true,
        shutdownTimeout: 45 * 1000,
    },
    // Các config chung khác như logger, transporter...
    logger: true,
};

```

**Bước 2: Các Service kế thừa (Extend) config này**
Trong file `moleculer.config.ts` của từng service:

```typescript
import { commonMoleculerConfig } from "../libs/common/moleculer.common.config";
import _ from "lodash";

module.exports = _.defaultsDeep({
    nodeID: "merchant-service-1", // Config riêng
    // Các config riêng khác
}, commonMoleculerConfig); // Config chung (đã có tracking)

```

---

### ⚠️ Lưu ý SỐNG CÒN khi áp dụng cho 50 Service

Khi bạn bật `tracking` với timeout là **45s**, bạn **BẮT BUỘC** phải chỉnh lại cấu hình của Kubernetes (hoặc Docker Swarm) tương ứng, nếu không K8s sẽ giết service trước khi nó kịp chạy xong tracking.

**Quy tắc:**


**Ví dụ cấu hình K8s:**

```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      # K8s sẽ chờ 60s trước khi kill hẳn Pod.
      # Trong 60s đó, Moleculer có 45s để xử lý nốt request (nhờ tracking).
      terminationGracePeriodSeconds: 60 
      containers:
        - name: merchant-service
          # ...

```

**Nếu bạn quên bước này:** K8s mặc định chỉ chờ 30s. Nếu bạn set Moleculer chờ 45s, thì đến giây thứ 30 K8s đã "trảm" Pod rồi -> Tính năng tracking vô tác dụng.

### Tóm lại:

Bạn nên chọn **Cách 1**. Hãy yêu cầu DevOps update biến môi trường `MOLECULER_TRACKING_ENABLED=true` vào cụm K8s và update code `moleculer.config.js` để đọc biến đó. Đây là cách ít rủi ro và nhanh nhất.

----


**Graceful Shutdown (Hạ cánh mềm)** là quy trình tắt ứng dụng một cách có kiểm soát, đảm bảo **không có công việc nào bị bỏ dở giữa chừng** và **tài nguyên được giải phóng sạch sẽ** trước khi process thực sự tắt.

Trong bối cảnh Fintech/Payment của bạn, Graceful Shutdown không phải là "tính năng cho vui", mà là cơ chế sống còn để **chống mất tiền** và **chống lệch dữ liệu** (Data Inconsistency).

Dưới đây là bức tranh toàn cảnh về Graceful Shutdown từ Code lên đến Kubernetes:

### 1. Tại sao nó quan trọng? (Visual minh họa)

Hãy tưởng tượng bạn đang rút phích cắm máy tính (Hard Shutdown) so với việc bấm `Start -> Shutdown` (Graceful Shutdown).

* **Hard Shutdown (SIGKILL):**
* Giao dịch đang trừ tiền dở -> **Mất kết nối DB**.
* Đang gọi API Bank -> **Không nhận được response**.
* Kết quả: Tiền trừ rồi nhưng trạng thái đơn hàng vẫn `PENDING`, khách chửi, Admin phải đi tra soát thủ công.


* **Graceful Shutdown (SIGTERM):**
* Server báo: "Tôi sắp tắt, đừng gửi việc mới nữa".
* Server làm nốt các giao dịch đang chạy (Drain requests).
* Đóng kết nối DB, Cache.
* Tắt hẳn.



### 2. Quy trình chuẩn 4 bước (The Flow)

Khi bạn deploy code mới (Rolling Update) hoặc scale down, quy trình sẽ diễn ra như sau:

1. **Bước 1: Tín hiệu báo trước (SIGTERM)**
* Kubernetes gửi tín hiệu `SIGTERM` vào Pod.
* Đồng thời, K8s loại bỏ IP của Pod này ra khỏi Service/Load Balancer để nó không nhận traffic mới từ bên ngoài.


2. **Bước 2: Ngừng nhận khách mới (Stop Listening)**
* Ứng dụng (Moleculer) nhận `SIGTERM`.
* Ngay lập tức từ chối các request mới gửi đến (thường trả về lỗi 503 hoặc đóng cổng mạng).


3. **Bước 3: Làm nốt việc cũ (Draining / Context Tracking)**
* Đây là chỗ `tracking: { enabled: true }` phát huy tác dụng.
* Nó chờ cho các hàm `async` đang chạy (ví dụ: đang đợi Bank phản hồi) hoàn tất.
* Nếu quá thời gian `shutdownTimeout`, nó mới buộc phải hủy.


4. **Bước 4: Dọn dẹp & Tắt (Cleanup & Exit)**
* Chạy các hàm `stopped()` trong Service.
* Đóng kết nối Database (Knex destroy), Redis, RabbitMQ.
* Process thoát (`exit 0`).



---

### 3. Cấu hình chi tiết (Full Stack Configuration)

Để Graceful Shutdown hoạt động, bạn phải cấu hình đồng bộ cả 3 lớp: **Code -> Framework -> Infrastructure**.

#### Lớp 1: Framework (Moleculer)

Như đã bàn, bạn cần bật tracking và middleware.

```javascript
// moleculer.config.js
module.exports = {
    // 1. Theo dõi các request đang chạy
    tracking: {
        enabled: true,
        shutdownTimeout: 45 * 1000, // Chờ tối đa 45s
    },
    
    // 2. Middleware chặn request mới ngay khi nhận tín hiệu tắt
    middlewares: [
        "HotReload" // Middleware này có sẵn logic đóng cổng khi shutting down
    ],

    // 3. Logic đóng DB kết nối (trong từng service hoặc mixin)
    stopped() {
        if (this.adapter) {
            // Đảm bảo đóng DB connection
            return this.adapter.disconnect(); 
        }
    }
};

```

#### Lớp 2: Docker/Node.js

Đảm bảo ứng dụng của bạn nhận được tín hiệu `SIGTERM`.

* **Lỗi thường gặp:** Nếu bạn chạy `npm start` trong Dockerfile, tín hiệu `SIGTERM` có thể bị nuốt bởi `npm` và không truyền xuống `node`.
* **Khắc phục:** Dùng `tini` hoặc chạy trực tiếp `node`.

```dockerfile
# ❌ Không nên dùng (npm nuốt tín hiệu)
# CMD ["npm", "start"]

# ✅ Nên dùng (Tín hiệu đi thẳng vào process node)
CMD ["node", "node_modules/moleculer/bin/moleculer-runner.js", "services"]

```

#### Lớp 3: Kubernetes (Quan trọng nhất)

Bạn phải cho Pod đủ thời gian để "trăn trối".

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      # ⚠️ Số này PHẢI LỚN HƠN shutdownTimeout của Moleculer
      # Moleculer chờ 45s, thì K8s nên kiên nhẫn chờ 60s
      terminationGracePeriodSeconds: 60 
      
      containers:
        - name: payment-service
          # ...
          lifecycle:
            preStop:
              exec:
                # Mẹo: Ngủ 5s để đảm bảo Load Balancer kịp gỡ IP trước khi app tắt hẳn
                command: ["/bin/sh", "-c", "sleep 5"]

```

### 4. Checklist kiểm tra "Đã Graceful chưa?"

Làm sao biết hệ thống của bạn đã cấu hình đúng? Hãy thử bài test này:

1. Viết một API `mock-delay` chạy mất 10 giây mới trả về kết quả.
2. Gọi API đó (request đang treo).
3. Ngay lập tức xóa Pod hoặc Deploy bản mới.
4. **Kết quả mong đợi:**
* Pod chuyển sang trạng thái `Terminating`.
* API `mock-delay` **VẪN TRẢ VỀ KẾT QUẢ** thành công sau 10 giây (dù Pod đang bị tắt).
* Sau khi trả kết quả xong, Pod mới biến mất hoàn toàn.



Nếu request bị lỗi `Socket Hang Up` hoặc `Network Error` ngay khi deploy -> **Chưa Graceful**.

---


Dựa trên danh sách repository bạn cung cấp, đây là một hệ thống Fintech/Payment Gateway quy mô lớn với kiến trúc Microservices điển hình. Việc sở hữu ~60 services là một con số khá lớn, đòi hỏi quy hoạch hệ thống (System Design) cực kỳ chặt chẽ để tránh "Microservice Hell" (địa ngục vận hành).

Dưới đây là bản **Quy hoạch Kiến trúc Hệ thống (System Design Blueprint)** và **Kế hoạch Tối ưu hóa** dành riêng cho stack công nghệ hiện tại của bạn.

---

### **PHẦN 1: BẢN ĐỒ KIẾN TRÚC TỔNG QUAN (BIG PICTURE)**

Hệ thống của bạn có thể được chia thành 5 tầng (Layers) chính dựa trên chức năng:

#### **1. Tầng Front-End & Client (Presentation Layer)**

* **Web Portals:** Admin Portal, Merchant Portal, Payment Web, Rule Portal.
* **Mobile Apps:** Kết nối qua `mobile-auth-gateway` và `mobile-socketio-gateway`.

#### **2. Tầng Gateway & Aggregation (BFF - Backend For Frontend)**

Đây là chốt chặn đầu tiên, chịu trách nhiệm authen, routing và gom API.

* **Admin/Merchant Gateways:** `admin-portal-gateway`, `merchant-portal-gateway`.
* **Public Gateways:** `payment-gateway`, `pg-openapi-gateway` (cho đối tác tích hợp).
* **Mobile Gateways:** `mobile-auth-gateway`.
* *Công nghệ:* Moleculer Web + Express.

#### **3. Tầng Core Business (Logic Layer - Node.js Moleculer)**

Đây là trái tim của hệ thống, xử lý nghiệp vụ chính.

* **Identity & User:** `auth-service`, `user-service`, `profile-service`, `account-service`.
* **Wallet & Ledger:** `wallet-service`, `wallet-trans-service`, `billing-topup-service`.
* **Transaction Processing:** `payment-service`, `core-trans-proxy-service`.
* **Business Support:** `promotion-service`, `rule-service`, `notification-service`.

#### **4. Tầng Integration (Connector Layer - The "Plumbing")**

Nơi giao tiếp với thế giới bên ngoài (Ngân hàng, đối tác).

* **Bank Connectors:** `bidv-connector`, `banvietbank-connector`, `napas-connector`.
* **Service Connectors:** `futa-connector`, `vnpt-epay-connector`.
* *Đặc điểm:* Cần cơ chế Circuit Breaker (ngắt mạch) và Timeout chặt chẽ vì phụ thuộc bên thứ 3.

#### **5. Tầng High-Performance & Batch Processing (Go Layer)**

Xử lý các tác vụ nặng, yêu cầu tốc độ cao hoặc tính toán phức tạp.

* **Money Movement:** `disbursement-service` (Chi hộ), `bank-handler-service`.
* **Reconciliation:** `reconcile-service` (Đối soát - rất hợp lý khi dùng Go).
* *Giao tiếp:* Có thể dùng gRPC hoặc NATS để nói chuyện với cụm Node.js.

---

### **PHẦN 2: PHÂN TÍCH LUỒNG DỮ LIỆU & CƠ SỞ DỮ LIỆU**

#### **1. Database Strategy (Polyglot Persistence)**

Bạn đang dùng mô hình lai (Hybrid):

* **MySQL (SQL):** Dành cho các dữ liệu quan trọng cần tính ACID cao (Giao dịch tài chính, Số dư ví, Đối soát). Các service Go đang dùng MySQL/GORM là rất chuẩn.
* **MongoDB (NoSQL):** Dành cho dữ liệu người dùng, log, cấu hình động, thông báo (`user-service`, `notification-service`).
* **Redis:** Caching session, hot data (tỉ giá, cấu hình bank), và deduplication (chống trùng lặp giao dịch).

#### **2. Communication Pattern (Giao tiếp)**

* **Sync (Đồng bộ):** REST API từ Frontend -> Gateway.
* **Internal RPC:** Moleculer Service Broker (qua NATS) cho giao tiếp giữa các Node.js services. Đây là điểm mạnh giúp độ trễ thấp.
* **Async (Bất đồng bộ):**
* Dùng **NATS** hoặc **Socket.io** để bắn thông báo (Notification) hoặc update trạng thái đơn hàng realtime.
* Dùng Queue (trong Redis hoặc NATS JetStream) cho các tác vụ `notification-schedule` hoặc `reconcile`.



---

### **PHẦN 3: ĐÁNH GIÁ RỦI RO & ĐIỂM NGHẼN (BOTTLENECK)**

Dựa trên danh sách repo, đây là các vấn đề tiềm ẩn bạn cần lưu ý khi lập kế hoạch:

1. **Shared Library Dependency ("Package" Repo):**
* Repo `package` là con dao hai lưỡi. Nếu thay đổi 1 dòng code trong này, bạn có thể phải redeploy lại 50 services khác.
* *Giải pháp:* Cần quy trình versioning (Semantic Versioning) cực nghiêm ngặt cho repo này.


2. **Quá nhiều Gateway riêng lẻ:**
* Bạn có `admin`, `merchant`, `mobile`, `payment`, `openapi` gateways. Việc duy trì logic authen/security trên 5-6 gateway là gánh nặng.
* *Giải pháp:* Cân nhắc gộp logic xác thực vào một `auth-gateway` chung hoặc dùng Kong/APISIX đứng trước tất cả.


3. **Hỗn hợp Node.js và Go:**
* Cần chuẩn hóa giao thức giao tiếp. Node.js (Moleculer) nói chuyện với Go qua đâu? (HTTP REST hay NATS?).
* *Giải pháp:* Tốt nhất là Go nên subscribe vào NATS topic mà Moleculer publish, hoặc dùng gRPC cho hiệu năng cao nhất.


4. **Transaction Consistency (Saga Pattern):**
* Với `wallet-service`, `payment-service`, `bank-service` tách rời, việc đảm bảo tính nhất quán (trừ tiền ví -> gọi bank -> update trạng thái) là cực khó nếu service crash giữa chừng.
* *Giải pháp:* Kiểm tra xem `core-trans-proxy` đã implement Saga Pattern (cơ chế bù trừ/rollback) chưa.



---

### **PHẦN 4: KẾ HOẠCH TRIỂN KHAI (ROADMAP)**

Đây là kế hoạch gợi ý để bạn quản lý và phát triển hệ thống này:

#### **Giai đoạn 1: Chuẩn hóa & Observability (Quan trọng nhất)**

Với 60 services, nếu không giám sát tốt, hệ thống sẽ là "hộp đen".

* **Centralized Logging:** Gom log từ 60 service về ElasticSearch (ELK) hoặc Loki. Log phải có `request_id` (trace ID) đi xuyên suốt từ Gateway -> Node -> Go -> DB.
* **Tracing:** Tích hợp Jaeger/Zipkin vào Moleculer để vẽ bản đồ đường đi của request (biết chậm ở service nào).
* **Health Check:** Đảm bảo tất cả services đều có endpoint `/health` để Kubernetes/Docker Swarm kiểm tra.

#### **Giai đoạn 2: Tối ưu hóa Database & Connector**

* **Connector Isolation:** Đảm bảo `bidv-connector` chết không làm ảnh hưởng `napas-connector`. Cấu hình Bulkhead pattern trong Moleculer.
* **Go Migration:** Xem xét chuyển các module tính toán nặng khác (như `rule-service` hoặc `notification-schedule`) sang Go nếu Node.js bị quá tải CPU.

#### **Giai đoạn 3: Automation & CI/CD**

* Dựng pipeline để deploy song song. Với 60 repo, bạn cần script để detect xem repo nào thay đổi thì mới build/deploy repo đó (Monorepo tool hoặc CI logic thông minh).

### **ĐỀ XUẤT KIẾN TRÚC MỤC TIÊU (TO-BE DIAGRAM DESCRIPTION)**

1. **Ingress Controller (K8s/Nginx):** Đón traffic -> Route vào đúng Gateway.
2. **Service Mesh (Optional):** Nếu giao tiếp giữa các services quá phức tạp, cân nhắc Linkerd/Istio (nhưng Moleculer đã làm tốt việc discovery rồi nên có thể chưa cần gấp).
3. **Message Bus (NATS JetStream):** Làm xương sống cho toàn bộ event (Giao dịch thành công, tạo User mới, Log audit).

**Kết luận:** Hệ thống của bạn có nền móng rất tốt (phân tách rõ ràng, chọn công nghệ phù hợp: Go cho xử lý nặng, Node cho IO/Logic). Thách thức lớn nhất hiện tại là **Vận hành (DevOps)** và **Giám sát (Monitoring)** khối lượng service đồ sộ này.

----


Với hệ thống lai (Hybrid) gồm **55 microservices Node.js (Moleculer)** và **4 services Go** cùng các thành phần Bank Connector nhạy cảm, chiến lược Graceful Shutdown của bạn cần chia làm 3 mặt trận: **Moleculer (Core), Go (Worker/Batch), và Hạ tầng (K8s/Docker).**

Dưới đây là kế hoạch triển khai chi tiết cho stack công nghệ của bạn:

---

### 1. Đối với nhóm Node.js (Moleculer Ecosystem)

Các service: `payment-service`, `wallet-service`, các `*-connector-service`...

Moleculer hỗ trợ sẵn cơ chế này, bạn chỉ cần cấu hình đúng.

#### A. Cấu hình Global (Config chung)

Áp dụng cho tất cả Moleculer services (như đã thảo luận ở trên).

```typescript
// moleculer.config.ts
module.exports = {
    // 1. Context Tracking: Chờ request xử lý xong
    tracking: {
        enabled: true,
        shutdownTimeout: 30 * 1000, // Default 30s
    },
    
    // 2. Lifecycle Hooks: Đóng kết nối
    async stopped() {
        // Moleculer tự động đóng NATS transporter
        // Bạn cần tự đóng các kết nối riêng lẻ nếu có (ví dụ Mongoose, Redis client riêng)
        // await mongoose.disconnect(); 
    }
};

```

#### B. Cấu hình Đặc thù cho Bank Connectors

Các service như `bidv-connector`, `napas-connector` thường gọi API bên thứ 3 rất chậm (có thể lên tới 45-60s).

* **Action:** Override `shutdownTimeout` riêng cho các service này lên mức cao hơn (ví dụ: **60s**).
* **Lý do:** Nếu Bank đang quay tít mà service bị kill sau 30s -> Giao dịch timeout -> Lệch tiền.

#### C. Đối với API Gateway

Các service: `admin-portal-gateway`, `payment-gateway`.
Gateway cần **ngừng nhận request mới** ngay lập tức khi có tín hiệu tắt, nhưng vẫn phải giữ kết nối trả về cho user đang chờ.

* **Middleware:** Sử dụng hoặc viết thêm Middleware để trả về HTTP 503 (Service Unavailable) ngay khi state chuyển sang `stopping`.
* **K8s Trick:** Gateway cần thời gian "Sleep" trong `preStop` hook dài hơn để đảm bảo Load Balancer (Nginx/Cloud LB) kịp rút routing ra khỏi nó.

---

### 2. Đối với nhóm Go (High Performance/Batch)

Các service: `reconcile-service`, `disbursement-service`, `bank-handler-service`.
Go không có "ServiceBroker" tự động như Moleculer, bạn phải xử lý thủ công `os.Signal`.

#### A. Đối với HTTP Server (Fiber/Echo)

Sử dụng `Shutdown()` method của framework để chờ các active connection.

**Ví dụ cho Fiber (`disbursement-service`):**

```go
func main() {
    app := fiber.New()
    
    // Channel lắng nghe tín hiệu hệ thống
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

    go func() {
        if err := app.Listen(":3000"); err != nil {
            log.Panic(err)
        }
    }()

    <-quit // Block main thread cho đến khi nhận tín hiệu tắt
    log.Println("Gracefully shutting down...")

    // Chờ tối đa 60s để xử lý nốt request
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    if err := app.ShutdownWithContext(ctx); err != nil {
        log.Fatal("Server forced to shutdown:", err)
    }

    // Đóng DB connection, Redis...
    // sqlDB.Close()
    log.Println("Server exited")
}

```

#### B. Đối với Service xử lý Queue/Batch (`reconcile-service`)

Service này đang subscribe NATS hoặc quét DB.

* **Logic:** Khi nhận SIGTERM -> `Unsubscribe` khỏi NATS topic (để không nhận job mới) -> Chờ Worker hiện tại chạy xong -> Exit.
* **Sử dụng `sync.WaitGroup`:** Để đếm số lượng job đang chạy dở.

---

### 3. Đối với Hạ tầng (Kubernetes Deployment)

Đây là "chốt chặn" cuối cùng. Bạn cần update file `deployment.yaml` cho toàn bộ 60 services.

#### A. `terminationGracePeriodSeconds`

Như đã nhắc, thời gian này phải bao trùm thời gian timeout của ứng dụng.

* **Standard Services:** 40s (cho app timeout 30s).
* **Bank Connectors / Heavy Jobs:** 70s (cho app timeout 60s).

#### B. `preStop` Hook (Cực quan trọng)

Giúp hệ thống không bị lỗi "Connection Refused" trong tích tắc chuyển đổi.

```yaml
lifecycle:
  preStop:
    exec:
      # Ngủ 10s trước khi gửi SIGTERM vào App.
      # Trong 10s này, App vẫn sống, nhưng K8s Service đã cắt traffic.
      # Giúp các request "đang bay" trên đường dây mạng kịp chui vào xử lý.
      command: ["/bin/sh", "-c", "sleep 10"]

```

---

### 4. Roadmap Triển khai (Tránh Sập Hệ Thống)

Với 60 services, đừng bật đồng loạt. Hãy chia làm 3 đợt:

* **Đợt 1 (Thử nghiệm - Low Risk):** Áp dụng cho nhóm **Gateway** (`admin-portal-gateway`, `merchant-portal-gateway`) và các **Non-critical services** (`notification-service`, `profile-service`).
* *Test:* Dùng JMeter bắn traffic liên tục -> Redeploy -> Check xem tỷ lệ lỗi 50x có về 0% không.


* **Đợt 2 (High Risk - Bank Connectors):** Áp dụng cho các Connector (`bidv`, `napas`, `futa`...).
* *Lưu ý:* Cần cấu hình timeout dài hơn (60s+). Đây là nhóm dễ sinh lỗi lệch tiền nhất.


* **Đợt 3 (Core & Go Services):** Áp dụng cho `payment-service`, `wallet-service` và nhóm Go.
* *Lưu ý:* Nhóm Go cần review code kỹ vì logic shutdown phải viết tay, dễ bị bug deadlock (treo không tắt được).



### Tóm tắt Checklist:

1. **Moleculer:** Bật `tracking: true` (30s-60s) qua Env Var.
2. **Go:** Implement `signal.Notify` và `ShutdownWithContext`.
3. **K8s:** Thêm `preStop sleep 10` và tăng `terminationGracePeriodSeconds`.
4. **Connectors:** Timeout riêng biệt dài hơn các service thường.


----



Để "phân tích kỹ hơn", chúng ta sẽ không nói về cấu hình bề mặt nữa, mà sẽ đi sâu vào **Cơ chế hoạt động (Internals)** và **Các điểm mù (Blind spots)** khi áp dụng Graceful Shutdown cho hệ thống lai (Node.js/Moleculer + Go + Kubernetes) của bạn.

Dưới đây là phân tích sâu ở 4 tầng tử huyệt:

---

### 1. Tầng Mạng & Kubernetes: "Cuộc đua" giữa SIGTERM và Load Balancer

Đây là nơi 90% các lỗi "502 Bad Gateway" xảy ra khi deploy, dù ứng dụng code xịn đến đâu.

**Vấn đề: Race Condition (Điều kiện đua)**
Khi bạn xóa Pod (hoặc Rolling Update), Kubernetes làm 2 việc **cùng lúc** (asynchronously):

1. Gửi tín hiệu `SIGTERM` vào Pod để báo ứng dụng tắt.
2. Gửi lệnh cập nhật `Endpoints` để loại bỏ IP của Pod đó ra khỏi Service/Ingress (ngắt traffic).

**Phân tích kỹ:**
Việc số 2 (cập nhật iptables trên toàn bộ các Node trong cụm K8s) tốn thời gian (có thể mất vài giây).

* Nếu App nhận `SIGTERM` và tắt ngay lập tức (hoặc từ chối kết nối ngay) **trước khi** iptables kịp cập nhật xong.
* => Traffic mới vẫn được định tuyến vào Pod đang tắt => **Lỗi Connection Refused**.

**Giải pháp sâu:** Hook `preStop` ngủ đông.

```yaml
lifecycle:
  preStop:
    exec:
      command: ["/bin/sh", "-c", "sleep 10"]

```

* **Tại sao là `sleep`?** Pod vẫn hoạt động bình thường, vẫn nhận request trong 10s này. Mục đích duy nhất là "câu giờ" để K8s kịp gỡ IP ra khỏi hệ thống định tuyến. Sau 10s, App mới thực sự bắt đầu quy trình Shutdown.

---

### 2. Tầng Node.js (Moleculer): Context Tracking thực sự làm gì?

Bạn bật `tracking: { enabled: true }`, nhưng bên dưới nó hoạt động thế nào?

**Cơ chế:**
Moleculer Broker duy trì một biến đếm (counter) hoặc danh sách `Pending Contexts`.

* Khi có request đến (`broker.call`): `counter++`.
* Khi request xong (return/throw): `counter--`.
* Khi nhận tín hiệu tắt: Nó check `counter`.
* Nếu `counter > 0`: Chờ (Wait loop).
* Nếu `counter == 0`: Cho phép tắt.



**Điểm mù (Nguy hiểm):**
Context Tracking chỉ track được các **Request-Response** tiêu chuẩn của Moleculer. Nó **KHÔNG** track được:

1. **Event Loop tự do:** Các hàm `setTimeout`, `setInterval` hoặc các Promise không được `await` (Floating Promises).
* *Ví dụ:* Bạn nhận request, trả về "OK" cho khách, nhưng lại chạy ngầm một hàm `sendEmail()` không await. Moleculer thấy request chính đã xong -> Tắt service -> Hàm `sendEmail` bị giết giữa chừng.


2. **Socket.io Connections (`mobile-socketio-gateway`):**
* Tracking của ServiceBroker không quản lý các kết nối WebSocket đang mở. Bạn phải tự handle việc này trong `stopped()` (ví dụ: gửi event báo Client reconnect sang node khác).



---

### 3. Tầng Go Service: Sự khác biệt về Context Cancellation

Với các service Go (`reconcile-service`, `disbursement-service`), cơ chế Graceful Shutdown khó hơn Node.js vì Go là đa luồng thực sự (Goroutines).

**Vấn đề: Context Propagation**
Trong Go, chúng ta thường truyền `context.Context` từ HTTP Request xuống DB Query.

```go
// Code lỗi thường gặp
func handler(w http.ResponseWriter, r *http.Request) {
    // Khi Server shutdown, ctx của request này bị Cancel NGAY LẬP TỨC
    ctx := r.Context() 
    
    // -> Câu query này sẽ fail ngay lập tức với lỗi "context canceled"
    // Dù server chưa tắt hẳn, nhưng nó ép request dừng.
    db.QueryContext(ctx, "UPDATE wallets SET balance = ...") 
}

```

**Phân tích kỹ:**
Khi Server Go nhận tín hiệu Shutdown, hành vi mặc định của nhiều framework là cancel context của các request đang chạy để giải phóng tài nguyên. Điều này dẫn đến giao dịch thất bại dù Pod vẫn còn sống.

**Giải pháp sâu:**
Cần tách biệt Context cho các tác vụ quan trọng (Critical Section).

* Nếu đang thực hiện trừ tiền/ghi DB, không được dùng `Request Context` (dễ bị kill), mà nên dùng `Background Context` hoặc cơ chế detaching context để đảm bảo câu query chạy xong dù HTTP connection đã đứt.

---

### 4. Tầng Database & Connector (Connection Draining)

**Vấn đề: Keep-Alive Connection**
Giả sử Service A gọi Service B (qua HTTP Keep-Alive).

* Service B tắt. Service A vẫn đang giữ kết nối TCP mở tới Service B (do cơ chế reuse connection).
* Service A gửi request tiếp theo vào đường ống đó -> Lỗi `RESET` (Connection Reset by Peer).

**Phân tích kỹ:**
Trong `stopped()` hook của Moleculer hoặc `Shutdown` của Go, bạn cần:

1. **DB Pool:** Đừng gọi `db.close()` ngay lập tức. Hãy để Pool tự drain (đợi các connection đang mượn được trả về) rồi mới close.
2. **HTTP Server:** Phải set header `Connection: close` cho các response cuối cùng trong giai đoạn shutting down để báo client (Gateway/Upstream) biết đường mà ngắt kết nối.

---

### TỔNG HỢP KỊCH BẢN THẤT BẠI (EDGE CASES)

Dù bạn cấu hình Graceful Shutdown hoàn hảo, hệ thống vẫn sẽ lỗi trong các trường hợp sau (Cần phương án dự phòng):

1. **Lỗi OOM (Out of Memory):** Pod bị Kernel giết ngay lập tức (`SIGKILL`). Không có hook nào chạy được.
* *Fix:* Cấp phát dư RAM, set limit hợp lý.


2. **Bank Connector Timeout quá dài:**
* Moleculer chờ 60s. Bank xử lý mất 61s.
* Moleculer buộc phải tắt (Force Kill).
* Bank trả kết quả thành công -> Hệ thống bạn không nhận được -> **Lệch tiền**.
* *Fix:* Đây là lý do tại sao **Reconcile Service (Đối soát)** và **Job Sweeper** là bắt buộc. Graceful Shutdown chỉ giảm thiểu rủi ro, không loại bỏ hoàn toàn.



### KẾT LUẬN CHO HỆ THỐNG CỦA BẠN

Với 60 services hỗn hợp, chiến lược "Phân tích sâu" dẫn đến hành động cụ thể sau:

1. **Mobile/Web Gateway:** Tập trung vào việc ngắt kết nối WebSocket êm đẹp và `preStop` sleep để LB cập nhật.
2. **Core Transaction (Node.js):** Dựa hoàn toàn vào `Context Tracking` nhưng phải rà soát code để loại bỏ các "Floating Promise" (Promise không await).
3. **Go Service:** Review lại code xử lý `Context` trong các hàm DB, đảm bảo không bị cancel ngang xương khi server shutdown.
4. **Bank Connector:** Chấp nhận thực tế là sẽ có giao dịch bị đứt, tập trung vào cơ chế **Idempotency** (để retry an toàn) và **Reconcile**.

----


Đây là bài toán khó nhất trong Microservices: **Distributed Transaction Consistency** trong quá trình deployment.

Khi một giao dịch đi qua dây chuyền: `Gateway` -> `Node Auth` -> `Node Payment` -> `Go Disbursement` -> `Bank`:

* Nếu bạn restart `Go Disbursement` đúng lúc `Node Payment` vừa gọi sang, kết nối đứt -> **Giao dịch gãy**.
* Graceful Shutdown từng node chỉ bảo vệ *nội bộ* node đó, không bảo vệ *mối liên kết* giữa các node.

Dưới đây là 3 tầng giải pháp để bảo vệ luồng giao dịch đi qua nhiều service (Node & Go) hỗn hợp:

### 1. Giải pháp Tầng Code: "Retry Policy" (Cấp cứu nhanh)

Trong môi trường Microservices, lỗi mạng tạm thời (do Pod restart) là điều bình thường. Service gọi (Caller) phải lì lợm hơn.

**Cơ chế:**
Khi `Node Payment` gọi sang `Go Disbursement` mà bị lỗi kết nối (do Go đang restart), `Node Payment` không được fail ngay. Nó phải **thử lại (retry)**.

* **Moleculer (Node.js):** Moleculer hỗ trợ sẵn cơ chế này.
```typescript
// Trong Node Payment Service
await ctx.call("disbursement.transfer", payload, {
    retries: 3, // Thử lại 3 lần
    timeout: 10000,
    // Chỉ retry khi gặp lỗi mạng hoặc service not found (lúc đang restart)
    retryPolicy: {
        enabled: true,
        delay: 1000,      // Chờ 1s
        maxDelay: 5000,   // Tăng dần
        factor: 2,        // Exponential backoff (1s -> 2s -> 4s)
        check: (err) => err && err.retryable // Chỉ retry lỗi cho phép
    }
});

```


* **Tại sao nó giải quyết được Graceful Shutdown?**
Trong quá trình Rolling Update của Kubernetes:
1. Pod Go cũ tắt (Service A gọi -> Lỗi mạng).
2. Service A chờ 1s -> Retry lần 1.
3. Trong 1s đó, Pod Go mới đã khởi động xong.
4. Retry lần 1 thành công -> **Giao dịch được cứu sống.**



---

### 2. Giải pháp Tầng Kiến trúc: "Async Queue / Event-Driven" (Giải pháp triệt để)

Thay vì gọi trực tiếp (Synchronous A gọi B), hãy dùng **Message Queue** (NATS JetStream có sẵn trong hạ tầng của bạn).

**Kịch bản:**

1. `Node Payment` nhận lệnh -> Ghi vào Queue: `topic: disbursement.requested`.
2. `Go Disbursement` (Worker) lắng nghe Queue -> Nhận job -> Xử lý.

**Tác dụng khi Shutdown:**

* Nếu `Go Service` đang restart, message vẫn **nằm yên trong Queue**.
* Không có kết nối trực tiếp nào bị đứt.
* Khi `Go Service` mới khởi động lên, nó subscribe lại vào Queue và xử lý tiếp các message đang chờ.
* **Kết quả:** Zero data loss. Giao dịch chỉ bị chậm đi vài giây chứ không bao giờ lỗi.

**Áp dụng cho hệ thống của bạn:**

* Dùng **NATS JetStream** (bền vững hơn NATS thường).
* Node.js (Moleculer) publish message.
* Go service subscribe group queue.

---

### 3. Giải pháp Tầng Hạ tầng: "Readiness Probes" (Điều tiết giao thông)

Kubernetes có 2 loại đầu dò: `Liveness` (Sống hay chết) và `Readiness` (Sẵn sàng nhận khách chưa).

**Vấn đề:**
Pod Go mới khởi động lên (Process chạy), nhưng chưa kết nối xong Database/Redis. Nếu K8s dẫn traffic vào ngay -> **Lỗi**.

**Giải pháp:**
Cấu hình `readinessProbe` thật kỹ. Chỉ khi nào Service kết nối xong DB, NATS, Redis thì mới báo "Ready".

```yaml
# Trong deployment.yaml của Go Service & Node Service
readinessProbe:
  httpGet:
    path: /health/ready # Viết API check kết nối DB/Redis
    port: 3000
  initialDelaySeconds: 5
  periodSeconds: 5

```

**Tác dụng:**
K8s sẽ **không tắt Pod cũ** cho đến khi Pod mới báo `Ready`. Đảm bảo lúc nào cũng có ít nhất 1 Pod khỏe mạnh để xử lý giao dịch.

---

### 4. Yêu cầu BẮT BUỘC: Idempotency (Tính Lũy Đẳng)

Dù bạn dùng cách 1 (Retry) hay cách 2 (Queue), rủi ro lớn nhất là **Duplicate Transaction** (Chuyển tiền 2 lần).

* *Ví dụ:* Go Service đã chuyển tiền xong, đang gửi response thì bị crash. Node Payment tưởng lỗi nên Retry -> Go Service làm lại lần nữa -> Mất tiền.

**Giải pháp:**
Trong mọi bảng Database giao dịch (đặc biệt là ở Go Service và Bank Connector), cột `transaction_id` hoặc `request_id` phải là **UNIQUE**.

```go
// Logic trong Go Service
func ProcessTransfer(req TransferRequest) {
    // 1. Check xem requestId này đã xử lý chưa?
    exists := repo.HasProcessed(req.RequestId)
    if exists {
        return repo.GetOldResponse(req.RequestId) // Trả lại kết quả cũ
    }

    // 2. Nếu chưa -> Xử lý
    // ...
}

```

### TỔNG KẾT CHIẾN LƯỢC CHO BẠN

Để xử lý luồng giao dịch đi qua nhiều Service (Node -> Go) an toàn khi Shutdown:

1. **Bước 1 (Hạ tầng):** Cấu hình **Readiness Probe** chuẩn chỉnh để đảm bảo Pod mới thực sự sẵn sàng trước khi Pod cũ tắt.
2. **Bước 2 (Node.js):** Cấu hình **Retry Policy** trong `ctx.call` của Moleculer (với backoff). Đây là cách dễ nhất để cứu các request bị đứt đoạn.
3. **Bước 3 (Go):** Implement **Idempotency Key** để chống lặp đơn khi Node.js retry.
4. **Dài hạn:** Chuyển các điểm gãy nhạy cảm (như gọi sang Disbursement) sang cơ chế **Async Queue (NATS)**.


---

Đây là một câu hỏi rất thực tế và là "nỗi đau" của các hệ thống Microservices khi vận hành.

Câu trả lời ngắn gọn: **Giao dịch SẼ chạy tiếp (nếu có cơ chế Retry/Queue), nhưng trải nghiệm người dùng (UX) sẽ RẤT TỆ nếu bạn giữ nguyên mô hình đồng bộ (Synchronous).**

Hãy phân tích mổ xẻ vấn đề này:

### 1. Vấn đề: "Hiệu ứng Domino" về thời gian chờ

Giả sử bạn có luồng giao dịch đồng bộ: `App` -> `Gateway` -> `Payment` -> `Disbursement` -> `Bank`.

Nếu `Disbursement` đang restart (mất 30s để khởi động và sẵn sàng):

1. **Payment Service** gọi sang, không thấy đâu -> Retry (thử lại).
2. **Cơ chế Retry** (ví dụ: thử 3 lần, mỗi lần chờ 5s) = Mất thêm 15s.
3. **Tổng thời gian:** User phải chờ **45s - 60s**.

**Hậu quả:**

* **Về kỹ thuật:** Gateway hoặc Load Balancer thường có timeout mặc định (ví dụ 60s). Nếu quá 60s, Gateway cắt kết nối -> Trả lỗi `504 Gateway Timeout` về cho App.
* **Về dữ liệu:** Giao dịch ở Backend *có thể* vẫn thành công sau đó (vì Retry thành công), nhưng User lại nhận được thông báo lỗi.
* **Về UX (User Experience):**
* Khách hàng hoang mang: *"Tiền trừ chưa? Lỗi rồi có cần làm lại không?"*
* Nếu khách làm lại -> **Double Charging** (Trừ tiền 2 lần) nếu không chặn Idempotency tốt.



### 2. Giải pháp Cứu vãn Trải nghiệm (UX Strategy)

Để giải quyết việc "chờ quá lâu", bạn buộc phải thay đổi cách giao tiếp với Frontend. Không thể bắt User nhìn vòng quay (spinner) mãi được.

#### Phương án A: Chuyển sang mô hình "Xử lý Bất đồng bộ" (Async Processing) - **Khuyên dùng**

Thay vì đợi xong hết mới báo, hãy báo "Đã tiếp nhận".

* **Luồng đi:**
1. User bấm "Chuyển tiền".
2. Gateway nhận lệnh -> Đẩy vào Queue -> **Trả về ngay lập tức:** `{ status: "PROCESSING", message: "Giao dịch đang xử lý" }`.
3. Frontend hiển thị màn hình: *"Giao dịch đang được thực hiện, vui lòng chờ..."*.


* **Cập nhật trạng thái:**
* **Cách 1 (Socket.io - Bạn đang có):** Khi Backend xử lý xong (dù mất 1-2 phút do restart), bắn sự kiện `TRANSACTION_SUCCESS` về App -> App hiện thông báo thành công.
* **Cách 2 (Polling - Hỏi thăm):** App cứ 5s gọi API `/check-status` một lần để xem xong chưa.


* **Kết quả:**
* User không bị timeout.
* User biết rõ hệ thống đang làm việc, không dám bấm thử lại lung tung.
* Service có restart 5 phút cũng không sao, vì lệnh đã nằm an toàn trong Queue.



#### Phương án B: Xử lý hiển thị khi timeout (Nếu vẫn giữ Synchronous)

Nếu bạn chưa kịp sửa kiến trúc sang Async, bạn phải xử lý ở Frontend và Gateway:

1. **Tăng Timeout:** Cấu hình lại `admin-portal-gateway` và `payment-gateway` để timeout lên tới **90s** (nếu Bank cho phép).
2. **Thông báo khôn khéo (Smart Error Handling):**
* Nếu API trả về lỗi `504 Timeout` (do chờ lâu quá), App **TUYỆT ĐỐI KHÔNG** hiện "Giao dịch thất bại".
* **Phải hiện:** *"Hệ thống không nhận được phản hồi. Vui lòng kiểm tra Lịch sử giao dịch trước khi thử lại."*
* Đây là "câu thần chú" để chặn việc User bấm chuyển tiền lần 2 gây mất tiền oan.



### 3. Minh họa Trải nghiệm người dùng (UX Flow)

Dưới đây là so sánh trải nghiệm giữa 2 cách xử lý khi Service đang restart:

**Trường hợp: Service đang restart mất 30s**

| Tiêu chí | Cách cũ (Đồng bộ) | Cách mới (Async/Queue) |
| --- | --- | --- |
| **Hành động** | User bấm nút, nhìn spinner quay. | User bấm nút, chuyển màn hình ngay. |
| **Thời gian chờ** | Chờ 30s - 60s. | Chờ 0.5s (API phản hồi "Đã nhận"). |
| **Cảm xúc** | Lo lắng, tưởng máy treo. | An tâm, thấy trạng thái "Đang xử lý". |
| **Rủi ro** | Gateway cắt kết nối -> Báo lỗi ảo. | Không bao giờ timeout. |
| **Kết quả** | User tưởng lỗi -> Bấm lại -> Trừ 2 lần. | App tự update trạng thái khi có Socket. |

### 4. Đề xuất hành động cho team của bạn

Dựa trên stack công nghệ `Moleculer` + `Socket.io` + `React` bạn đang có:

1. **Frontend (React/Mobile):**
* Implement cơ chế **Polling** (gọi check status mỗi 5s) hoặc lắng nghe **Socket.io** event cho các giao dịch quan trọng.
* Sửa thông báo lỗi: Gặp timeout -> Cảnh báo check lịch sử.


2. **Backend (Gateway):**
* Với các API quan trọng (Nạp/Rút), chuyển sang trả về `202 Accepted` (Đã nhận) thay vì đợi `200 OK` (Đã xong).


3. **Vận hành:**
* Hạn chế restart vào giờ cao điểm.
* Nếu restart, hãy restart từng instance một (Rolling Update) và đảm bảo `Readiness Probe` được cấu hình đúng để luôn có ít nhất 1 service sống phục vụ khách.



**Tóm lại:** Restart service làm tăng độ trễ (latency). Để User không ức chế, đừng bắt họ chờ kết quả cuối cùng. Hãy xác nhận là "Đã nhận lệnh" và trả kết quả sau qua kênh khác (Socket/Notification).


----



Dựa trên quy mô hệ thống (~60 services, Hybrid Node.js/Go) và các vấn đề đã phân tích, đây là **Implementation Plan (Kế hoạch thực thi)** chi tiết, được chia theo giai đoạn để giảm thiểu rủi ro (Risk Mitigation).

Mục tiêu: **Zero Downtime Deployment** và **Không mất giao dịch**.

---

### 📅 GIAI ĐOẠN 1: ỔN ĐỊNH HẠ TẦNG (INFRASTRUCTURE STABILIZATION)

**Mục tiêu:** Đảm bảo Service tắt/bật mượt mà, không cắt đứt kết nối đột ngột.
**Thời gian ước tính:** 1 tuần.
**Rủi ro:** Thấp (Chủ yếu là config).

#### Bước 1.1: Cấu hình Moleculer (Node.js Services)

* [ ] **Action:** Cập nhật `moleculer.config.ts` cho toàn bộ service (hoặc qua Base Config).
* [ ] **Config:**
```javascript
tracking: {
    enabled: process.env.MOLECULER_TRACKING_ENABLED === 'true',
    shutdownTimeout: parseInt(process.env.MOLECULER_SHUTDOWN_TIMEOUT || '30000')
}

```


* [ ] **Triển khai:** Update biến môi trường trên K8s:
* Các service thường: `MOLECULER_SHUTDOWN_TIMEOUT = 30000` (30s).
* Bank Connectors: `MOLECULER_SHUTDOWN_TIMEOUT = 60000` (60s).



#### Bước 1.2: Cấu hình Kubernetes (K8s Deployment)

* [ ] **Action:** Update file YAML Deployment cho tất cả services.
* [ ] **Config:**
* Thêm `preStop` hook (Quan trọng nhất để tránh lỗi Connection Refused):
```yaml
lifecycle:
  preStop:
    exec:
      command: ["/bin/sh", "-c", "sleep 10"]

```


* Tăng `terminationGracePeriodSeconds`:
* Service thường: **45s** (Lớn hơn timeout 30s + sleep 10s).
* Bank Connectors: **75s**.





#### Bước 1.3: Cập nhật Go Services

* [ ] **Action:** Sửa file `main.go` của 4 service Go (`disbursement`, `reconcile`...).
* [ ] **Code:** Implement `signal.Notify` để bắt `SIGTERM` và dùng `server.Shutdown(ctx)` chờ request hoàn tất (thay vì exit ngay).

---

### 📅 GIAI ĐOẠN 2: TĂNG CƯỜNG ĐỘ TIN CẬY (RESILIENCE HARDENING)

**Mục tiêu:** Giao dịch tự phục hồi nếu mạng bị đứt do restart.
**Thời gian ước tính:** 2 tuần.
**Rủi ro:** Trung bình (Cần test kỹ logic).

#### Bước 2.1: Cấu hình Retry Policy (Moleculer)

* [ ] **Action:** Cấu hình retry khi gọi RPC giữa các service.
* [ ] **Logic:**
* Trong `moleculer.config.ts`:
```javascript
retryPolicy: {
    enabled: true,
    retries: 3,
    delay: 500,
    maxDelay: 2000,
    factor: 2,
    check: (err) => err && err.retryable // Chỉ retry lỗi mạng/503
}

```




* [ ] **Target:** Áp dụng trước cho cặp `Payment Service` -> `Disbursement Service`.

#### Bước 2.2: Implement Idempotency (Chống lặp)

* [ ] **Action:** Rà soát DB Schema của các bảng Giao dịch (Transaction, WalletLog).
* [ ] **Task:**
* Đảm bảo cột `request_id` hoặc `transaction_ref` có **Unique Index**.
* Logic Code (Go & Node): Trước khi insert/update tiền, luôn `SELECT` check xem `request_id` đã tồn tại chưa.



#### Bước 2.3: Readiness Probes

* [ ] **Action:** Thêm endpoint `/health/ready` cho tất cả service.
* [ ] **Logic:** Endpoint này chỉ trả về `200 OK` khi:
* Đã kết nối DB thành công.
* Đã kết nối NATS/Redis thành công.


* [ ] **K8s Config:** Cấu hình `readinessProbe` trỏ vào endpoint này để K8s không điều hướng traffic vào Pod "chưa tỉnh ngủ".

---

### 📅 GIAI ĐOẠN 3: TỐI ƯU TRẢI NGHIỆM (UX IMPROVEMENT)

**Mục tiêu:** Giảm cảm giác chờ đợi, xử lý trường hợp timeout.
**Thời gian ước tính:** 2 - 3 tuần.
**Rủi ro:** Cao (Liên quan đến Frontend & Logic luồng).

#### Bước 3.1: Nâng cấp Frontend (React/Mobile)

* [ ] **Action:** Sửa logic màn hình chờ giao dịch.
* [ ] **Logic:**
* Nếu API pending > 15s: Hiển thị thông báo "Giao dịch đang xử lý, vui lòng chờ...".
* Nếu API lỗi 504 (Timeout): **KHÔNG** báo thất bại. Hiển thị "Vui lòng kiểm tra lịch sử giao dịch".
* (Optional) Thêm cơ chế Polling: Gọi API `/transactions/{id}/status` mỗi 5s để cập nhật trạng thái nếu Socket chưa về.



#### Bước 3.2: Async Response (Backend Gateway)

* [ ] **Action:** Chuyển đổi các API Nạp/Rút tiền quan trọng.
* [ ] **Logic:**
* Thay vì `await` kết quả cuối cùng từ Bank -> Trả về `202 Accepted` sau khi đã đẩy vào Queue/Service xử lý.
* Client lắng nghe Socket event `TRANSACTION_COMPLETED` để update UI.



---

### 📅 GIAI ĐOẠN 4: LƯỚI AN TOÀN (SAFETY NET & OPS)

**Mục tiêu:** Dọn dẹp các giao dịch bị lỗi (vì không có hệ thống nào hoàn hảo 100%).
**Thời gian ước tính:** Song song hoặc sau Giai đoạn 2.

#### Bước 4.1: Cronjob Sweeper

* [ ] **Action:** Viết Job quét giao dịch treo.
* [ ] **Logic:**
* Tìm giao dịch `PENDING` > 15 phút.
* Gọi API tra soát sang Bank/Partner.
* Sync lại trạng thái về DB.



#### Bước 4.2: Admin Tooling

* [ ] **Action:** Thêm nút chức năng trên CMS Portal.
* [ ] **Features:**
* Nút `Check Status` (Gọi tra soát thủ công).
* Nút `Retry` (Chạy lại step bị lỗi).



---

### 🚀 CHIẾN LƯỢC ROLLOUT (TRIỂN KHAI)

Để tránh "sập toàn tập" với 60 service, hãy triển khai theo thứ tự ưu tiên:

1. **Pilot (Ngày 1-2):** Áp dụng Giai đoạn 1 (Config Graceful Shutdown) cho các service ít quan trọng (`notification-service`, `promotion-service`).
* *Test:* Restart thử xem log có báo "Waiting for contexts..." không.


2. **Expansion (Ngày 3-5):** Áp dụng cho nhóm Gateway và Web Portal.
3. **Critical Core (Tuần 2):** Áp dụng cho `payment`, `wallet` và nhóm Go.
* *Lưu ý:* Cần monitoring kỹ (dùng Jaeger/Log) trong lúc deploy nhóm này.


4. **Connectors (Tuần 3):** Áp dụng cho Bank Connectors (nhớ set timeout cao hơn).

### ✅ TỔNG KẾT CHECKLIST CHO DEV LEAD

Bạn hãy copy checklist này đưa vào Sprint Planning:

* [ ] Đã thêm biến môi trường `MOLECULER_TRACKING_ENABLED` chưa?
* [ ] Đã thêm `preStop` hook vào file K8s Deployment chưa?
* [ ] Đã check Unique Index cho bảng Transaction chưa?
* [ ] Frontend đã handle lỗi 504 Timeout chưa?
* [ ] Đã có Cronjob quét giao dịch treo chưa?

----


Đây là **LỘ TRÌNH THỰC THI CHI TIẾT (MASTER IMPLEMENTATION PLAN)** được tổng hợp từ toàn bộ các vấn đề đã thảo luận (Graceful Shutdown, Retry, Idempotency, Queue, Saga, UX).

Lộ trình này được thiết kế để áp dụng cho hệ thống ~60 services (Hybrid Node.js/Go) của bạn, đi từ sửa lỗi hạ tầng (Infrastructure) đến tái cấu trúc lõi (Core Re-architecture).

---

## 📅 GIAI ĐOẠN 0: RÀ SOÁT & CHUẨN BỊ (PREPARATION)

*Mục tiêu: Đảm bảo nền móng vững chắc trước khi thay đổi config.*

### 0.1. Kiểm tra Database Schema (Idempotency Audit)

* [ ] **Rà soát bảng Transaction/WalletLog:** Kiểm tra xem các bảng này đã có **UNIQUE INDEX** cho cột `request_id` (hoặc `reference_code`) chưa.
* *Yêu cầu:* Nếu chưa có -> Tạo migration script đánh index ngay. Đây là "khiên chắn" bắt buộc để chống trừ tiền kép (Double Charging).


* [ ] **Chuẩn hóa Redis Key:** Review cách đặt key lock trong Redis (nếu có dùng distributed lock). Đảm bảo key có TTL (Time-To-Live) để tránh deadlock khi service crash.

### 0.2. Nâng cấp Shared Library (`package` repo)

* [ ] **Thêm Helper Graceful Shutdown:** Viết sẵn các hàm helper trong thư viện chung để các service con chỉ việc import và dùng, tránh sửa code lặp lại 50 lần.
* [ ] **Versioning:** Đóng băng version hiện tại, tạo version mới cho đợt nâng cấp này.

---

## 📅 GIAI ĐOẠN 1: ỔN ĐỊNH HẠ TẦNG (INFRASTRUCTURE STABILIZATION)

*Mục tiêu: Service tắt/bật mượt mà, không cắt kết nối "ngang xương". Xử lý lỗi 502/Connection Refused.*

### 1.1. Cấu hình Graceful Shutdown (Node.js/Moleculer)

*Áp dụng: 55 services Node.js*

* [ ] **Cập nhật `moleculer.config.ts`:**
```typescript
tracking: {
    enabled: process.env.MOLECULER_TRACKING_ENABLED === 'true',
    shutdownTimeout: parseInt(process.env.MOLECULER_SHUTDOWN_TIMEOUT || '30000') // 30s mặc định
}

```


* [ ] **Phân loại Timeout:**
* Nhóm thường (User, Auth...): Timeout **30s**.
* Nhóm Connector (BIDV, Napas...): Timeout **60s** (Do bank phản hồi chậm).



### 1.2. Cấu hình Graceful Shutdown (Go Services)

*Áp dụng: 4 services Go (`disbursement`, `reconcile`...)*

* [ ] **Refactor `main.go`:**
* Sử dụng `signal.Notify` lắng nghe `SIGTERM`.
* Dùng `server.Shutdown(ctx)` thay vì để chương trình tự exit.
* **Quan trọng:** Tách `Context` của DB Query ra khỏi `Context` của HTTP Request (Background Context) để query không bị kill khi HTTP connection đóng.



### 1.3. Cấu hình Kubernetes (Deployment YAML)

*Áp dụng: Toàn bộ 60 services*

* [ ] **Thêm `preStop` Hook:**
```yaml
lifecycle:
  preStop:
    exec:
      command: ["/bin/sh", "-c", "sleep 10"] # "Câu giờ" cho Load Balancer cập nhật IP

```


* [ ] **Tăng `terminationGracePeriodSeconds`:**
* Nhóm thường: **45s** ( > 30s timeout + 10s sleep).
* Nhóm Connector: **75s**.


* [ ] **Cấu hình `readinessProbe`:**
* Trỏ vào API `/health/ready` (API này phải check connect DB/Redis/NATS).
* Chỉ cho traffic vào khi Service thực sự sẵn sàng.



---

## 📅 GIAI ĐOẠN 2: LƯỚI AN TOÀN DỮ LIỆU (RESILIENCE & SAFETY NET)

*Mục tiêu: Nếu mạng đứt do restart, hệ thống tự thử lại. Nếu thử lại thất bại, có tool quét dọn.*

### 2.1. Cấu hình Retry Policy (Service-to-Service)

* [ ] **Moleculer Config:** Bật retry policy với exponential backoff.
```javascript
retryPolicy: {
    enabled: true,
    retries: 3,
    delay: 500,
    maxDelay: 2000,
    factor: 2,
    check: (err) => err && err.retryable // Chỉ retry lỗi mạng/503
}

```


* [ ] **Go Client:** Nếu Go gọi ngược lại Node, cấu hình HTTP Client có Retry (dùng thư viện như `go-retryablehttp`).

### 2.2. Implement Job "Quét rác" (Transaction Sweeper)

* [ ] **Tạo Service mới hoặc Module:** `reconcile-service` (Go) hoặc `scheduler-service`.
* [ ] **Logic Cronjob (Chạy 5-10 phút/lần):**
1. Scan DB tìm giao dịch `PENDING` > 15 phút.
2. Gọi API tra soát sang Bank/Partner.
3. Update trạng thái cuối cùng (SUCCESS/FAILED) vào DB.
4. Alert vào Slack/Telegram nếu giao dịch vẫn treo không xác định.



### 2.3. Admin Tooling (Cứu hộ thủ công)

* [ ] **CMS Portal:** Thêm module quản lý giao dịch treo.
* [ ] **Chức năng:**
* `Check Status`: Gọi tra soát real-time.
* `Force Success/Fail`: Cập nhật trạng thái bằng tay (có Audit Log lý do).



---

## 📅 GIAI ĐOẠN 3: TÁI CẤU TRÚC ASYNC & SAGA (CORE RE-ARCHITECTURE)

*Mục tiêu: Decoupling hệ thống. Chuyển đổi các luồng quan trọng (Payment -> Disbursement) sang Bất đồng bộ.*

### 3.1. Chuyển đổi sang NATS JetStream (Queue)

*Thay thế RPC `ctx.call` trực tiếp giữa Node và Go.*

* [ ] **Thiết lập Stream:** Tạo Stream `EVENTS_PAYMENT` trên NATS JetStream (Lưu file, Replicas 3).
* [ ] **Producer (Node.js - Payment Service):**
* Thay vì gọi `disbursement.transfer`, hãy Publish event:
`payment.deducted` (Payload: `{ tx_id, amount, bank_info }`).


* [ ] **Consumer (Go - Disbursement Service):**
* Viết Worker Subscribe `payment.deducted`.
* **Logic:** Nhận message -> Chuyển tiền Bank -> Update DB.
* **Manual ACK:** Chỉ gửi ACK cho NATS khi DB đã update xong. Nếu crash, message tự động được gửi lại cho Pod khác.



### 3.2. Implement Saga Pattern (Orchestrator)

*Xử lý hoàn tiền tự động khi lỗi.*

* [ ] **Saga Coordinator:** (Nên đặt tại `payment-service` hoặc tách service riêng).
* [ ] **Định nghĩa State Machine:**
* Step 1: `Wallet Deduct` (Node) -> Done.
* Step 2: `Bank Transfer` (Go) -> Failed?
* Step 3 (Compensate): `Wallet Refund` (Node) -> Triggered.


* [ ] **Xử lý Rollback:**
* Nếu Consumer (Go) báo lỗi (hoặc hết retry queue) -> Publish event `disbursement.failed`.
* Coordinator nhận event -> Gọi action hoàn tiền vào ví user.



---

## 📅 GIAI ĐOẠN 4: TỐI ƯU TRẢI NGHIỆM & GIÁM SÁT (UX & OPS)

*Mục tiêu: Khách hàng không ức chế khi chờ đợi, Dev dễ debug.*

### 4.1. Nâng cấp Frontend (React/Mobile)

* [ ] **Xử lý Timeout 504:**
* Catch lỗi HTTP 504 từ Gateway.
* Hiển thị: *"Hệ thống đang xử lý, vui lòng kiểm tra lịch sử"* (Thay vì báo đỏ "Thất bại").


* [ ] **Cơ chế Polling/Socket:**
* Sau khi submit, App lắng nghe Socket event `TX_COMPLETED`.
* Fallback: Gọi API `/status` mỗi 5s để cập nhật trạng thái nếu Socket mất.



### 4.2. API Gateway (Async Response)

* [ ] **Update Endpoint Nạp/Rút:**
* Trả về HTTP 202 (Accepted) ngay khi đẩy job vào Queue thành công.
* Body: `{ status: "PROCESSING", request_id: "..." }`.



### 4.3. Observability (Giám sát)

* [ ] **Distributed Tracing (Jaeger):**
* Tích hợp Moleculer Tracing.
* Đảm bảo `request_id` được truyền xuyên suốt: Gateway -> Node -> NATS -> Go -> DB.


* [ ] **Alerting:**
* Cảnh báo khi Queue backlog tăng cao (Go service xử lý chậm).
* Cảnh báo khi tỷ lệ rollback (Saga compensation) > 1%.



---

## 🚦 TIÊU CHÍ NGHIỆM THU (SUCCESS METRICS)

| Giai đoạn | Tiêu chí hoàn thành (Definition of Done) |
| --- | --- |
| **GĐ 1** | Restart 1 service bất kỳ (kể cả Go) không gây lỗi 502 cho Gateway. Log không báo lỗi "Socket Hang up". |
| **GĐ 2** | Tắt Service Disbursement (Go) 1 phút rồi bật lại, các request từ Payment (Node) tự động retry thành công, không mất đơn. |
| **GĐ 3** | Rút dây mạng Service Go khi đang xử lý, sau khi bật lại, giao dịch trong Queue tiếp tục chạy. Nếu lỗi Bank, tiền ví tự hoàn về (Saga). |
| **GĐ 4** | User không bao giờ thấy thông báo "Lỗi không xác định" khi server restart. Admin có tool để xử lý các ca lỗi hiếm gặp. |

### ĐỀ XUẤT THỨ TỰ TRIỂN KHAI CHO 60 SERVICES

1. **Nhóm Pilot:** `notification-service`, `promotion-service` (Ít rủi ro).
2. **Nhóm Gateway:** `admin-portal-gateway`, `merchant-portal-gateway`.
3. **Nhóm Core:** `user`, `auth`, `profile`.
4. **Nhóm Finance (Critical):** `wallet`, `payment`, `disbursement` (Go).
5. **Nhóm Connector:** `bidv`, `napas`... (Cần test timeout kỹ nhất).