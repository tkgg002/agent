# Giải pháp Kỹ thuật (Technical Solutions)

## 1. File: `scripts/migrate-mongo-to-postgres.cjs`
**Hành động**: Tạo mới hoặc Thêm vào cuối file.
**Mục tiêu**: Định nghĩa mảng `CORE_FIELDS` (đóng danh sách), hàm `getBsonType` (phân giải type) và hàm chính `buildCorePayloadSourceFieldTypes`.

```javascript
const CORE_FIELDS = new Set([
    "_id", "transId", "originalTransHisId", "requestId", "transType", "status",
    "sysTrans", "value", "noted", "serviceCode", "channelId", "isDelete",
    "createdAt", "updatedAt", "info", "sender", "receiver", "error"
]);

function getBsonType(value) {
    if (value === null) return "null";
    
    const jsType = typeof value;
    if (jsType === "string") return "string";
    if (jsType === "boolean") return "boolean";
    if (jsType === "number") return "number"; 
    
    if (jsType === "object") {
        if (Array.isArray(value)) return "array";
        if (value instanceof Date) return "date";
        
        // 1. Quét theo BSON Wrapper (Từ MongoDB Node.js Driver)
        if (value._bsontype) {
            const t = value._bsontype.toLowerCase();
            if (t === "objectid") return "objectId";
            if (t === "decimal128") return "decimal128";
            if (t === "minkey") return "minKey";
            if (t === "maxkey") return "maxKey";
            if (t === "int32") return "int32";
            if (t === "long") return "long";
            if (t === "double") return "double";
            if (t === "timestamp") return "timestamp";
            if (t === "binary") return "binary";
            if (t === "regex") return "regex";
            if (t === "symbol") return "symbol";
            if (t === "code") return "code";
            if (t === "dbref") return "dbRef";
            return t; // Fallback
        }
        
        // 2. Quét theo Canonical EJSON (Từ Kafka/Debezium CDC)
        if (value.$oid !== undefined) return "objectId";
        if (value.$numberInt !== undefined) return "int32";
        if (value.$numberLong !== undefined) return "long";
        if (value.$numberDouble !== undefined) return "double";
        if (value.$numberDecimal !== undefined) return "decimal128";
        if (value.$date !== undefined) return "date";
        if (value.$timestamp !== undefined) return "timestamp";
        if (value.$binary !== undefined) return "binary";
        if (value.$regularExpression !== undefined) return "regex";
        if (value.$symbol !== undefined) return "symbol";
        if (value.$code !== undefined) return "code";
        if (value.$dbPointer !== undefined) return "dbRef";
        if (value.$minKey !== undefined) return "minKey";
        if (value.$maxKey !== undefined) return "maxKey";
        if (value.$undefined === true) return "undefined"; // Để hàm gọi ném lỗi
        
        // Các object thông thường (vd: info, sender, receiver)
        return "object";
    }
    
    return jsType;
}

function buildCorePayloadSourceFieldTypes(doc) {
    if (!doc || typeof doc !== "object") return null;
    
    const types = {};
    
    // 1. Cấm lọt field lạ (Strict contract)
    for (const key of Object.keys(doc)) {
        if (!CORE_FIELDS.has(key)) {
            throw new Error(`Quarantine: Unknown field '${key}' is not mapped in the 18 core fields.`);
        }
    }
    
    // 2. Extract Type
    for (const field of CORE_FIELDS) {
        if (!(field in doc)) {
            continue; // Missing => Không đưa vào object output
        }
        
        const val = doc[field];
        
        // Bắt lỗi undefined (JS undefined hoặc EJSON undefined)
        if (val === undefined || (val && typeof val === 'object' && val.$undefined === true)) {
             throw new Error(`Quarantine: BSON undefined is not allowed for field '${field}'`);
        }
        
        types[field] = getBsonType(val);
    }
    
    // 3. Ràng buộc Validation cốt lõi
    if (!types._id || types._id === "null") {
        throw new Error("Quarantine: _id is missing or null");
    }
    if (!types.transId || types.transId === "null") {
        throw new Error("Quarantine: transId is missing or null");
    }
    
    return types;
}

module.exports = {
    buildCorePayloadSourceFieldTypes,
    getBsonType
};
```

## 2. File: `test/fixtures/core-payload-contract.cjs`
**Hành động**: Tạo mới hoặc Thêm vào cuối file.
**Mục tiêu**: Chứa các Object mẫu (Fixtures) cho Unit Test, bao quát các case hợp lệ và Quarantine.

```javascript
const contractFixtures = {
    // 1. Payload chuẩn với đầy đủ EJSON Types
    validFullEjsonPayload: {
        _id: { $oid: "60b8d295f1dca20015942345" },
        transId: "TRX-9999",
        originalTransHisId: "TRX-8888",
        requestId: null,             // Explicit null
        transType: "PAYMENT",
        status: "SUCCESS",
        sysTrans: true,
        value: { $numberInt: "50000" }, 
        noted: "",                   // Empty string, không tự đổi thành null
        serviceCode: null,
        channelId: null,
        isDelete: false,             // False, không đổi thành missing
        createdAt: { $date: "2023-01-01T00:00:00Z" },
        updatedAt: { $date: "2023-01-01T00:00:00Z" },
        info: { billId: "B-123" },   // Nested object
        sender: {},                  // Empty object
        receiver: [],                // Empty array
        error: null
    },

    // Output kỳ vọng cho validFullEjsonPayload (không có key nếu thiếu field)
    expectedValidFullTypes: {
        _id: "objectId",
        transId: "string",
        originalTransHisId: "string",
        requestId: "null",
        transType: "string",
        status: "string",
        sysTrans: "boolean",
        value: "int32",
        noted: "string",
        serviceCode: "null",
        channelId: "null",
        isDelete: "boolean",
        createdAt: "date",
        updatedAt: "date",
        info: "object",
        sender: "object",
        receiver: "array",
        error: "null"
    },

    // 2. Payload bị lỗi thiếu _id
    quarantineMissingId: {
        transId: "TRX-9999",
        value: { $numberInt: "50000" }
    },

    // 3. Payload dính field rác chưa khai báo (Garbage Field)
    quarantineUnknownField: {
        _id: { $oid: "60b8d295f1dca20015942345" },
        transId: "TRX-9999",
        randomGarbageField: "banned"
    }
};

module.exports = contractFixtures;
```
