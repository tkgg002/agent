# 01 Requirements: Audit & Xử lý Lỗi Confluence 404 (`No space with key : ~712020a3dcebcbe93646fea464f119dd38b08c`) khi dùng `sre@goopay.vn`

tier: T2
suffix: confluence_space_404_2026_10_29
date: 2026-10-29

## 1. Yêu cầu của User
- Kiểm tra nguyên nhân lỗi khi dùng tài khoản `sre@goopay.vn` và API Token của nó để push báo cáo Confluence Daily Report:
  ```text
  check existing confluence page failed: search content returned status 404: {"statusCode":404,"data":{"authorized":true,"valid":true,"errors":[],"successful":true},"message":"com.atlassian.confluence.api.service.exceptions.api.NotFoundException: No space with key : ~712020a3dcebcbe93646fea464f119dd38b08c"}
  ```
- Làm rõ:
  1. Tại sao `sre@goopay.vn` + token của nó đã xác thực thành công (`"authorized":true,"valid":true`) nhưng lại bị lỗi `404 NotFoundException: No space with key : ~712020a3dcebcbe93646fea464f119dd38b08c`?
  2. Cách khắc phục trên Atlassian Confluence Cloud (cấp quyền Space / Group nào, dùng tài khoản nào có quyền `administer:space`) và cải tiến trong `centralized-data-service/pkgs/confluence/client.go` để tự động đồng bộ `SpaceKey` từ `ParentPageID` + chẩn đoán rõ lỗi phân quyền Space `404 (authorized: true)`.

## 2. Ràng buộc Thực thi từ Bài học (Technical Constraints — S0 Boot)
- **`L003` (`Propose-Only Discipline & Explicit Approval Gate`):** Giữ chế độ **100% Read-Only (Propose-Only)** trên toàn bộ source code `data-hub` ở bước `S1–S3`. Trình bày rõ phân tích gốc rễ + hướng xử lý + **Code Demo** trực tiếp trên chat và dừng chờ lệnh `APPROVE` tường minh trước khi sửa code.
- **`L009` (`Pre-Plan Adversarial Critique Wave DG0`):** Kiểm chứng trực tiếp trên Atlassian Confluence Cloud API (`GET /wiki/rest/api/space/~712020a3dcebcbe93646fea464f119dd38b08c?expand=permissions`, `GET /wiki/rest/api/user/memberof`, `GET /wiki/rest/api/content/500432919?expand=space`) và chạy `laya_gate.py run --stage dg0` trước khi lập `09_tasks_solution`.
- **`L020` (`Direct-Answer Summary First & 1-to-1 Finding Mapping`):** Mở đầu phản hồi bằng khối **Direct Summary** (2–4 câu) trả lời trực diện vì sao `sre@goopay.vn` bị 404 và cách xử lý ngay; ánh xạ 1-1 từng phát hiện `F1..Fn` sang giải pháp.
- **`L027` (`End-to-End Secret Masking`):** Tuyệt đối không ghi thô bất kỳ API Token hay mật khẩu nào vào file workspace memory hoặc output.
- **`L043` (`Topic Sanitization, Confluence Ancestors & Reader-Centric Reports`):** Bảo toàn 100% việc truyền `"ancestors": [{"id": parentPageID}]` trong cả `createPage` (`POST`) và `updatePage` (`PUT`) của `pkgs/confluence/client.go`.
