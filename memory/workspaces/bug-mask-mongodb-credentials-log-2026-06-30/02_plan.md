# Implementation Plan - Mask MongoDB Credentials in Connection Log

## Bối cảnh và Mục tiêu
Rà soát và sửa đổi log kết nối MongoDB để tránh hiển thị thông tin xác thực (username:password) dạng text thô ra log hệ thống, phòng ngừa rò rỉ bảo mật.

Log hiện tại đang hiển thị:
`{"level":"info","ts":1782753770.2089758,"msg":"MongoDB connected","url":"mongodb://readonly-user:EfG567HiJk890LmNoPqRsTuVwXyZ12@10.200.186.11:27017,..."}`

Mục tiêu sau khi sửa đổi:
`{"level":"info","ts":1782753770.2089758,"msg":"MongoDB connected","url":"mongodb://****:****@10.200.186.11:27017,..."}`

---

## Các thay đổi đề xuất

### Component: Centralized Data Service (data-hub/centralized-data-service)

#### [MODIFY] [client.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/mongodb/client.go)
- Khai báo một hàm `maskMongoURI(uri string) string` sử dụng Regular Expression để tìm và che giấu phần credentials trong connection string.
- Cập nhật hàm `NewClient` để sử dụng `maskMongoURI(cfg.URL)` khi ghi log thông tin kết nối thành công:
  ```go
  logger.Info("MongoDB connected", zap.String("url", maskMongoURI(cfg.URL)))
  ```

---

## Chi tiết mã nguồn sửa đổi dự kiến (Draft Diff)

```go
package mongodb

import (
	"context"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

type MongoConfig struct {
	URL string `mapstructure:"url"`
}

// maskMongoURI che giấu phần username và password trong chuỗi kết nối MongoDB
func maskMongoURI(uri string) string {
	re := regexp.MustCompile(`^(mongodb(?:\+srv)?://)[^:]+:([^@]+)@`)
	return re.ReplaceAllString(uri, "${1}****:****@")
}

func NewClient(ctx context.Context, cfg MongoConfig, logger *zap.Logger) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.URL))
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	logger.Info("MongoDB connected", zap.String("url", maskMongoURI(cfg.URL)))
	return client, nil
}
```

---

## Kế hoạch kiểm thử & Xác minh (Verification Plan)

### Kiểm thử Đơn vị (Unit Test)
- Viết thêm unit test trong `client_test.go` (hoặc test trực tiếp hàm `maskMongoURI` bằng cách chuyển nó thành một phần của bộ test) để xác minh các trường hợp URL kết nối MongoDB khác nhau (bao gồm cả `mongodb://` thông thường và `mongodb+srv://`, có hoặc không có replicaSet, v.v.).

Ví dụ các test cases:
- Input: `mongodb://readonly-user:EfG567HiJk890LmNoPqRsTuVwXyZ12@10.200.186.11:27017,10.200.186.12:27017/admin?replicaSet=goopay`
  Output kỳ vọng: `mongodb://****:****@10.200.186.11:27017,10.200.186.12:27017/admin?replicaSet=goopay`
- Input: `mongodb+srv://root:MySuperSecurePassword@cluster0.mongodb.net/test`
  Output kỳ vọng: `mongodb+srv://****:****@cluster0.mongodb.net/test`
- Input: `mongodb://localhost:27017/test` (không có auth)
  Output kỳ vọng: `mongodb://localhost:27017/test` (giữ nguyên, không crash)

### Kiểm thử Tích hợp (Integration / Local Build)
- Chạy lệnh `go build` hoặc `go test` tại thư mục `/Users/trainguyen/Documents/work/data-hub/centralized-data-service` để đảm bảo code biên dịch thành công và không gây lỗi cú pháp.
