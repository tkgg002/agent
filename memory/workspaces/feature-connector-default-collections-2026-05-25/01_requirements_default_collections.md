# 01_requirements_default_collections — Yêu cầu

> **Phase**: `default_collections`
> **Source of truth**: User message ngày 2026-05-25.

---

## 1. User story

```
Là Ops admin sử dụng CMS,
Tôi muốn khi tạo Mongo Source Connector mà KHÔNG nhập gì vào ô Collections,
Hệ thống PHẢI CDC toàn bộ collections của database đó (default = all),
Đồng thời UI PHẢI cho tôi biết rõ ràng rằng "để trống = đầy đủ".
```

## 2. Functional requirements

| ID | Yêu cầu | Tiêu chí Done |
|---|---|---|
| **R1** | Form field `Collections` (Mongo connector) PHẢI có hint visible giải thích "Để trống = CDC toàn bộ collections" | User nhìn form thấy text giải thích, không cần đọc doc external |
| **R2** | Khi user để trống và submit, connector PHẢI được tạo thành công và CDC ALL collections của DB | Verify bằng smoke test E2E (Section 6 — Test Cases) |
| **R3** | Display sau khi tạo (list / detail view) PHẢI phân biệt được "không filter" vs "đã filter cụ thể" | Show `(All collections)` hoặc tương đương khi `collection.include.list` rỗng/missing |
| **R4** | Behavior cũ (user nhập list cụ thể) PHẢI giữ nguyên 100% — backward compatible | Existing connectors không bị regression |
| **R5** | Placeholder hiện tại (`users,orders,payments`) có thể giữ làm ví dụ, nhưng KHÔNG được trông giống required hint | UX review pass: text giải thích nổi bật hơn placeholder |

## 3. Non-functional requirements

| ID | Yêu cầu |
|---|---|
| **N1** | KHÔNG thay đổi BE handler logic / Debezium config (audit đã xác nhận runtime đúng) |
| **N2** | KHÔNG migrate DB / schema |
| **N3** | Build pass (`pnpm build` hoặc `npm run build` trên `cdc-cms-web`) |
| **N4** | Lint / vet pass |
| **N5** | A11y: hint text phải có ARIA association với input (Antd `Form.Item` `extra` / `tooltip` đã hỗ trợ sẵn) |
| **N6** | i18n: nếu codebase đã có i18n → dùng key; nếu chưa → hardcode tiếng Việt (consistent với CLAUDE.md §0) |

## 4. Out of scope (defer)

| Item | Lý do | Phase đề xuất |
|---|---|---|
| UI "Select collections from picker" (gọi BE list collections của Mongo URI) | Phức tạp hơn, cần BE endpoint mới | future phase `connector-collection-picker` |
| Validate format `db.collection,db.collection` chuẩn Debezium | Đang trust user input | future phase |
| Auto-detect Mongo connector class (filter UI theo connector type) | Đã có select connector kind ở step trước | future phase |
| Sync hint text này sang FE khác (vd portal admin Goopay) | Khác repo, khác scope | future phase |

## 5. Acceptance criteria (Definition of Done)

Phase này được tính DONE khi **TẤT CẢ** các tiêu chí sau PASS:

- [ ] **A1**: UI form connector kiểu Mongo hiển thị hint text giải thích empty Collections behavior. Screenshot evidence trong `report_default_collections_2026-05-25.md`.
- [ ] **A2**: Build FE `cdc-cms-web` PASS không warning mới.
- [ ] **A3**: Smoke test: tạo connector Mongo với empty Collections → verify trên Kafka Connect REST API rằng connector active, status RUNNING, KHÔNG có key `collection.include.list` trong config.
- [ ] **A4**: Smoke test: insert/update 1 document vào collection KHÔNG được user khai báo trước đây → verify CDC event vẫn được capture (CDC all).
- [ ] **A5**: Backward compat: 1 connector cũ với explicit Collections list vẫn hoạt động bình thường sau khi rebuild FE.
- [ ] **A6**: List view hiển thị `(All collections)` cho connector empty list, hiển thị tên cụ thể cho connector có list.
- [ ] **A7**: `/security-agent` review PASS hoặc không HIGH/CRITICAL finding.
- [ ] **A8**: `report_default_collections_2026-05-25.md` filled với evidence thực tế.
- [ ] **A9**: `05_progress.md` APPEND đầy đủ audit log mỗi milestone.
- [ ] **A10**: `agent/memory/global/active_plans.md` cập nhật trạng thái workspace.

## 6. Risk matrix

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Audit hypothesis sai — BE thực ra reject empty config | Low | High | M0 verify trước qua test API trên local stack |
| Debezium version dùng connector khác (Mongo connector Confluent vs Debezium chính thức) có default behavior khác | Low | Medium | M0 đọc Kafka Connect connector class trong config, tra doc version tương ứng |
| Connector cũ đang dùng `database.include.list` rỗng → bị ảnh hưởng khi đổi hint UI | Very low | Low | Hint chỉ là text, không đổi runtime |
| List view component không tồn tại / khó sửa | Low | Low | M1 audit trước; nếu phức tạp → defer R3 sang sub-phase |
| Hint text gây xấu UI (Antd Form layout vertical/horizontal khác nhau) | Medium | Low | Dùng `extra` (chuẩn Antd) thay vì custom div |

## 7. Inverse requirements (CẤM)

| Cấm | Lý do |
|---|---|
| Cấm thêm logic FE auto-inject `collection.include.list: "*"` | "*" không hợp lệ Debezium, gây regression |
| Cấm sửa `compactConfig` để KHÔNG filter empty values | Sẽ break các config khác đang dựa vào behavior này |
| Cấm thêm validation "required" cho Collections | Phá yêu cầu R2 |
| Cấm sửa BE handler / Debezium config / Kafka Connect manifest | Out of scope, audit đã xác nhận đúng |
| Cấm tạo migration DB | Không có thay đổi schema |
