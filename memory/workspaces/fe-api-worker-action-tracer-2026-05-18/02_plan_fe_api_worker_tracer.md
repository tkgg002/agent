# Plan: FE API Worker Action Tracer

## English
1. Record current git status for FE, API, and worker to avoid overwriting user changes.
2. Locate the FE buttons and exact HTTP requests for `Sync Fields to Shadow` and `Snapshot Now`.
3. Locate the API endpoints receiving those requests and verify whether they publish NATS commands or only update API-local state.
4. Locate worker subscribers and handlers for the expected commands.
5. Fix the smallest broken link in each flow.
6. Add action tracer metadata/logging across FE/API/worker payloads.
7. Validate with tests/build and runtime/service checks.
8. Write report with changed files and actual verification results.

## Tiếng Việt
1. Ghi nhận git status hiện tại của FE, API, worker để không đè thay đổi sẵn có.
2. Tìm button FE và request HTTP thật cho `Sync Fields to Shadow`, `Snapshot Now`.
3. Tìm API endpoint nhận request và verify có publish NATS command không hay chỉ update state nội bộ.
4. Tìm worker subscribe/handler command tương ứng.
5. Fix link bị đứt nhỏ nhất cho từng luồng.
6. Thêm tracer metadata/logging xuyên FE/API/worker payload.
7. Validate bằng test/build và service check.
8. Ghi report nêu file thay đổi và kết quả verify thật.

