# Requirements: FE API Worker Action Tracer

## English
- Trace the actual call chain for `Sync Fields to Shadow` and `Snapshot Now` from FE through API to worker.
- Fix the missing trigger for both actions at the correct boundary.
- Add explicit tracer/correlation logging or payload metadata so operators can see which FE action reached API and worker.
- Keep source-driven behavior: API should publish the proper worker command, worker should handle it through NATS/centralized-data-service paths.
- Add tests or source-level verification around the changed call chain.
- Write a repo-local `report_*.md` after validation.

## Tiếng Việt
- Trace call chain thật cho `Sync Fields to Shadow` và `Snapshot Now` từ FE qua API tới worker.
- Fix đúng boundary khiến 2 action không trigger worker.
- Thêm tracer/correlation log hoặc metadata để biết FE action nào đã tới API và worker.
- Giữ behavior đúng core system: API publish command đúng, worker xử lý qua NATS/centralized-data-service.
- Thêm test hoặc verify source-level cho chain bị sửa.
- Ghi `report_*.md` trong repo sau khi validate.

