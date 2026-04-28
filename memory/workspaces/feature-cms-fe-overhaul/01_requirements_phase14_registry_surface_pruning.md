# Requirements — Phase 14 Registry Surface Pruning

## Mục tiêu

- Audit `/api/registry` theo usage thật để xác định:
  - phần nào còn sống cho operator-flow
  - phần nào chỉ là compatibility shell
  - phần nào là API dead nên gỡ luôn

## Yêu cầu

1. Không cắt nhầm capability mà FE/operator-flow đang dùng thật.
2. Nếu gỡ API, phải gỡ cả route lẫn handler dead tương ứng.
3. Nếu API còn sống nhưng semantics đã đổi, phải update swagger/comment theo phase này.
4. Verify lại bằng backend tests và frontend build.
