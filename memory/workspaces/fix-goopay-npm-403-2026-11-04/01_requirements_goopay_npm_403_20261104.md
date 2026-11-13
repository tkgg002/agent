---
tier: T2
suffix: goopay_npm_403_20261104
workspace: fix-goopay-npm-403-2026-11-04
created_at: 2026-11-04T09:38:45+07:00
status: IN_PROGRESS
---

# 01. Requirements — Xử lý lỗi `yarn install` trả về `403` từ `https://npm.goopay.vn/@goopay%2fgoopay-library`

## 1. Yêu cầu của User
- **Vấn đề 1 (Đã tháo gỡ):** Lỗi `yarn install` trả về `403` từ `https://npm.goopay.vn/@goopay%2fgoopay-library`.
- **Vấn đề 2 (Đã tháo gỡ):** Lỗi `FieldCrypto: ENCRYPTION_ACTIVE_VERSION or ENCRYPTION_KEY_V{n} is not configured` khi khởi động `customer-service`.
- **Vấn đề 3 (Đã tháo gỡ):** Lỗi `Service 'auth-service.signin' is not registered` khi gọi `/api/v1/admin-portal/auth/signin`.
- **Vấn đề 4 (Hiện tại — Lỗi `ServiceNotAvailableError: Service 'config-service.getAllSystemConfigsByIds' is not available`):** Khi `auth-service.signin` chạy tới `AuthLogic.getRoutesOfUser` (`/Users/trainguyen/Documents/work/auth-service/logics/auth.logic.js:325:45`), hệ thống báo:
  ```text
  [2026-11-04T03:39:16.215Z] WARN  auth-1-/BROKER         : Service 'config-service.getAllSystemConfigsByIds' is not available.
  ServiceNotAvailableError: Service 'config-service.getAllSystemConfigsByIds' is not available.
      at AuthLogic.getRoutesOfUser (/Users/trainguyen/Documents/work/auth-service/logics/auth.logic.js:325:45)
      at async AuthLogic.generateTokenAndGetUserDetail (/Users/trainguyen/Documents/work/auth-service/logics/auth.logic.js:1944:19)
      at async AuthLogic.signin (/Users/trainguyen/Documents/work/auth-service/logics/auth.logic.js:228:19)
  ```
- Yêu cầu: Xác định nguyên nhân gốc rễ khiến `config-service.getAllSystemConfigsByIds` không khả dụng trên Broker dù tiến trình `config-service` (`PID 52405`) đang chạy, và đề xuất cách khắc phục.

## 2. Ràng buộc Thực thi (Technical Constraints từ Global Lessons)
- **L004 (Single-Blocker Laser Focus):** Bám sát duy nhất lỗi chặn `yarn install` 403 trên `@goopay/goopay-library`, trình bày cô đọng, trực diện.
- **L015 (Propose-Only Discipline & Explicit Approval Gate):** Giữ chế độ Read-Only tuyệt đối đối với source code/config dự án cho tới khi qua `DG0` + `DG1` và nhận lệnh `APPROVE` tường minh từ User; hiển thị trực tiếp **Code Demo** ngay trên khung chat.
- **L021 (Pre-Plan Adversarial Critique Wave DG0):** Phản biện kiểm chứng thực tế trạng thái HTTP của `https://npm.goopay.vn`, `.npmrc` / `.yarnrc`, `yarn cache`, và các bản `@goopay/goopay-library` có sẵn trên máy trước khi lập giải pháp `S3`.
- **L027 (Targeted Lookup & Prior Workspace Reuse):** Tra cứu hồ sơ workspace trước đó (`task-goopay-dependency-analysis`) và các repository `*-service` trên máy để tận dụng tri thức/tài nguyên sẵn có.
- **L032 (Direct-Answer Summary First & 1-to-1 Finding Mapping):** Mở đầu phản hồi bằng khối **Direct Summary** (2–4 câu) trả lời trực diện câu hỏi chính; ánh xạ giải pháp 1-1 theo từng phát hiện `F1..Fn`.
- **Rule #20 (Secret/PII Masking):** Tuyệt đối mask mọi `_authToken`, password, token trong `.npmrc`/log trước khi hiển thị hoặc ghi vào memory.
