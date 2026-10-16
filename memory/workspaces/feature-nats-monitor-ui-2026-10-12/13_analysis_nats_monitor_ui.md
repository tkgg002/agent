# Phân tích Kỹ thuật: NATS Monitor UI

## 1. Bản chất dữ liệu từ NATS Monitor Port 18222
- Cổng HTTP monitor của NATS server (8222 được map ra 18222 trên host) cung cấp dữ liệu JSON độc lập với giao thức TCP.
- `GET /varz`: Cung cấp thông số runtime, version, uptime, cpu, mem, in_msgs, out_msgs, in_bytes, out_bytes, connections count.
- `GET /jsz?streams=true&config=true&consumers=true`: Cung cấp cấu trúc phân cấp:
  `account_details[0].stream_detail[]`:
    - `name`: tên stream.
    - `config.subjects`: mảng subjects.
    - `state.messages`: số message.
    - `state.bytes`: dung lượng bytes.
    - `state.first_seq`, `state.last_seq`: sequence tracking.
    - `consumer_detail[]`:
      - `name`: tên consumer.
      - `num_ack_pending`: số tin nhắn chờ ACK.
      - `num_redelivered`: số tin nhắn gửi lại.
      - `num_waiting`: số lượng fetch chờ.
- `GET /connz`: Cung cấp mảng `connections[]` với `cid`, `ip`, `port`, `name`, `subscriptions`, `in_bytes`, `out_bytes`, `uptime`.

## 2. Ưu thế của giải pháp Backend Aggregator
- Thay vì Frontend trực tiếp gọi tới `localhost:18222` (vướng lỗi CORS hoặc không truy cập được khi triển khai môi trường Docker/K8s/Staging), Frontend gọi tới Backend `/api/system/nats/overview`.
- Backend gom 3 request thành 1 response duy nhất, tối ưu latency và giảm số lượng round-trip giữa client và server.
