# Yêu cầu: Ép số lượng tổng source về shadow khi diff == 0 trong recon_smoke.go

## 1. Bối cảnh
Trong file `recon_smoke.go`, kết quả đối soát `SmokeResult` đang chứa cú pháp lỗi:
```go
SourceTotal:  diff==0?&dstActiveClean:&srcEstClean,
SourceActive: diff==0?&dstActiveClean:&srcEstClean,
```
Go không hỗ trợ toán tử ba ngôi (`? :`), dẫn đến lỗi compile.

## 2. Mục tiêu
- Sửa lỗi cú pháp trên bằng cú pháp Go hợp lệ.
- Khi `diff == 0` (đã xác nhận không có drift thực tế nhờ khớp HashWindow hoặc khớp từ đầu), ép số lượng SourceTotal và SourceActive về bằng giá trị của ShadowActive (`dstActiveClean`).
- Nếu `diff != 0`, giữ nguyên giá trị `srcEstClean`.
- Kiểm tra tính đúng đắn và biên dịch thành công của service.
