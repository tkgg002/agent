#!/bin/bash
# Description: Quota check and Key Rotation script for Antigravity
# Usage: ./quota_check.sh <model_name>

MODEL=$1
SOURCE_FILE="/Users/trainguyen/Documents/work/agent/models.env"

if [ -f "$SOURCE_FILE" ]; then
    source "$SOURCE_FILE"
else
    echo "No models.env found, skipping rotation logic."
    exit 0
fi

# Logic giả lập: Nếu nhận tham số từ CLI báo lỗi 429, thực hiện xoay vòng key
# Trong thực tế, CC CLI sẽ tự quản lý session, nhưng script này giúp Brain chuẩn bị môi trường

echo "[Quota] Checking limits for role: $MODEL (Role identifier)"

# Logic xoay vòng Pool: provider:model:key
# Chúng ta sẽ đọc Pool tương ứng (BRAIN_POOL hoặc MUSCLE_POOL)
if [[ "$MODEL" == *"flash"* ]]; then
    POOL_VAL=$MUSCLE_POOL
    POOL_NAME="MUSCLE_POOL"
else
    POOL_VAL=$BRAIN_POOL
    POOL_NAME="BRAIN_POOL"
fi

if [ ! -z "$POOL_VAL" ]; then
    IFS=',' read -ra ITEMS <<< "$POOL_VAL"
    # Lấy item đầu tiên (đang active)
    ACTIVE_ITEM=${ITEMS[0]}
    
    # Giải mã item: provider:model:key
    IFS=':' read -ra PARTS <<< "$ACTIVE_ITEM"
    PROVIDER=${PARTS[0]}
    MODEL_NAME=${PARTS[1]}
    KEY=${PARTS[2]}
    
    echo "[Quota] Active Provider: $PROVIDER"
    echo "[Quota] Active Model: $MODEL_NAME"
    
    # Giả lập xoay vòng: Đưa item vừa dùng xuống cuối
    NEW_POOL=""
    for ((i=1; i<${#ITEMS[@]}; i++)); do
        NEW_POOL+="${ITEMS[i]},"
    done
    NEW_POOL+="$ACTIVE_ITEM"
    
    echo "[Quota] Pool rotated for $POOL_NAME."
else
    echo "[Quota] No Model Pool configured."
fi

exit 0
