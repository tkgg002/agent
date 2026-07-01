# Plan: Điều tra lỗi không đồng bộ record thiếu sau khi bắn Debezium Signal

## 1. Thu thập Thông tin & Kiểm tra Logs (Investigation)
- **Hành động 1.1**: Kiểm tra cấu hình connector Debezium của bảng `payment_bills` xem có đúng `signal.data.collection` không (tránh silent fail khi thiếu config này).
- **Hành động 1.2**: Kiểm tra Kafka topics:
    - Xem signal có thực sự đến Kafka và Debezium Connect có đọc được không.
    - Kiểm tra xem Debezium có publish event dữ liệu của ID `41063` lên Kafka data topic tương ứng của `payment_bills` không.
- **Hành động 1.3**: Kiểm tra logs của Debezium Connect container xem có lỗi gì khi xử lý signal này không (ví dụ: NPE, cursor exhausted, hoặc key routing mismatch).
- **Hành động 1.4**: Kiểm tra logs của Sinkworker xem có nhận được message từ Kafka data topic của `payment_bills` với ID `41063` không, và có lỗi gì khi ghi vào Shadow DB không (ví dụ: type mapping, column mismatch, hoặc logic LWW/OCC block).

## 2. Xác định Nguyên nhân Gốc rễ (Root Cause Analysis)
- Dựa trên logs thu thập được từ bước 1, phân tích xem lỗi nằm ở khâu nào:
    - **Khâu 1 (Signal Delivery)**: Debezium nhận được signal nhưng không xử lý (thiếu config watermark collection, wrong signal key, wrong signal format...).
    - **Khâu 2 (Debezium Read & Publish)**: Debezium xử lý signal, query MongoDB nhưng bị lỗi khi đọc record hoặc khi gửi event dữ liệu lên Kafka data topic (schema registry compatibility error, serialization error...).
    - **Khâu 3 (Kafka -> Sinkworker)**: Sinkworker không consume được event hoặc silently drop (route cache stale, filter table inactive...).
    - **Khâu 4 (Sinkworker -> Shadow DB)**: Sinkworker upsert vào Shadow DB bị lỗi (conflict key, type mismatch, hoặc OCC/LWW guard lọc bỏ vì timestamp).

## 3. Khắc phục & Sửa lỗi (Fixing)
- Thực hiện sửa cấu hình, code hoặc dữ liệu tương ứng với lỗi tìm thấy.
- Bảo đảm thay đổi tối giản và elegant.

## 4. Xác minh Kết quả (Verification)
- Trigger lại heal signal và kiểm tra xem record ID `41063` đã xuất hiện ở Shadow DB chưa.
- Kiểm tra logs của các component để đảm bảo luồng chạy sạch sẽ, không có warning/error.
