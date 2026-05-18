# Context: Flow 1 Final Release & 404 Resolution
## Mục tiêu: Hoàn thiện Flow 1 và fix lỗi 404 lỳ lợm

### Current Issues:
- GET /api/v1/introspection/mongo/databases returns 404.
- User needs to add multiple MongoDB instances (Multi-instance support).
- Service state in FE is not synced with actual BE capability.

### Constraints:
- Must verify with real CURL/Browser before reporting.
- Must follow Lesson [2026-04-17]: Restart + Verify.
