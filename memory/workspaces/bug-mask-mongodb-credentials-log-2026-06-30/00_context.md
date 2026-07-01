# Context: Bug Mask MongoDB Credentials in Connection Log

## Bối cảnh
User phát hiện ra log kết nối MongoDB hiển thị URL chứa clear-text credentials dạng:
`{"level":"info","ts":1782753770.2089758,"msg":"MongoDB connected","url":"mongodb://readonly-user:EfG567HiJk890LmNoPqRsTuVwXyZ12@10.200.186.11:27017,10.200.186.12:27017,10.200.186.13:27017/admin?replicaSet=goopay&authSource=admin&serverSelectionTimeoutMS=5000&connectTimeoutMS=5000"}`

Điều này gây rò rỉ thông tin nhạy cảm (username, password) ra log hệ thống.

## Mục tiêu (DoD)
- Rà soát toàn bộ codebase để tìm các chỗ log URL kết nối MongoDB.
- Mask hoặc loại bỏ phần credentials (username:password) khỏi URL trước khi ghi log.
- URL ghi log sau khi mask sẽ có dạng: `mongodb://****:****@10.200.186.11:27017...` hoặc loại bỏ hoàn toàn phần credentials, chỉ giữ lại host/port hoặc dùng thư viện parse URL chuẩn để ẩn thông tin nhạy cảm.
- Kiểm tra lại bằng cách viết test hoặc rà soát để đảm bảo không làm gãy kết nối thật, chỉ thay đổi phần log hiển thị.
