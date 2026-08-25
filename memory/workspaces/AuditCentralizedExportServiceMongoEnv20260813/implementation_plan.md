# Implementation Plan: Audit MongoDB ENV URL for centralized-export-service (Updated)

## 1. Yêu cầu từ User
- Tìm hiểu cấu hình MongoDB ENV URL hiện tại của dự án `centralized-export-service`.
- Đảm bảo khi chạy local, tất cả các cấu hình MongoDB trỏ về MongoDB localhost chạy ở local máy của user.
- Giữ nguyên sự phân tách database cho từng alias (ví dụ `default` kết nối database `centrallized-export-service`, alias `payment-bill` kết nối database `payment-bill-service`).
- Xem xét khả năng thu gọn cấu hình trong file `.run.local.env`.

## 2. Kết quả phân tích & Đánh giá tính khả thi
- **Khả thi:** Có khả thi 100%.
- **Vấn đề cấu hình:** 
  - Dự án có nhiều alias (default, payment-bill, payment, v.v.). Mỗi alias được thiết kế chọc vào database riêng để cô lập dữ liệu.
  - Các biến URI kết nối MongoDB được check validation bắt buộc tại `VaultSecurity.checkMissingEnvVariable()` lúc khởi động. Nếu thiếu bất kỳ biến nào trong `REQUIRED_KEYS` thì app sẽ crash.
  - Do đó, nếu xoá các URI khác và chỉ để lại `MONGO_URI` trong file `.run.local.env`, ứng dụng sẽ crash.

## 3. Các giải pháp đề xuất

### Giải pháp 1: Khai báo thủ công các Mongo URI trỏ về Local MongoDB (Không sửa code - Khuyên dùng)
Khai báo đầy đủ các biến môi trường MongoDB trong `.run.local.env` nhưng tất cả đều có giá trị trỏ về máy local (`localhost:27017`) và giữ đúng database name của từng alias để tránh ghi đè dữ liệu chéo:

```properties
MONGO_URI=mongodb://localhost:27017/centrallized-export-service
MONGO_POOLSIZE=1

PAYMENT_MONGO_URI=mongodb://localhost:27017/payment-service
PAYMENT_MONGO_POOLSIZE=1

PAYMENT_BILL_MONGO_URI=mongodb://localhost:27017/payment-bill-service
PAYMENT_BILL_MONGO_POOLSIZE=1

TRANS_HIS_MONGO_URI=mongodb://localhost:27017/core-trans-proxy-history-service
TRANS_HIS_MONGO_POOLSIZE=1

NOTIFICATION_MONGO_URI=mongodb://localhost:27017/notification-service
NOTIFICATION_MONGO_POOLSIZE=1

CUSTOMER_MONGO_URI=mongodb://localhost:27017/customer-service
CUSTOMER_MONGO_POOLSIZE=1

PROFILE_MONGO_URI=mongodb://localhost:27017/profile-service
PROFILE_MONGO_POOLSIZE=1

PROMOTION_MONGO_URI=mongodb://localhost:27017/promotion-service
PROMOTION_MONGO_POOLSIZE=1

BOOKING_TICKET_MONGO_URI=mongodb://localhost:27017/booking-ticket-service
BOOKING_TICKET_MONGO_POOLSIZE=1

DISBURSEMENT_MONGO_URI=mongodb://localhost:27017/disbursement-service
DISBURSEMENT_MONGO_POOLSIZE=1

CONFIG_MONGO_URI=mongodb://localhost:27017/config-service
CONFIG_MONGO_POOLSIZE=1

TICKET_MONGO_URI=mongodb://localhost:27017/ticket-service
TICKET_MONGO_POOLSIZE=1

MERCHANT_MONGO_URI=mongodb://localhost:27017/merchant-service
MERCHANT_MONGO_POOLSIZE=1

CORE_TRANS_PROXY_MONGO_URI=mongodb://localhost:27017/core-trans-proxy-service
CORE_TRANS_PROXY_MONGO_POOLSIZE=1
```

---

### Giải pháp 2: Điều chỉnh code để tự động phân giải Database Name từ `MONGO_URI` (Sửa code)
Nếu anh muốn file `.run.local.env` thật ngắn gọn, chỉ cần khai báo 1 dòng `MONGO_URI` duy nhất, chúng ta có thể sửa đổi code để tự động phân giải phần Host của `MONGO_URI` và ghép thêm database name mặc định của từng alias tương ứng.

#### Chi tiết thay đổi:
Trong file [svc-env.ts](file:///Users/trainguyen/Documents/work/centralized-export-service/svc-env.ts), tại cuối hàm `SVC_ENV.setEnvironments` (dòng 190):

```typescript
		// Tự động phân tách và gán các Mongo URI khác theo MONGO_URI
		const defaultMongoUri = SVC_ENV.get().MONGO_URI;
		if (defaultMongoUri) {
			const MONGO_DB_MAPPING: Record<string, string> = {
				PAYMENT_BILL_MONGO_URI: "payment-bill-service",
				PAYMENT_MONGO_URI: "payment-service",
				TRANS_HIS_MONGO_URI: "core-trans-proxy-history-service",
				NOTIFICATION_MONGO_URI: "notification-service",
				CUSTOMER_MONGO_URI: "customer-service",
				PROFILE_MONGO_URI: "profile-service",
				PROMOTION_MONGO_URI: "promotion-service",
				BOOKING_TICKET_MONGO_URI: "booking-ticket-service",
				DISBURSEMENT_MONGO_URI: "disbursement-service",
				CONFIG_MONGO_URI: "config-service",
				MERCHANT_MONGO_URI: "merchant-service",
				TICKET_MONGO_URI: "ticket-service",
				CORE_TRANS_PROXY_MONGO_URI: "core-trans-proxy-service"
			};
			Object.keys(MONGO_DB_MAPPING).forEach(key => {
				if (!SVC_ENV.get()[key]) {
					try {
						const connectionString = defaultMongoUri.startsWith("mongodb://") || defaultMongoUri.startsWith("mongodb+srv://")
							? defaultMongoUri
							: `mongodb://${defaultMongoUri}`;
						
						// Sử dụng URL class để parse và thay thế pathname một cách an toàn
						const parsedUrl = new URL(connectionString);
						parsedUrl.pathname = `/${MONGO_DB_MAPPING[key]}`;
						SVC_ENV.set(key, parsedUrl.toString());
					} catch (e) {
						// Fallback thô nếu URL không hợp lệ
						SVC_ENV.set(key, defaultMongoUri);
					}
				}
			});
		}
```
