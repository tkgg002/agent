# Project Context — GooPay

> **Last Updated**: 2026-02-10
> **Maintained by**: /context-manager

## Overview

**GooPay** là hệ thống fintech/payment gateway phục vụ thanh toán, chuyển tiền, nạp/rút, mua vé.
- **~60 microservices**: 55 Node.js/TypeScript (Moleculer + NATS), 4 Go (Fiber/Echo), 6 React frontends.
- **Users**: End-users (mobile app), Merchants (portal), Admin operators (admin portal).

## Domain Knowledge

### Terminologies
- **Disbursement**: Giải ngân - chuyển tiền từ ví GooPay ra tài khoản ngân hàng
- **Reconciliation (Đối soát)**: So khớp dữ liệu giao dịch giữa GooPay và đối tác/ngân hàng
- **Saga**: Pattern điều phối giao dịch phân tán với compensating actions
- **Payment Bill**: Thanh toán hóa đơn (điện, nước, viễn thông...)
- **Connector**: Service kết nối với ngân hàng cụ thể (BIDV, Napas, VietinBank...)

### Business Rules
- Giao dịch tài chính PHẢI có idempotency key (`request_id`)
- Wallet deduct phải có compensating action (credit back) khi flow fail
- Banking connectors cần timeout 60s+ (bank phản hồi chậm)
- Mọi giao dịch stuck >15 phút phải được sweeper tự động phát hiện

## Service Groups

| Group | Services | Đặc điểm |
|-------|----------|----------|
| **Financial Core** | wallet, wallet-trans, payment, payment-bill, bank-transfer, disbursement(Go), reconcile(Go) | Critical, ảnh hưởng tiền |
| **Banking Connectors** | bidv-connector, napas-connector, banvietbank-connector, vnpt-epay-connector, common-connector, bank-handler(Go) | External deps, timeout 60s+ |
| **Gateways** | admin-portal-gw, merchant-portal-gw, payment-gw, mobile-auth-gw, mobile-socketio-gw | Traffic entry points |
| **Business** | merchant, customer, promotion, rule, notification, scheduler, booking-ticket | Business logic |
| **Utilities** | export, centralized-export, config, qr, storage-gw, notification-schedule | Supporting |

## Current State
- **Phase**: Refactoring — GĐ0 (Rà soát & Chuẩn bị) đang triển khai
- **Major Issues**:
  - Graceful shutdown thiếu/không đồng nhất → lỗi 502
  - Synchronous RPC coupling (spaghetti) → không resilient
  - Saga pattern chỉ có wallet-trans-service implement
  - Thiếu distributed tracing, centralized logging
  - Transaction lơ lửng khi pod restart

## Key Dependencies
```
wallet-service restart → TOÀN BỘ flow nạp/rút/thanh toán đứng (High Blast Radius)
payment-service = "Fat Service" → điều phối Wallet + Bank Transfer + Proxy
```
