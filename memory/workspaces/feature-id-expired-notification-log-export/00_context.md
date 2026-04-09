# Workspace: feature-id-expired-notification-log-export

## Mục tiêu
Tạo export type mới `IDExpiredNotificationLogExport` trong `centralized-export-service`, lấy logic từ `profile-service/idexpired`.

## Scope
- **Source**: `profile-service/idexpired/applications/query/export-profile-exp.query.ts`
- **Target project**: `centralized-export-service`

## Files bị ảnh hưởng
| File | Action |
|------|--------|
| `data-transfers/entities/profile/id-expired-notification-log.entity.ts` | [NEW] Tạo mới |
| `data-transfers/entities/index.ts` | [MODIFY] Thêm import + export |
| `defined/app-setting.ts` | [MODIFY nếu cần] Thêm DB_COLLECTION key |

## Entity Structure (từ profile-service)
```
IDExpiredNotificationLogExport
  - notificationId: string
  - idExpired: string
  - status: string         // SUCCESS | FAILED
  - title: string
  - content: string
  - sentAt: Date
  - error: string
  - sendType: string       // NOTIFY | SMS | EMAIL | ...
  - createdAt: Date
  - IdExpired: {           // ref populated
      profileId, customerId, phone, fullname,
      type, expiredAt, createdAt, updatedAt
    }
```

## Collection name (từ schema)
`id-expired-notification-log`

## Status
🟡 Active — 2026-02-27
