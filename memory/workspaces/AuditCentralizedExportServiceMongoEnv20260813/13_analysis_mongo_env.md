# Audit Report: MongoDB Connection URL for centralized-export-service (Updated)

## 1. Kết quả Audit & Phản hồi từ User
- **Phân tách Database theo Alias:** Dự án sử dụng nhiều database MongoDB riêng biệt cho từng alias trong `DB_LIST` nhằm đảm bảo tính cô lập dữ liệu nghiệp vụ:
  - Alias `default` kết nối database `centrallized-export-service` (thông qua `MONGO_URI`).
  - Alias `payment-bill` kết nối database `payment-bill-service` (thông qua `PAYMENT_BILL_MONGO_URI`).
  - Alias `payment` kết nối database `payment-service` (thông qua `PAYMENT_MONGO_URI`).
  - v.v.
- **Ràng buộc:** Vì `REQUIRED_KEYS` tại [svc-env.ts](file:///Users/trainguyen/Documents/work/centralized-export-service/svc-env.ts) bắt buộc tất cả các biến URI Mongo phải tồn tại, và hệ thống sẽ crash khi start app nếu thiếu bất kỳ biến nào, chúng ta không thể đơn thuần xoá bỏ các biến URI này khỏi file `.run.local.env`.
- **Mục tiêu của User:** Đưa cấu hình MongoDB về local chạy trên 1 link kết nối duy nhất (cùng Host/Port `mongodb://localhost:27017`) nhưng **phải giữ nguyên phân tách database tương ứng với từng alias** chứ không được gom chung dữ liệu vào 1 database duy nhất.

---

## 2. Các giải pháp đề xuất

### Giải pháp 1: Khai báo thủ công các Mongo URI trỏ về Local (Không sửa code)
Cách này an toàn nhất vì không thay đổi logic code của dự án, giữ nguyên cơ chế hoạt động nghiêm ngặt của `REQUIRED_KEYS`.
Anh cập nhật file `/Users/trainguyen/Documents/work/centralized-export-service/.run.local.env` như sau (giữ đúng database name tương ứng với từng alias):

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

*Đánh giá:* 
- Cách này giúp anh chỉ cần khai báo một dòng `MONGO_URI=mongodb://localhost:27017/centrallized-export-service` trong `.run.local.env`.
- Các biến Mongo URI khác nếu bị thiếu sẽ tự động kế thừa thông tin host/port/credentials/options của `MONGO_URI` nhưng được trỏ tới đúng database tương ứng (ví dụ `/payment-bill-service`).
- Giải pháp này an toàn và thông minh hơn vì sử dụng Node.js `URL` class để phân tách, đảm bảo giữ nguyên 100% options kết nối (như replicaSet, credentials, authSource...) nếu có.
