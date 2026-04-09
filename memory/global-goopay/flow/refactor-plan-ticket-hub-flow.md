# Kế hoạch Refactor chi tiết: Luồng Hỗ trợ Khách hàng (Ticket Hub Flow - CQRS)

## 1. Mục tiêu

- **Ổn định CQRS:** Đảm bảo `ticket-service` với CQRS pattern không mất event khi restart.
- **Event Sourcing:** Xem xét triển khai Event Sourcing để có audit trail hoàn chỉnh.
- **Eventual Consistency:** Xử lý vấn đề đọc-ghi không đồng bộ.

---

## 2. Các Service liên quan

- `ticket-service` (CQRS Architecture)

---

## 3. Phân tích kiến trúc CQRS hiện tại

```
Command Side:
    User/Admin -> ticket-service: CreateTicket, UpdateTicket, CloseTicket
         |
         v
    Command Handler -> Write to Main DB
         |
         v
    Emit Event: ticket.created, ticket.updated, ticket.closed

Query Side:
    User/Admin -> ticket-service: GetTicket, ListTickets
         |
         v
    Query Handler -> Read from Read Model (có thể là view/cache)
```

**Điểm yếu:**
- Nếu service restart sau khi write nhưng trước khi emit event -> Read Model không được cập nhật.
- Event loss -> Data inconsistency giữa Write và Read model.

---

## 4. Kế hoạch thực thi chi tiết

### **Giai đoạn 1: Ổn định Hạ tầng**

#### 1.1. Graceful Shutdown

**File:** `moleculer.config.ts`

```typescript
tracking: {
    enabled: true,
    shutdownTimeout: 30000,
},
```

### **Giai đoạn 2: Transactional Outbox Pattern**

**Mục đích:** Đảm bảo Event được emit ngay cả khi service crash sau khi write.

#### 2.1. Schema `EventOutbox` (MongoDB)

```typescript
{
    eventId: string;
    aggregateId: string;        // ticketId
    aggregateType: 'TICKET';
    eventType: 'CREATED' | 'UPDATED' | 'CLOSED' | 'ASSIGNED' | 'COMMENTED';
    payload: object;
    status: 'PENDING' | 'PUBLISHED' | 'FAILED';
    createdAt: Date;
    publishedAt: Date;
}
```

**Index:**
- `INDEX` trên `status` + `createdAt`
- `INDEX` trên `aggregateId`

#### 2.2. Command Handler với Outbox

**File:** `ticket-service/handlers/create-ticket.handler.ts`

```typescript
async handle(command: CreateTicketCommand) {
    const session = await mongoose.startSession();
    session.startTransaction();

    try {
        // 1. Create ticket
        const ticket = await TicketModel.create([{
            ticketId: generateId(),
            title: command.title,
            description: command.description,
            status: 'OPEN',
            createdBy: command.userId,
        }], { session });

        // 2. Write event to Outbox (trong cùng transaction)
        await EventOutboxModel.create([{
            eventId: generateId(),
            aggregateId: ticket[0].ticketId,
            aggregateType: 'TICKET',
            eventType: 'CREATED',
            payload: {
                ticketId: ticket[0].ticketId,
                title: ticket[0].title,
                createdBy: ticket[0].createdBy,
            },
            status: 'PENDING',
        }], { session });

        await session.commitTransaction();

        return ticket[0];

    } catch (error) {
        await session.abortTransaction();
        throw error;
    } finally {
        session.endSession();
    }
}
```

#### 2.3. Outbox Publisher (Background Worker)

```typescript
async publishPendingEvents() {
    const pending = await EventOutboxModel.find({ status: 'PENDING' })
        .sort({ createdAt: 1 })
        .limit(100);

    for (const event of pending) {
        try {
            // Emit event qua Moleculer
            await this.broker.emit(`ticket.${event.eventType.toLowerCase()}`, event.payload);

            // Mark as published
            event.status = 'PUBLISHED';
            event.publishedAt = new Date();
            await event.save();

        } catch (error) {
            event.status = 'FAILED';
            await event.save();
            this.logger.error('Failed to publish event', { eventId: event.eventId, error });
        }
    }
}
```

### **Giai đoạn 3: Read Model Projection**

#### 3.1. Event Handler để Update Read Model

```typescript
events: {
    "ticket.created": {
        group: "ticket-read-model",
        async handler(ctx: Context<TicketCreatedPayload>) {
            // Upsert vào Read Model (có thể là một collection/view riêng)
            await TicketReadModel.findOneAndUpdate(
                { ticketId: ctx.params.ticketId },
                {
                    $set: {
                        ticketId: ctx.params.ticketId,
                        title: ctx.params.title,
                        status: 'OPEN',
                        createdBy: ctx.params.createdBy,
                        createdAt: new Date(),
                    },
                },
                { upsert: true }
            );
        }
    },
    "ticket.updated": {
        group: "ticket-read-model",
        async handler(ctx: Context<TicketUpdatedPayload>) {
            await TicketReadModel.findOneAndUpdate(
                { ticketId: ctx.params.ticketId },
                { $set: ctx.params.changes }
            );
        }
    },
    "ticket.closed": {
        group: "ticket-read-model",
        async handler(ctx: Context<TicketClosedPayload>) {
            await TicketReadModel.findOneAndUpdate(
                { ticketId: ctx.params.ticketId },
                { $set: { status: 'CLOSED', closedAt: new Date() } }
            );
        }
    }
}
```

### **Giai đoạn 4: Startup Recovery**

```typescript
async onServiceStarted() {
    // Resume publishing pending events
    await this.publishPendingEvents();

    // Start background worker
    setInterval(() => this.publishPendingEvents(), 5000);
}
```

### **Giai đoạn 5: Event Replay (Recovery Tool)**

Công cụ để rebuild Read Model từ Event Outbox nếu cần:

```typescript
async replayEvents(ticketId?: string) {
    const query = ticketId 
        ? { aggregateId: ticketId, status: 'PUBLISHED' }
        : { status: 'PUBLISHED' };

    const events = await EventOutboxModel.find(query).sort({ createdAt: 1 });

    // Clear read model (nếu replay toàn bộ)
    if (!ticketId) {
        await TicketReadModel.deleteMany({});
    }

    // Replay từng event
    for (const event of events) {
        await this.broker.emit(`ticket.${event.eventType.toLowerCase()}`, event.payload);
    }

    this.logger.info(`Replayed ${events.length} events`);
}
```

---

## 5. Checklist

| Component | Action |
|---|---|
| EventOutbox Schema | Tạo với Index |
| Command Handlers | Refactor để write vào Outbox trong transaction |
| Background Publisher | Implement và chạy định kỳ |
| Event Handlers | Đảm bảo idempotent |
| Replay Tool | Implement cho recovery |

---

## 6. Rủi ro và Giải pháp

| Rủi ro | Giải pháp |
|---|---|
| Event loss khi restart | Transactional Outbox Pattern |
| Read Model out of sync | Event Handler idempotent, Replay Tool |
| Duplicate event publish | Check status trước khi publish |
| Performance degradation | Background worker, batch processing |

---

## 7. Thứ tự triển khai

1. Tạo Schema `EventOutbox`
2. Refactor Command Handlers để dùng Outbox
3. Implement Background Publisher
4. Refactor Event Handlers cho Read Model
5. Implement Replay Tool
6. Cấu hình Graceful Shutdown
7. Test toàn diện với kịch bản restart
