# Architectural Decisions - feature-id-expired-notification-log-export

## 1. Tách registry của Processors để phá Circular Dependency
- **Bối cảnh**: Lỗi 500 "Unknown export type" xảy ra do `logics/index.ts` import `ExportLogic` và ngược lại `ExportLogic` import `logics/index.ts`. Điều này khiến danh sách processors bị rỗng khi module đang load.
- **Quyết định**: Tạo file `logics/processors.ts` tách biệt chỉ để chứa các export của Processor.
- **Hệ quả**: `ExportLogic` luôn load được đầy đủ class, sửa triệt để lỗi logic động.

## 2. Mocking validation trong unit test
- **Bối cảnh**: `class-validator` yêu cầu metadata decorator phức tạp trong môi trường test.
- **Quyết định**: Thực hiện mock `validateParams` trong unit test để tập trung vào logic transform dữ liệu (core logic).
