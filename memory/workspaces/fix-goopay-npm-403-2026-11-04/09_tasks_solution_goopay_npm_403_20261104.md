---
tier: T2
suffix: goopay_npm_403_20261104
workspace: fix-goopay-npm-403-2026-11-04
created_at: 2026-11-04T09:46:30+07:00
status: PROPOSED
---

# 09. Tasks Solution — Xử lý dứt điểm lỗi `yarn install` trả về `403` từ `https://npm.goopay.vn/@goopay%2fgoopay-library`

## 1. Bối cảnh & Ràng buộc Thực thi (Technical Constraints)
- **L004 (Single-Blocker Laser Focus):** Tập trung giải quyết trực diện lỗi `403 Forbidden` khi chạy `yarn install` đối với gói `@goopay/goopay-library`.
- **L006 (Verbatim Current-File Code Demo):** Trích dẫn nguyên văn nội dung hiện tại tại HEAD của `/Users/trainguyen/Documents/work/customer-service/package.json:34-40` và `/Users/trainguyen/Documents/work/package/package.json:1-19`.
- **L015 (Propose-Only Discipline):** Giữ nguyên chế độ Read-Only, trình bày Code Demo trực tiếp trên khung chat và dừng chờ `APPROVE` tường minh.
- **L032 (1-to-1 Finding Mapping):** Ánh xạ giải pháp 1-1 theo từng phát hiện `F1` và `F2`.

---

## 2. Phân tích Nguyên nhân Gốc rễ (`VERIFIED` Findings `F1..F2`)

### Phát hiện `F1`: Máy chủ Private Registry `https://npm.goopay.vn/` chặn `HTTP 403 Forbidden` tại tầng OpenResty Reverse Proxy và thiếu entry `@goopay/goopay-library@^1.2.0` trong `yarn.lock`
- **Bằng chứng (`VERIFIED`):**
  1. Tại `/Users/trainguyen/Documents/work/customer-service/package.json:37`, dependency được khai báo là `"goopay-library": "npm:@goopay/goopay-library@^1.2.0"`.
  2. Tại `/Users/trainguyen/Documents/work/customer-service/.npmrc:1` và `~/.yarnrc:5`, scope `@goopay:registry` trỏ về `https://npm.goopay.vn/`.
  3. Tại `/Users/trainguyen/Documents/work/customer-service/yarn.lock:1858-1861`, lockfile hiện tại chỉ có entry `goopay-library@^1.0.1` (`https://registry.npmjs.org/goopay-library/-/goopay-library-1.0.1.tgz`), hoàn toàn chưa có entry cho `goopay-library@npm:@goopay/goopay-library@^1.2.0`.
  4. Khi chạy `yarn` / `yarn install`, Yarn gửi request `GET https://npm.goopay.vn/@goopay%2fgoopay-library` để phân giải phiên bản `^1.2.0`. Máy chủ `https://npm.goopay.vn/` lập tức trả về `HTTP/2 403 Forbidden` từ tầng **OpenResty** (`<center><h1>403 Forbidden</h1></center><hr><center>openresty</center>`, `server: GooPay`) cho mọi request từ IP hiện tại (chặn IP Whitelist/VPN ở tầng cổng proxy, đồng thời `.npmrc` cũng không có `//npm.goopay.vn/:_authToken`).

### Phát hiện `F2`: `customer-service` bắt buộc cần `@goopay/goopay-library@1.2.0` (chứa `field-crypto.*`) và nguồn `1.2.0` chính chủ đang mở sẵn qua GitLab SSH (`git@gitlab.goopay.vn:goopay-services/package.git`) + `/Users/trainguyen/Documents/work/package`
- **Bằng chứng (`VERIFIED`):**
  1. Tại `/Users/trainguyen/Documents/work/customer-service/lib/encryption.config.ts:5-7`, dịch vụ nạp trực tiếp:
     - `require("goopay-library/helpers/field-crypto.helper")`
     - `require("goopay-library/helpers/field-crypto.middleware")`
     Do đó không thể hạ cấp về `goopay-library@^1.0.1` (bản public trên `registry.npmjs.org`) hay bản `1.1.9` cũ vì hai bản đó chưa có bộ `field-crypto.*`.
  2. Kiểm tra kho mã nguồn chính thức của `@goopay/goopay-library` tại `/Users/trainguyen/Documents/work/package` (`origin: git@gitlab.goopay.vn:goopay-services/package.git`):
     - Kết nối SSH tới `git@gitlab.goopay.vn:goopay-services/package.git` hoạt động thông suốt 100%.
     - Nhánh `origin/main` (`8c666d016eae159682de32b2a16c3209b8a840e0`) trên GitLab chính là phiên bản **`1.2.0`** có đầy đủ `helpers/field-crypto.helper.js` và `helpers/field-crypto.middleware.js`, trong khi thư mục local `/Users/trainguyen/Documents/work/package/package.json:1-3` đang dừng ở commit cũ `559d8b6` (`1.1.9`).
     - Các service cùng hệ thống như `/Users/trainguyen/Documents/work/export-service/package.json:27` và `/Users/trainguyen/Documents/work/storage-gateway/package.json:22` đã áp dụng chuẩn `"goopay-library": "file:../package"` để cài đặt trực tiếp từ `/Users/trainguyen/Documents/work/package` mà không bị chặn bởi `https://npm.goopay.vn/`.

---

## 3. Phương án Kỹ thuật Duy nhất & Tối ưu Nhất (Ánh xạ 1-1 với `F1..F2`)

### 3.1. Giải pháp cho `F1` — Đồng bộ `@goopay/goopay-library@1.2.0` từ GitLab SSH về `/Users/trainguyen/Documents/work/package`
- Chạy `git -C /Users/trainguyen/Documents/work/package pull origin main` để cập nhật kho `/Users/trainguyen/Documents/work/package` từ `1.1.9` (`559d8b6`) lên `1.2.0` (`8c666d016eae159682de32b2a16c3209b8a840e0`), nạp đầy đủ `helpers/field-crypto.helper.js` và `helpers/field-crypto.middleware.js` về đĩa local qua kênh GitLab SSH (bỏ qua hoàn toàn cổng HTTP `https://npm.goopay.vn/` đang bị OpenResty chặn `403`).

### 3.2. Giải pháp cho `F2` — Trỏ `goopay-library` trong `/Users/trainguyen/Documents/work/customer-service/package.json:37` sang `"file:../package"` (đồng bộ chuẩn `export-service` & `storage-gateway`) và chạy `yarn install`
- Cập nhật `/Users/trainguyen/Documents/work/customer-service/package.json:37` từ `"goopay-library": "npm:@goopay/goopay-library@^1.2.0"` sang `"goopay-library": "file:../package"`.
- Chạy `yarn install` trong `/Users/trainguyen/Documents/work/customer-service` để Yarn liên kết trực tiếp bản `@goopay/goopay-library@1.2.0` vào `node_modules/goopay-library` và cập nhật `yarn.lock` sạch lỗi `403`.

---

## 4. Bảng Ánh xạ Điểm Gọi & Luồng Thực thi (Call-Site & Lifecycle Wiring)

| Thành phần | Vị trí `file:line` | Vai trò / Thời điểm kích hoạt |
| :--- | :--- | :--- |
| Nguồn thư viện nội bộ | `/Users/trainguyen/Documents/work/package/package.json:1-19` | Kho Git gốc của `@goopay/goopay-library` (`git@gitlab.goopay.vn:goopay-services/package.git`), được cập nhật lên `1.2.0` trước khi chạy `yarn install`. |
| Khai báo Dependency | `/Users/trainguyen/Documents/work/customer-service/package.json:34-40` | Điểm khai báo gói `goopay-library` được `yarn install` đọc ở pha phân giải gói (Package Resolution Stage). |
| Caller Runtime | `/Users/trainguyen/Documents/work/customer-service/lib/encryption.config.ts:4-8` | Nạp `goopay-library/helpers/field-crypto.helper` và `goopay-library/helpers/field-crypto.middleware` khi khởi động `customer-service` / chạy `vitest`. |
| Startup Validation | `/Users/trainguyen/Documents/work/customer-service/start.ts:441` -> `/Users/trainguyen/Documents/work/customer-service/lib/sensitive-crypto.ts:474-484` -> `/Users/trainguyen/Documents/work/customer-service/node_modules/goopay-library/helpers/field-crypto.helper.js:82-92,222-227` | Kiểm tra `ENCRYPTION_ACTIVE_VERSION={n}` phải có biến môi trường `ENCRYPTION_KEY_V{n}` tương ứng (32 ký tự UTF-8 hoặc 64 ký tự hex) trong `.run.local.env:38-40` khi khởi động service. |

---

## 5. Phát hiện `F3` & Giải pháp cho `F3` (Lỗi `FieldCrypto: ENCRYPTION_ACTIVE_VERSION or ENCRYPTION_KEY_V{n} is not configured` lúc khởi động)

### 5.1. Phát hiện `F3` (`VERIFIED`): Lệch số hiệu phiên bản giữa `ENCRYPTION_ACTIVE_VERSION=2` và `ENCRYPTION_KEY_V1` trong `/Users/trainguyen/Documents/work/customer-service/.run.local.env:38-40`
- **Bằng chứng (`VERIFIED`):**
  1. Tại `/Users/trainguyen/Documents/work/customer-service/node_modules/goopay-library/helpers/field-crypto.helper.js:15-36,82-92`:
     - `isConfigured()` gọi `const active = parseActiveVersion()` (đọc `process.env.ENCRYPTION_ACTIVE_VERSION`), sau đó gọi `resolveKeyForVersion(active)` để tìm đúng biến `process.env["ENCRYPTION_KEY_V" + active]`.
  2. Kiểm tra `/Users/trainguyen/Documents/work/customer-service/.run.local.env:38-40` (được đưa vào từ commit `80060a887d3150d7dceb2d886c954560347009f7`):
     - Dòng 39 đặt `ENCRYPTION_ACTIVE_VERSION=2`.
     - Dòng 40 chỉ khai báo `ENCRYPTION_KEY_V1=<64_hex_chars>` mà **không có `ENCRYPTION_KEY_V2`** (hoặc `ENCRYPTION_ACTIVE_VERSION` bị để lệch là `2` thay vì `1`).
  3. Khi `start.ts:441` gọi `assertCustomerFieldCryptoAtStartup(this.logger)` (`lib/sensitive-crypto.ts:483`), `resolveKeyForVersion(2)` tìm `process.env.ENCRYPTION_KEY_V2` nhận `undefined` -> trả về `null` -> `isConfigured()` trả về `false` -> văng lỗi `Error: FieldCrypto: ENCRYPTION_ACTIVE_VERSION or ENCRYPTION_KEY_V{n} is not configured` tại `field-crypto.helper.js:226`.

### 5.2. Giải pháp cho `F3` — Đồng bộ `ENCRYPTION_ACTIVE_VERSION=1` (và khai báo kèm `ENCRYPTION_KEY_V2` phòng trường hợp DB dev/local có bản ghi đã mã hóa với prefix `gpd:v2:`/`gp:v2:`)
- Tại `/Users/trainguyen/Documents/work/customer-service/.run.local.env:38-40`:
  - Đổi `ENCRYPTION_ACTIVE_VERSION=1` để khớp trực tiếp với `ENCRYPTION_KEY_V1` hiện có (hoặc thêm dòng `ENCRYPTION_KEY_V2=<64_hex_chars>` nếu cần giữ `ENCRYPTION_ACTIVE_VERSION=2`).
  - Khai báo đồng thời cả `ENCRYPTION_KEY_V1` và `ENCRYPTION_KEY_V2` (cùng giá trị key 64-hex hiện tại trong `.run.local.env:40`, tương tự pattern tại `lib/sensitive-crypto.test.ts:187-188`) giúp vừa vượt qua `assertConfiguredAtStartup()` vừa giải mã (`decryptValue`) được cả bản ghi `gpd:v1:...` lẫn `gpd:v2:...` trong DB local/dev.

---

## 6. Code Demo (Verbatim HEAD Diff)

### 6.1. Cập nhật `/Users/trainguyen/Documents/work/customer-service/.run.local.env:38-40` (Giá trị key được mask theo Rule #20)
```diff
--- a/.run.local.env
+++ b/.run.local.env
@@ -36,5 +36,6 @@ FRAUD_RULE_DOMAIN_ID=AML
 
 # Field crypto (goopay-library FieldCryptoHelper)
-ENCRYPTION_ACTIVE_VERSION=2
+ENCRYPTION_ACTIVE_VERSION=1
 ENCRYPTION_KEY_V1=<GIỮ_NGUYÊN_KEY_64_HEX_HIỆN_TẠI>
+ENCRYPTION_KEY_V2=<GIỮ_NGUYÊN_KEY_64_HEX_HIỆN_TẠI>
```

---

## 7. Phát hiện `F4` & Giải pháp cho `F4` (Lỗi `Service 'auth-service.signin' is not registered`)

### 7.1. Phát hiện `F4` (`VERIFIED`): Thiếu biến `MERCHANT_API_KEY_JWT_SECRET` trong `/Users/trainguyen/Documents/work/auth-service/.run.local.env` khiến `auth-service` tự gọi `this.broker.destroyService(this)` ngay lúc khởi động
- **Bằng chứng (`VERIFIED`):**
  1. Tiến trình `auth-service` (`PID 38489`) vẫn đang chạy trên máy nhưng khi `admin-portal-gateway` gọi `auth-service.signin`, Broker báo `Service 'auth-service.signin' is not registered`.
  2. Tại `/Users/trainguyen/Documents/work/auth-service/svc-env.js:43`, hàm `getRequiredKey()` bắt buộc phải có khóa `"MERCHANT_API_KEY_JWT_SECRET"`.
  3. Tại `/Users/trainguyen/Documents/work/auth-service/start.js:410-425`:
     - Khi chạy local (không kết nối được `dev-vault.goopay.vn`), `VaultSecurity.checkMissingEnvVariable()` (`/Users/trainguyen/Documents/work/auth-service/security/vault.security.js:18-28`) đối chiếu các khóa trong `/Users/trainguyen/Documents/work/auth-service/.run.local.env` với `SVC_ENV.getRequiredKey()`.
     - Kiểm tra thực tế `/Users/trainguyen/Documents/work/auth-service/.run.local.env` cho thấy khóa duy nhất bị thiếu là **`MERCHANT_API_KEY_JWT_SECRET`**.
     - Khi `isMissing === true`, `start.js:416` gọi **`this.broker.destroyService(this)`** và `start.js:423` gọi lại `this.broker.destroyService(this)` trong `.catch(...)`. Hệ quả là `auth-service` bị gỡ hoàn toàn khỏi Moleculer Broker (trong khi tiến trình `moleculer-runner` vẫn treo ở `mol $`), làm mọi lời gọi `auth-service.signin` bị lỗi `Service 'auth-service.signin' is not registered`.

### 7.2. Giải pháp cho `F4` — Bổ sung `MERCHANT_API_KEY_JWT_SECRET` vào `/Users/trainguyen/Documents/work/auth-service/.run.local.env` và khởi động lại `auth-service`
- Thêm dòng `MERCHANT_API_KEY_JWT_SECRET=<secret>` vào cuối `/Users/trainguyen/Documents/work/auth-service/.run.local.env` (lấy giá trị chuẩn từ `/Users/trainguyen/Documents/work/data-hub/goopay-testing/testing-application/onpremise-fpt/__secrets/auth-service:1`).
- Khởi động lại tiến trình `auth-service` (`yarn local` / `npm run local`) để `moleculer-runner --envfile .run.local.env` nạp biến môi trường mới vào `process.env` và đăng ký `auth-service.signin` lên NATS (`GOOPAY-CORE`).

### 7.3. Code Demo cho `F4` (`/Users/trainguyen/Documents/work/auth-service/.run.local.env:38-41`, giá trị secret được mask theo Rule #20)
```diff
--- a/.run.local.env
+++ b/.run.local.env
@@ -38,3 +38,4 @@ ORG_APP_ID= 611a2ec6eccd622f978bbee0
 MOBILE_KEY_JWT=<MASKED>
 PORTAL_KEY_JWT=<MASKED>
 SALT_ROUNDS=10
+MERCHANT_API_KEY_JWT_SECRET=<GIÁ_TRỊ_TỪ_ONPREMISE_FPT_SECRETS_AUTH_SERVICE>
```

---

## 8. Phát hiện `F5` & Giải pháp cho `F5` (Lỗi `ServiceNotAvailableError: Service 'config-service.getAllSystemConfigsByIds' is not available`)

### 8.1. Phát hiện `F5` (`VERIFIED`): `MONGO_URI` trong `/Users/trainguyen/Documents/work/config-service/.run.local.env:7` trỏ vào `localhost:27020` (không có MongoDB lắng nghe -> `ECONNREFUSED 127.0.0.1:27020`), khiến `config-service` tự gọi `this.stopService(this.broker, "STOP_SERVICE_ERR_CONNECT_DB")` -> `broker.destroyService(...)` ngay khi khởi động
- **Bằng chứng (`VERIFIED`):**
  1. Tiến trình `config-service` (`PID 52405`) đang mở trên máy nhưng action `config-service.getAllSystemConfigsByIds` (đăng ký tại `/Users/trainguyen/Documents/work/config-service/start.ts:102`) không tồn tại trên NATS Broker.
  2. Kiểm tra `/Users/trainguyen/Documents/work/config-service/.run.local.env:7`:
     - Đang cấu hình `MONGO_URI=mongodb://***:***@localhost:27020/config-service?authSource=admin`.
     - Kiểm tra socket thực tế trên máy: cổng `127.0.0.1:27020` không có tiến trình nào lắng nghe (`connect ECONNREFUSED 127.0.0.1:27020`), trong khi các service đang chạy cùng phiên (`auth-service/.run.local.env:9`, `user-service/.run.local.env:7`, `customer-service/.run.local.env:8`) đều kết nối tới cụm MongoDB dev `10.200.186.11:27017,10.200.186.12:27017,10.200.186.13:27017/...?replicaSet=goopay&authSource=admin`.
  3. Tại `/Users/trainguyen/Documents/work/config-service/start.ts:446-452` và `:502-506`:
     - Khi `dbHandler.createConnection` (`/Users/trainguyen/Documents/work/config-service/dbHandler/mongo.ts:11-29`) trả về lỗi `ECONNREFUSED 127.0.0.1:27020`, `start.ts:450` gọi `this.stopService(this.broker, "STOP_SERVICE_ERR_CONNECT_DB")`.
     - Hàm `stopService` (`start.ts:503`) thực thi `broker.destroyService(broker.services.find(x => x.name === SVC_ENV.get().SERVICE_NAME))` — hủy đăng ký `config-service` khỏi Broker trong khi tiến trình `moleculer-runner` (`PID 52405`) vẫn giữ prompt `mol $`.

### 8.2. Giải pháp cho `F5` — Cập nhật `MONGO_URI` trong `/Users/trainguyen/Documents/work/config-service/.run.local.env:7` sang cụm `10.200.186.11..13/config-service` (khớp với `CONFIG_MONGO_URI` tại `/Users/trainguyen/Documents/work/centralized-export-service/.run.local.env:59`) và khởi động lại `config-service`
- Thay dòng `MONGO_URI` tại `/Users/trainguyen/Documents/work/config-service/.run.local.env:7` bằng chuỗi kết nối `config-service` trên cụm `10.200.186.11:27017,10.200.186.12:27017,10.200.186.13:27017/config-service?replicaSet=goopay&authSource=admin` (đã verify kết nối thành công `readyState === 1`).
- Khởi động lại tiến trình `config-service` (`yarn local` / `npm run local`) để `config-service` kết nối DB thành công, nạp `setBankErrorCodesToRedis()` và giữ action `config-service.getAllSystemConfigsByIds` trên NATS Broker.

### 8.3. Code Demo cho `F5` (`/Users/trainguyen/Documents/work/config-service/.run.local.env:5-9`, thông tin xác thực được mask theo Rule #20)
```diff
--- a/.run.local.env
+++ b/.run.local.env
@@ -5,5 +5,5 @@ REDIS_SENTINELS=localhost:26379
 REDIS_SENTINEL_NAME=mymaster
-MONGO_URI=mongodb://<USER>:<PASS>@localhost:27020/config-service?authSource=admin
+MONGO_URI=mongodb://<USER>:<PASS>@10.200.186.11:27017,10.200.186.12:27017,10.200.186.13:27017/config-service?replicaSet=goopay&authSource=admin
 PORT=3001
 TRANSPORTER=nats://localhost:4222
```

### 5.1. Cập nhật `/Users/trainguyen/Documents/work/package` lên `origin/main` (`1.2.0`)
```bash
git -C /Users/trainguyen/Documents/work/package pull origin main
```

Sau khi pull, `/Users/trainguyen/Documents/work/package/package.json:1-19` cập nhật từ `1.1.9` lên `1.2.0`:
```diff
--- a/package.json
+++ b/package.json
@@ -1,19 +1,22 @@
 {
   "name": "@goopay/goopay-library",
-  "version": "1.1.9",
+  "version": "1.2.0",
   "description": "GooPay Library",
   "main": "index.js",
+  "scripts": {
+    "test": "node --test test/*.spec.js"
+  },
   "dependencies": {
     "axios": "^1.7.9",
     "dateformat": "^4.6.3",
     "http-status-codes": "^2.3.0",
     "json-diff": "^1.0.6",
-    "lodash": "^4.17.21",
+    "lodash": "^4.18.1",
     "mongo-sanitize": "^1.1.0",
     "mongoose": "^7.7.0",
     "randomatic": "^3.1.1",
     "dayjs": "^1.11.7"
   },
   "author": "GooPay",
   "license": "ISC"
 }
```

### 5.2. Cập nhật `/Users/trainguyen/Documents/work/customer-service/package.json:34-40`
```diff
--- a/package.json
+++ b/package.json
@@ -34,7 +34,7 @@
   "dependencies": {
     "bcrypt": "^5.1.1",
     "dotenv": "^10.0.0",
-    "goopay-library": "npm:@goopay/goopay-library@^1.2.0",
+    "goopay-library": "file:../package",
     "hi-base32": "^0.5.1",
     "ioredis": "^4.28.5",
     "jsonwebtoken": "^9.0.2",
```
