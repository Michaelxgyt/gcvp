#!/bin/bash
set -e; echo "--- V2Ray Proxy Entrypoint (Cloud Native Method) ---"
PROJECT_ID=$(curl -s "http://metadata.google.internal/computeMetadata/v1/project/project-id" -H "Metadata-Flavor: Google")
if [ -z "$PROJECT_ID" ]; then echo "FATAL: Could not retrieve PROJECT_ID."; exit 1; fi; echo "Project ID detected: $PROJECT_ID"
ACCESS_TOKEN=$(curl -s "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token" -H "Metadata-Flavor: Google" | jq -r '.access_token')
if [ -z "$ACCESS_TOKEN" ] || [ "$ACCESS_TOKEN" = "null" ]; then echo "FATAL: Could not retrieve ACCESS_TOKEN."; exit 1; fi; echo "Access token retrieved."
API_URL="https://firestore.googleapis.com/v1/projects/${PROJECT_ID}/databases/(default)/documents:runQuery"
QUERY_BODY='{"structuredQuery":{"from":[{"collectionId":"v2ray_users"}],"where":{"fieldFilter":{"field":{"fieldPath":"is_active"},"op":"EQUAL","value":{"booleanValue":true}}}}}'
echo "Querying Firestore for active users..."; USERS_RESPONSE=$(curl -s -X POST "${API_URL}" -H "Authorization: Bearer ${ACCESS_TOKEN}" -H "Content-Type: application/json" -d "${QUERY_BODY}")
if echo "$USERS_RESPONSE" | jq -e '.error' > /dev/null; then echo "FATAL: Firestore API error:"; echo "$USERS_RESPONSE"; exit 1; fi
CLIENTS_JSON=$(echo "$USERS_RESPONSE" | jq -c '[.[] | .document.fields | select(.uuid and .email) | {id: .uuid.stringValue, email: .email.stringValue, level: 0}] | select(length > 0)')
if [ -z "$CLIENTS_JSON" ] || [ "$CLIENTS_JSON" = "[]" ] || [ "$CLIENTS_JSON" = "null" ]; then echo "WARNING: No active users found."; CLIENTS_JSON="[]"; else echo "Successfully retrieved user data."; fi
jq --argjson clients "$CLIENTS_JSON" '.inbounds[0].settings.clients = $clients' /app/config_template.json > /etc/xray/config.json
echo "Final config generated. Starting Xray..."; exec /usr/bin/xray -config /etc/xray/config.json
