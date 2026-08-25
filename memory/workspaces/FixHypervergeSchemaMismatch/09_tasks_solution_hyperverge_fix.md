# Technical Solution Proposal: Remove Hardcoded "_id" -> "id" Overrides

## 1. File Modification Diffs

### Target File 1: `internal/handler/shadow/event_handler.go`

#### Change Location 1: Lines 353-355

```diff
 			if pgPKField == "_id" {
-				pgPKField = "id"
 			}
```
*(Xóa hoàn toàn câu lệnh override `pgPKField = "id"`)*

#### Change Location 2: Lines 384-386

```diff
-		if !mappedPK && pkField == "_id" {
-			pgPKField = "id"
-		}
```
*(Xóa hoàn toàn block ép đè `pgPKField = "id"`)*

---

### Target File 2: `internal/handler/source/bridge_handler.go`

#### Change Location: Lines 281-283

```diff
-		if resolved.pgPKField == "" || resolved.pgPKField == "_id" {
-			resolved.pgPKField = "id"
-		}
+		if resolved.pgPKField == "" {
+			resolved.pgPKField = "_id"
+		}
```
*(Xóa bỏ việc ép `_id` thành `"id"` khi resolve binding)*

---

## 2. Technical Justification
- **Chính xác 100%:** Khi `PrimaryKeyField` của MongoDB collection (như `hyperverge-face-match`) là `_id`, hệ thống giữ nguyên `pgPKField = "_id"`.
- **DML Integrity:** Câu SQL upsert sinh ra sẽ là `INSERT INTO shadow_testhecs.hyperverge_face_match ("_id", ...) VALUES (...) ON CONFLICT ("_id") DO UPDATE ...`, khớp hoàn toàn 100% với schema PostgreSQL đã tạo.
