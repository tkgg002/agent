# Plan: TriggeredBy Refactor

## English
1. Inspect current worker flow for activity logging, command handling, scheduler paths, and Kafka consumer batch handling.
2. Introduce a small core activity package for TriggeredBy constants, operation names, and optional debug/action helpers.
3. Wire existing scheduler/NATS/Kafka call sites to the centralized constants with minimal behavior change.
4. Add Kafka post-consume action support with a no-op default, explicit debug logs, and tests.
5. Run Go tests/build checks that cover changed packages.
6. Run security/review scans and write the final repo report.

## Tiếng Việt
1. Đọc luồng worker hiện tại: activity log, command handler, scheduler, và Kafka batch handling.
2. Tạo một package core nhỏ để quản lý TriggeredBy constants, operation names, và helper debug/action nếu cần.
3. Thay các call site scheduler/NATS/Kafka sang constants tập trung, hạn chế đổi behavior.
4. Thêm Kafka post-consume action với default no-op, log debug rõ, và test.
5. Chạy Go test/build cho các package bị ảnh hưởng.
6. Chạy security/review scan và viết report cuối trong repo.

