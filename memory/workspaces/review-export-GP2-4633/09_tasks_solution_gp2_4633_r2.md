# 09. Tasks Solution — `gp2_4633_r2` (Propose-Only, chờ APPROVE; sửa trong repo `centralized-export-service`)

| Finding | Đề xuất sửa | Code demo |
| :--- | :--- | :--- |
| F1 (S1, chặn merge) | Bỏ ngoặc quanh tham số | `.filter(s => s === STATUS_SUCCESS)` |
| D1 | Dùng helper canonical | `return [...transHisModelNames(), "MerchantModel"];` (hoặc `getRequiredModelName` giữ Mongo nếu quyết định D2 là cố ý) |
| D3 | Thêm `select` vào query, truyền `SELECT_FIELDS` từ pure | query `{filter, sort, select}` như `GetAllTransHisExportQuery` |
| D5 | Kế thừa base | `class PartnerUserReserveExportParams extends TransHisExportParams` |
| D4 | Dùng lại validator dùng chung (nếu có), không thêm bản copy | import thay vì khai báo lại |
| F5 | Chặn range âm | `dateDiff >= 0 && dateDiff <= MAX_DAYS_EXPORT` |
| F4 | trim status | `String(params.status).split(",").map(s => s.trim())` |
| F6 | Đồng nhất múi giờ | `moment().utcOffset(7).format("DD-MM-YYYY")` |
| F2/F7/D2 | Xác nhận với team (tie-breaker `_id`, kiểu số tiền, cấu hình nguồn); không sửa trong PR này nếu là nợ chung | thêm `_id` vào sort khi đồng ý |

Chưa sửa gì trong repo. Chờ `APPROVE`.
