# Architecture Decisions: TracesHardening

## ADR 1: Graceful Shutdown Pattern
- **Decision**: Thay đổi cơ chế graceful shutdown từ việc gọi `os.Exit(0)` trong goroutine signal receiver sang việc block main thread bằng signal channel, sau đó tuần tự thực hiện shutdown các dependencies và kết thúc hàm `main()` để đảm bảo tất cả các hàm `defer` (đặc biệt là flushing trace provider) được gọi đầy đủ.
- **Rationale**: `os.Exit()` dừng process ngay lập tức và bỏ qua hoàn toàn các lệnh `defer`, dẫn đến việc các spans chưa kịp flush sẽ bị mất vĩnh viễn trong SigNoz.
