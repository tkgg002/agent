# Báo cáo Phân tích Dependency GooPay (Nâng cao)

## 1. Kết luận: GooPay ĐANG bị "Spaghetti Dependency"
Dựa trên kết quả rà soát tài liệu hệ thống (`global-goopay/flow`) và scan trực tiếp một số core services (`payment-service`, `wallet-service`), hệ thống GooPay hiện tại đang gặp tình trạng **Synchronous RPC Coupling** (phụ thuộc đồng bộ qua RPC).

## 2. Các bằng chứng kỹ thuật (Evidence)

### 2.1. Synchronous RPC Chain (Chuỗi gọi đồng bộ)
Hệ thống sử dụng Moleculer với cơ chế `ctx.call` (RPC qua NATS) một cách lạm dụng. Các flow quan trọng như **Payment Bill** đi qua một chuỗi dài:
`payment-gateway` -> `payment-bill-service` -> `payment-service` -> `core-trans-proxy-service` -> `bank-connector`.
- **Rủi ro**: Nếu bất kỳ service nào ở cuối chuỗi (ví dụ: bank-connector) bị chậm hoặc restart, toàn bộ chuỗi phía trước sẽ bị treo (blocked).
- **Hiện trạng**: Theo tài liệu `refactor.md`, lỗi 542/502 xảy ra thường xuyên khi deploy do thiếu Graceful Shutdown trong chuỗi đồng bộ này.

### 2.2. Fat Services (Service "béo")
- **`payment-service`**: Đóng vai trò là "Orchestrator" nhưng lại chứa quá nhiều logic điều phối (Wallet + Bank Transfer + Proxy). 
- **`wallet-service`**: Là điểm hội tụ của gần như tất cả các flow tài chính. Nếu `wallet-service` xuống, toàn bộ hệ thống đứng yên (High Blast Radius).

### 2.3. Thiếu Decoupling (Bất động bộ)
- Mặc dù dùng NATS làm Transport, nhưng phần lớn giao tiếp vẫn là Request-Response (đồng bộ) thay vì Event-Driven (bất động bộ).
- Saga Pattern hiện chỉ mới được chuẩn hóa ở `wallet-trans-service`, các service khác vẫn đang gọi trực tiếp và thiếu cơ chế rollback/compensate tự động (đang được lên kế hoạch refactor tại GĐ3).

## 3. Phân tích các Giai đoạn Refactor vs. Saga Coordinator

Hệ thống đang chịu sự phụ thuộc đồng bộ (Synchronous RPC Coupling) nặng nề. Việc "bẻ gãy" tình trạng này đòi hỏi một sự thay đổi về kiến trúc điều phối chứ không chỉ là thay đổi kỹ thuật gọi hàm.

Dựa trên tài liệu `@Coordinator/saga-coordinator-design.md`, chúng ta có cái nhìn sâu hơn về 3 giai đoạn đã đề xuất:

### GĐ1: Graceful Shutdown & Retry (Tầng Hạ tầng - Cơ bản)
- **Bản chất**: Chỉ là "giảm đau". Nó giúp các service đơn lẻ không làm mất request khi tắt/mở. 
- **Hạn chế**: Nó không giải quyết được việc Service A vẫn phải "biết" và "chờ" Service B. Nếu mạng lag 60s, cả chuỗi vẫn treo.

### GĐ2: Transaction Sweeper (Tầng Dọn dẹp - Safety Net)
- **Bản chất**: Là "lưới an toàn". Nó quét các giao dịch bị rò rỉ (leak) do lỗi infra cực nặng mà Retry không cứu được.
- **Hạn chế**: Đây là cơ chế bị động (reactive), không giúp flow thanh thoát hơn.

### GĐ3: ctx.call sang ctx.emit (Tầng Kiến trúc - Cốt lõi)
Đây là nơi **Saga Coordinator** xuất hiện. Có hai cách thực hiện GĐ3:
1. **Choreography (Spaghetti bất đồng bộ)**: A bắn event, B nghe rồi bắn tiếp, C nghe... -> Không ai nắm giữ trạng thái tổng thể. Khó debug, khó rollback.
2. **Orchestration (Saga Coordinator)**: Đây chính là giải pháp "Đường ray" mà tài liệu nhắc đến.
   - **Coordinator là "Bộ não"**: Nó giữ State Machine. Nó biết bước 1 xong thì đến bước 2.
   - **Decoupling thực sự**: Service A không cần biết Service B là ai. Nó chỉ làm việc của nó (ví dụ: trừ tiền) và báo cáo lại cho Coordinator qua NATS JetStream.
   - **Compensating (Rollback)**: Nếu bước 2 lỗi, Coordinator tự biết gọi lại Service A để hoàn tiền dựa trên lịch sử trong DB.

## 4. Tại sao "Coordinator" là giải pháp tối hậu?
- **Phá vỡ Spaghetti**: Thay vì chuỗi "A -> B -> C", chúng ta có "A <=> Coordinator <=> B". 
- **Durable State**: Trạng thái giao dịch được lưu xuống DB trước khi thực hiện. Server sập xong dậy vẫn biết đang ở bước nào để chạy tiếp.
- **Config-Driven**: Thay đổi flow nghiệp vụ bằng YAML, không cần sửa code logic của từng service.

## 5. Tổng kết
Ý kiến của bạn rất chính xác: GĐ1 và GĐ2 chỉ là các bước tiền đề để hệ thống "sống sót" qua lỗi. **GĐ3 với Saga Coordinator mới là "cuộc cách mạng" thực sự** để bẻ gãy Spaghetti Dependency và đưa GooPay lên kiến trúc Async chuẩn 2026.
