# Kế hoạch Refactor chi tiết: Luồng eKYC

## 1. Mục tiêu

- **Ổn định hệ thống:** Đảm bảo không mất request eKYC khi `vmg-ekyc-connector-service` restart.
- **Async Processing:** Xử lý bất đồng bộ vì eKYC có thể mất vài giây.
- **Retry Failed:** Tự động retry các request thất bại do lỗi tạm thời.

---

## 2. Các Service liên quan

- `vmg-ekyc-connector-service` (Gọi API VMG để xác thực)
- `user-service` (Cập nhật trạng thái KYC của user)
- `profile-service` (Lưu thông tin định danh)

---

## 3. Phân tích luồng hiện tại

```
User upload CMND/CCCD -> App
         |
         v
vmg-ekyc-connector-service: Gọi API VMG OCR + Face Match
         |
         v
         +--> Thành công: Trả về thông tin đã xác thực
         |
         +--> Thất bại: Trả về lỗi
         |
         v
user-service: Cập nhật trạng thái KYC
profile-service: Lưu thông tin
```

**Điểm yếu:**
- VMG API có thể chậm (5-10s) hoặc timeout.
- Nếu service restart khi đang gọi VMG -> Request bị mất.
- User phải chờ đợi lâu (UX tệ).

---

## 4. Kế hoạch thực thi chi tiết

### **Giai đoạn 1: Ổn định Hạ tầng**

#### 1.1. Graceful Shutdown

**File:** `moleculer.config.ts`

```typescript
tracking: {
    enabled: true,
    shutdownTimeout: 45000, // VMG API chậm
},
```

### **Giai đoạn 2: Async Processing**

#### 2.1. Schema `EkycRequest` (MongoDB)

```typescript
{
    requestId: string;          // Idempotency Key
    userId: string;
    imageUrls: {
        front: string;
        back: string;
        selfie: string;
    };
    status: 'PENDING' | 'PROCESSING' | 'SUCCESS' | 'FAILED' | 'NEED_REVIEW';
    vmgResponse: object;
    extractedData: {
        fullName: string;
        idNumber: string;
        dob: Date;
        address: string;
        // ...
    };
    retryCount: number;
    error: string;
    createdAt: Date;
    updatedAt: Date;
    processedAt: Date;
}
```

**Index:**
- `UNIQUE INDEX` trên `requestId`
- `INDEX` trên `userId`
- `INDEX` trên `status` + `updatedAt`

#### 2.2. Producer (Gateway)

**File:** Mobile/Web Gateway

```typescript
async initiateEkyc(payload: EkycPayload) {
    // 1. Upload images to storage
    const imageUrls = await this.uploadImages(payload.images);

    // 2. Tạo request record
    const request = await EkycRequestModel.create({
        requestId: payload.requestId || generateId(),
        userId: payload.userId,
        imageUrls,
        status: 'PENDING',
    });

    // 3. Emit event để xử lý async
    await this.broker.emit('ekyc.requested', {
        requestId: request.requestId,
        userId: request.userId,
        imageUrls,
    });

    // 4. Trả về ngay cho user
    return {
        requestId: request.requestId,
        status: 'PROCESSING',
        message: 'Đang xác thực, vui lòng chờ...',
    };
}
```

#### 2.3. Consumer (vmg-ekyc-connector-service)

```typescript
events: {
    "ekyc.requested": {
        group: "ekyc-connector",
        async handler(ctx: Context<EkycRequestedPayload>) {
            const { requestId, userId, imageUrls } = ctx.params;

            // Update status = PROCESSING
            await EkycRequestModel.updateOne(
                { requestId },
                { $set: { status: 'PROCESSING' } }
            );

            try {
                // Gọi VMG API
                const vmgResult = await this.callVmgApi({
                    frontImage: imageUrls.front,
                    backImage: imageUrls.back,
                    selfieImage: imageUrls.selfie,
                });

                // Lưu kết quả
                await EkycRequestModel.updateOne(
                    { requestId },
                    {
                        $set: {
                            status: 'SUCCESS',
                            vmgResponse: vmgResult,
                            extractedData: this.parseVmgResponse(vmgResult),
                            processedAt: new Date(),
                        },
                    }
                );

                // Cập nhật user profile
                await this.broker.call('user-service.updateKycStatus', {
                    userId,
                    status: 'VERIFIED',
                });

                await this.broker.call('profile-service.updateIdentity', {
                    userId,
                    identityData: this.parseVmgResponse(vmgResult),
                });

                // Notify user
                await this.broker.emit('notification.send', {
                    userId,
                    type: 'EKYC_SUCCESS',
                });

            } catch (error) {
                const request = await EkycRequestModel.findOne({ requestId });
                request.retryCount = (request.retryCount || 0) + 1;

                if (request.retryCount < 3) {
                    request.status = 'PENDING'; // Để sweeper retry
                } else {
                    request.status = 'FAILED';
                    // Notify user để thử lại thủ công
                    await this.broker.emit('notification.send', {
                        userId,
                        type: 'EKYC_FAILED',
                        data: { requestId },
                    });
                }

                request.error = error.message;
                await request.save();
            }
        }
    }
}
```

### **Giai đoạn 3: Job Sweeper**

```typescript
async sweepFailedEkyc() {
    const pending = await EkycRequestModel.find({
        status: 'PENDING',
        retryCount: { $lt: 3 },
        updatedAt: { $lt: new Date(Date.now() - 5 * 60 * 1000) }, // > 5 phút
    });

    for (const request of pending) {
        await this.broker.emit('ekyc.requested', {
            requestId: request.requestId,
            userId: request.userId,
            imageUrls: request.imageUrls,
        });
    }
}
```

### **Giai đoạn 4: UX Improvements**

#### 4.1. Polling API

```typescript
// GET /ekyc/:requestId/status
async getEkycStatus(requestId: string) {
    const request = await EkycRequestModel.findOne({ requestId });
    if (!request) {
        throw new Error('Request not found');
    }

    return {
        status: request.status,
        message: this.getStatusMessage(request.status),
        extractedData: request.status === 'SUCCESS' ? request.extractedData : null,
    };
}
```

#### 4.2. WebSocket Notification

Khi eKYC hoàn thành, bắn event qua Socket.io để App cập nhật ngay.

---

## 5. Checklist Idempotency

| Bảng/Collection | Cột Unique | Action |
|---|---|---|
| `ekyc_requests` | `requestId` | Tạo UNIQUE INDEX |
| `profiles` | `userId` | Đảm bảo có INDEX |

---

## 6. Rủi ro và Giải pháp

| Rủi ro | Giải pháp |
|---|---|
| VMG API timeout | Async processing, Sweeper retry |
| Service restart khi đang xử lý | Status = PROCESSING -> Sweeper sẽ re-emit |
| Duplicate request | Idempotency Key |
| User chờ lâu | Async + Notification |

---

## 7. Thứ tự triển khai

1. Tạo Schema `EkycRequest` và Index
2. Cấu hình Graceful Shutdown (45s)
3. Implement Producer (Gateway)
4. Implement Consumer (Connector)
5. Triển khai Sweeper
6. Thêm Polling API và WebSocket notification
7. Test toàn diện
