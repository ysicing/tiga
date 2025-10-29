# Quick Start Guide: K8s Terminal Recording & Audit

**Feature Branch**: `010-k8s-pod-009`
**Date**: 2025-10-27
**Status**: Implementation Guide

## Overview

This guide provides step-by-step instructions to verify the K8s terminal recording and audit enhancement features. Follow these steps to manually test the implementation before running automated tests.

## Prerequisites

- Go 1.24+ installed
- Docker installed and running
- `kind` or `minikube` installed (for local K8s cluster)
- `curl` and `jq` installed (for API testing)
- Tiga project cloned and built

## Step 1: Environment Setup

### 1.1 Start Local K8s Cluster

```bash
# Create a kind cluster (recommended)
kind create cluster --name tiga-test

# Verify cluster is running
kubectl cluster-info
kubectl get nodes
```

### 1.2 Build and Start Tiga Service

```bash
# Navigate to project root
cd /root/go/src/github.com/ysicing/tiga

# Build backend (includes frontend build)
task backend

# Ensure recording feature is enabled in config.yaml
cat > config.yaml <<EOF
server:
  port: 12306
database:
  type: sqlite
  path: tiga.db
recording:
  enabled: true              # ✅ Enable recording
  base_path: ./recordings
  storage_type: local
  retention_days: 90
audit:
  buffer_size: 1000
  batch_size: 100
  flush_interval: 1s
EOF

# Start the service
./bin/tiga

# Expected output:
# [INFO] Starting Tiga v0.x.x
# [INFO] Database migration completed
# [INFO] Server listening on :12306
```

### 1.3 Initialize Test Data

```bash
# Open a new terminal

# Login and get JWT token
TOKEN=$(curl -s -X POST http://localhost:12306/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}' | jq -r '.data.access_token')

# Verify token
echo "JWT Token: $TOKEN"

# Import local K8s cluster
CLUSTER_ID=$(curl -s -X POST http://localhost:12306/api/v1/clusters \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name":"tiga-test",
    "kubeconfig":"'$(cat ~/.kube/config | base64 -w0)'"
  }' | jq -r '.data.id')

echo "Cluster ID: $CLUSTER_ID"

# Verify cluster is added
curl -s -X GET http://localhost:12306/api/v1/clusters \
  -H "Authorization: Bearer $TOKEN" | jq '.data'
```

**Expected Output**:
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "tiga-test",
      "health_status": "healthy",
      "node_count": 1,
      "pod_count": 0
    }
  ]
}
```

## Step 2: Verify Node Terminal Recording

### 2.1 Connect to Node Terminal (Manual Web UI)

1. Open browser: `http://localhost:12306`
2. Login with admin/admin
3. Navigate to **K8s → Clusters → tiga-test**
4. Click **Nodes** tab
5. Click **Terminal** icon for the node
6. Execute commands in the terminal:
   ```bash
   ls -la /
   ps aux
   uptime
   df -h
   ```
7. Wait 10-15 seconds (generate enough activity)
8. Close the terminal (disconnect)

### 2.2 Verify Recording Created

```bash
# Query recordings (should see k8s_node recording)
curl -s -X GET "http://localhost:12306/api/v1/recordings?recording_type=k8s_node" \
  -H "Authorization: Bearer $TOKEN" | jq '.'

# Expected output:
# {
#   "data": [
#     {
#       "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
#       "session_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
#       "user_id": "550e8400-e29b-41d4-a716-446655440000",
#       "username": "admin",
#       "recording_type": "k8s_node",
#       "type_metadata": {
#         "cluster_id": "550e8400-e29b-41d4-a716-446655440000",
#         "node_name": "tiga-test-control-plane"
#       },
#       "started_at": "2025-10-27T10:00:00Z",
#       "ended_at": "2025-10-27T10:00:15Z",
#       "duration": 15,
#       "storage_path": "./recordings/k8s_node/2025-10-27/7c9e6679-7425-40de-944b-e07fc1f90ae7.cast",
#       "file_size": 12345,
#       "format": "asciinema"
#     }
#   ],
#   "total": 1
# }

# Save recording ID for later
RECORDING_ID=$(curl -s -X GET "http://localhost:12306/api/v1/recordings?recording_type=k8s_node" \
  -H "Authorization: Bearer $TOKEN" | jq -r '.data[0].id')

echo "Recording ID: $RECORDING_ID"
```

### 2.3 Verify Recording File Exists

```bash
# Check recording file exists
ls -lh ./recordings/k8s_node/$(date +%Y-%m-%d)/

# Expected: 7c9e6679-7425-40de-944b-e07fc1f90ae7.cast (or similar UUID.cast)

# Verify Asciinema v2 format
head -n 3 ./recordings/k8s_node/$(date +%Y-%m-%d)/*.cast

# Expected output (Asciinema v2 format):
# {"version": 2, "width": 120, "height": 30, "timestamp": 1730000000, "title": "k8s-node: tiga-test-control-plane"}
# [0.123456, "o", "root@tiga-test-control-plane:~# "]
# [1.234567, "i", "ls -la /\r"]
```

### 2.4 Playback Recording

```bash
# Get playback content (Asciinema format)
curl -s -X GET "http://localhost:12306/api/v1/recordings/$RECORDING_ID/play" \
  -H "Authorization: Bearer $TOKEN" > /tmp/recording.cast

# Verify format
file /tmp/recording.cast
# Expected: /tmp/recording.cast: ASCII text

# Play with asciinema (if installed)
asciinema play /tmp/recording.cast

# Or view in browser (copy recording ID and paste into Web UI Recordings page)
```

**✅ Success Criteria**:
- Recording appears in API response with `recording_type: "k8s_node"`
- `type_metadata` contains `cluster_id` and `node_name`
- Recording file exists in correct path: `./recordings/k8s_node/{YYYY-MM-DD}/{id}.cast`
- File is valid Asciinema v2 format (Header + Frames)
- Playback shows terminal session activity

## Step 3: Verify Container Terminal Recording

### 3.1 Deploy Test Pod

```bash
# Create a test deployment
kubectl create deployment nginx --image=nginx:latest

# Wait for pod to be ready
kubectl wait --for=condition=Ready pod -l app=nginx --timeout=60s

# Get pod name
POD_NAME=$(kubectl get pods -l app=nginx -o jsonpath='{.items[0].metadata.name}')
echo "Pod Name: $POD_NAME"
```

### 3.2 Connect to Container Terminal (Manual Web UI)

1. Open browser: `http://localhost:12306`
2. Navigate to **K8s → Workloads → Pods**
3. Find the `nginx-*` pod
4. Click **Terminal** icon
5. Select container: `nginx`
6. Execute commands in the terminal:
   ```bash
   ls -la /etc/nginx
   cat /etc/nginx/nginx.conf | head -20
   ps aux
   env
   ```
7. Wait 10-15 seconds
8. Close the terminal

### 3.3 Verify Container Recording Created

```bash
# Query k8s_pod recordings
curl -s -X GET "http://localhost:12306/api/v1/recordings?recording_type=k8s_pod" \
  -H "Authorization: Bearer $TOKEN" | jq '.'

# Expected output includes:
# {
#   "recording_type": "k8s_pod",
#   "type_metadata": {
#     "cluster_id": "550e8400-e29b-41d4-a716-446655440000",
#     "namespace": "default",
#     "pod_name": "nginx-6c8b5b5d4f-abc123",
#     "container_name": "nginx"
#   }
# }
```

**✅ Success Criteria**:
- Recording appears with `recording_type: "k8s_pod"`
- `type_metadata` contains `cluster_id`, `namespace`, `pod_name`, `container_name`
- Recording file exists in `./recordings/k8s_pod/{YYYY-MM-DD}/{id}.cast`

## Step 4: Verify Resource Operation Audit

### 4.1 Create Resource (Deployment)

```bash
# Create a deployment via API
curl -s -X POST "http://localhost:12306/api/v1/k8s/clusters/$CLUSTER_ID/deployments" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "default",
    "name": "test-app",
    "image": "nginx:latest",
    "replicas": 3
  }' | jq '.'

# Wait a moment for audit log to be written (async, < 1 second)
sleep 2
```

### 4.2 Query Audit Logs

```bash
# Query K8s audit events (resource creation)
curl -s -X GET "http://localhost:12306/api/v1/audit/events?subsystem=kubernetes&action=CreateResource" \
  -H "Authorization: Bearer $TOKEN" | jq '.data[0]'

# Expected output:
# {
#   "id": "8d7c6b5a-4321-fedc-ba98-76543210abcd",
#   "timestamp": 1730019600000,
#   "action": "CreateResource",
#   "resource_type": "deployment",
#   "resource": {
#     "type": "deployment",
#     "identifier": "test-app",
#     "data": {
#       "cluster_id": "550e8400-e29b-41d4-a716-446655440000",
#       "cluster_name": "tiga-test",
#       "namespace": "default",
#       "resource_name": "test-app",
#       "success": "true"
#     }
#   },
#   "subsystem": "kubernetes",
#   "user": {
#     "uid": "550e8400-e29b-41d4-a716-446655440000",
#     "username": "admin",
#     "type": "user"
#   },
#   "client_ip": "127.0.0.1"
# }
```

### 4.3 Update Resource

```bash
# Update deployment (scale replicas)
curl -s -X PATCH "http://localhost:12306/api/v1/k8s/clusters/$CLUSTER_ID/deployments/default/test-app" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"replicas": 5}' | jq '.'

sleep 2

# Query update audit
curl -s -X GET "http://localhost:12306/api/v1/audit/events?subsystem=kubernetes&action=UpdateResource&resource_name=test-app" \
  -H "Authorization: Bearer $TOKEN" | jq '.data[0].resource.data'

# Expected: includes "change_summary" field
# {
#   "cluster_id": "...",
#   "namespace": "default",
#   "resource_name": "test-app",
#   "change_summary": "replicas: 3 -> 5",
#   "success": "true"
# }
```

### 4.4 Delete Resource

```bash
# Delete deployment
curl -s -X DELETE "http://localhost:12306/api/v1/k8s/clusters/$CLUSTER_ID/deployments/default/test-app" \
  -H "Authorization: Bearer $TOKEN" | jq '.'

sleep 2

# Query delete audit
curl -s -X GET "http://localhost:12306/api/v1/audit/events?subsystem=kubernetes&action=DeleteResource" \
  -H "Authorization: Bearer $TOKEN" | jq '.data[0]'
```

**✅ Success Criteria**:
- All 3 operations (Create, Update, Delete) generate audit events
- Audit events contain required fields: `cluster_id`, `namespace`, `resource_name`, `success`
- Update audit includes `change_summary`
- All audits have `subsystem: "kubernetes"`

## Step 5: Verify Terminal Access Audit

### 5.1 Query Terminal Access Audit

```bash
# Query node terminal access audit
curl -s -X GET "http://localhost:12306/api/v1/audit/events?subsystem=kubernetes&action=NodeTerminalAccess" \
  -H "Authorization: Bearer $TOKEN" | jq '.data[0]'

# Expected output:
# {
#   "id": "af9e8d7c-6543-2fed-dcba-98765432edcb",
#   "action": "NodeTerminalAccess",
#   "resource_type": "k8s_node",
#   "resource": {
#     "type": "k8s_node",
#     "identifier": "tiga-test-control-plane",
#     "data": {
#       "cluster_id": "550e8400-e29b-41d4-a716-446655440000",
#       "resource_name": "tiga-test-control-plane",
#       "recording_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
#       "success": "true"
#     }
#   },
#   "subsystem": "kubernetes",
#   "user": {
#     "uid": "550e8400-e29b-41d4-a716-446655440000",
#     "username": "admin",
#     "type": "user"
#   }
# }

# Query pod terminal access audit
curl -s -X GET "http://localhost:12306/api/v1/audit/events?subsystem=kubernetes&action=PodTerminalAccess" \
  -H "Authorization: Bearer $TOKEN" | jq '.data[0]'
```

**✅ Success Criteria**:
- Terminal access audits are created for both node and container terminals
- Audit events have `recording_id` linking to TerminalRecording
- `resource_type` is `k8s_node` or `k8s_pod`

## Step 6: Verify Audit Log Query Features

### 6.1 Filter by User

```bash
# Get current user ID
USER_ID=$(curl -s -X GET http://localhost:12306/api/v1/auth/me \
  -H "Authorization: Bearer $TOKEN" | jq -r '.data.id')

# Filter by user
curl -s -X GET "http://localhost:12306/api/v1/audit/events?subsystem=kubernetes&user_id=$USER_ID" \
  -H "Authorization: Bearer $TOKEN" | jq '.total'

# Should return count > 0
```

### 6.2 Filter by Time Range

```bash
# Get events from last 1 hour
START_TIME=$(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%SZ)
END_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)

curl -s -X GET "http://localhost:12306/api/v1/audit/events?subsystem=kubernetes&start_time=$START_TIME&end_time=$END_TIME" \
  -H "Authorization: Bearer $TOKEN" | jq '.data | length'

# Should return count of events in the time range
```

### 6.3 Filter by Action Type

```bash
# Get all modify operations (create + update + delete)
curl -s -X GET "http://localhost:12306/api/v1/audit/events?subsystem=kubernetes" \
  -H "Authorization: Bearer $TOKEN" | \
  jq '[.data[] | select(.action | test("CreateResource|UpdateResource|DeleteResource"))] | length'

# Get all read operations
curl -s -X GET "http://localhost:12306/api/v1/audit/events?subsystem=kubernetes&action=ViewResource" \
  -H "Authorization: Bearer $TOKEN" | jq '.total'
```

### 6.4 Pagination

```bash
# Get page 1 (first 50 events)
curl -s -X GET "http://localhost:12306/api/v1/audit/events?subsystem=kubernetes&page=1&page_size=50" \
  -H "Authorization: Bearer $TOKEN" | jq '{total: .total, page: .page, page_size: .page_size, count: (.data | length)}'

# Expected:
# {
#   "total": 100,
#   "page": 1,
#   "page_size": 50,
#   "count": 50
# }
```

### 6.5 Audit Statistics

```bash
# Get audit statistics
curl -s -X GET "http://localhost:12306/api/v1/audit/stats?subsystem=kubernetes" \
  -H "Authorization: Bearer $TOKEN" | jq '.'

# Expected output:
# {
#   "total_count": 100,
#   "by_action": {
#     "CreateResource": 15,
#     "UpdateResource": 8,
#     "DeleteResource": 3,
#     "ViewResource": 50,
#     "NodeTerminalAccess": 12,
#     "PodTerminalAccess": 12
#   },
#   "by_resource_type": {
#     "deployment": 10,
#     "service": 5,
#     "pod": 20,
#     "k8s_node": 12,
#     "k8s_pod": 12
#   },
#   "success_rate": {
#     "successful": 98,
#     "failed": 2,
#     "rate": 0.98
#   }
# }
```

**✅ Success Criteria**:
- All filter parameters work correctly
- Pagination returns correct page/total/count
- Statistics accurately reflect audit data

## Step 7: Verify Recording Limits and Cleanup

### 7.1 Test 2-Hour Limit (Optional - Long Running)

⚠️ **This test takes 2 hours to complete. Skip for quick verification.**

```bash
# Connect to a terminal and keep it open for > 2 hours
# Expected behavior:
# - At 2:00:00, recording stops automatically
# - User receives WebSocket notification: "Recording stopped: 2 hour limit reached"
# - Terminal connection remains active (not disconnected)
# - Recording duration = 7200 seconds (exactly 2 hours)
```

### 7.2 Test Recording Cleanup

```bash
# Check current recordings count
BEFORE_COUNT=$(curl -s -X GET "http://localhost:12306/api/v1/recordings" \
  -H "Authorization: Bearer $TOKEN" | jq '.total')

echo "Before cleanup: $BEFORE_COUNT recordings"

# Manually trigger cleanup (90-day retention, dry run)
curl -s -X POST "http://localhost:12306/api/v1/recordings/cleanup" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"dry_run": true, "retention_days": 90}' | jq '.'

# Expected: deleted_count = 0 (all recordings are < 90 days old)

# Test with shorter retention (1 day, dry run)
curl -s -X POST "http://localhost:12306/api/v1/recordings/cleanup" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"dry_run": true, "retention_days": 1}' | jq '.'

# Expected: deleted_count > 0, deleted_recordings array populated
```

### 7.3 Test Audit Cleanup

```bash
# Check current audit events count
AUDIT_BEFORE=$(curl -s -X GET "http://localhost:12306/api/v1/audit/events?subsystem=kubernetes" \
  -H "Authorization: Bearer $TOKEN" | jq '.total')

echo "Before cleanup: $AUDIT_BEFORE audit events"

# Manually trigger cleanup (90-day retention, dry run)
curl -s -X POST "http://localhost:12306/api/v1/audit/cleanup" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"dry_run": true, "retention_days": 90, "subsystem": "kubernetes"}' | jq '.'

# Expected: deleted_count = 0 (all events are < 90 days old)
```

**✅ Success Criteria**:
- Dry run cleanup returns correct deleted_count and deleted_recordings/events
- No data is actually deleted when dry_run=true
- Real cleanup (dry_run=false) deletes expired records

## Step 8: Frontend Verification (Manual)

### 8.1 Recordings Page

1. Open browser: `http://localhost:12306`
2. Navigate to **System Management → Recordings**
3. Verify:
   - ✅ Filter by `recording_type` includes `k8s_node` and `k8s_pod` options
   - ✅ Filter by cluster works (select tiga-test cluster)
   - ✅ Recordings table shows K8s recordings with cluster/node/pod info
   - ✅ Click **Play** button opens Asciinema player
   - ✅ Playback is smooth and accurate

### 8.2 Audit Logs Page

1. Navigate to **System Management → Audit Logs**
2. Verify:
   - ✅ Filter by `subsystem` includes `kubernetes` option
   - ✅ Filter by `action` includes K8s-specific actions (CreateResource, ViewResource, NodeTerminalAccess, etc.)
   - ✅ Filter by `resource_type` includes K8s resource types
   - ✅ Filter by cluster works
   - ✅ Time range filter works
   - ✅ Click **Details** button shows full audit event with metadata
   - ✅ Update operations show change summary
   - ✅ Terminal access audits link to recordings (click recording_id navigates to Recordings page)

### 8.3 K8s Integration

1. Navigate to **K8s → Clusters → tiga-test**
2. Verify:
   - ✅ Node terminal button works (opens terminal, starts recording)
   - ✅ Pod terminal button works (opens terminal, starts recording)
   - ✅ Terminal operations are smooth (< 100ms delay)
   - ✅ Creating/updating/deleting resources feels fast (< 50ms delay)
   - ✅ No noticeable performance degradation

## Step 9: Performance Verification

### 9.1 Terminal Connection Latency

```bash
# Measure terminal connection time
time curl -s -X POST "http://localhost:12306/api/v1/k8s/clusters/$CLUSTER_ID/nodes/tiga-test-control-plane/terminal" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Connection: Upgrade" \
  -H "Upgrade: websocket"

# Target: < 100ms (recording overhead)
```

### 9.2 Resource Operation Latency

```bash
# Measure deployment creation time
time curl -s -X POST "http://localhost:12306/api/v1/k8s/clusters/$CLUSTER_ID/deployments" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"namespace":"default","name":"perf-test","image":"nginx:latest","replicas":1}' > /dev/null

# Target: < 1s total (< 50ms audit overhead)
```

### 9.3 Audit Query Performance

```bash
# Measure audit query time
time curl -s -X GET "http://localhost:12306/api/v1/audit/events?subsystem=kubernetes&page=1&page_size=50" \
  -H "Authorization: Bearer $TOKEN" > /dev/null

# Target: < 500ms (with proper indexes)
```

**✅ Success Criteria**:
- Terminal connection: < 100ms overhead
- Resource operations: < 50ms audit overhead
- Audit queries: < 500ms response time

## Step 10: Cleanup

```bash
# Delete test cluster
kind delete cluster --name tiga-test

# Stop Tiga service (Ctrl+C)

# Remove test database and recordings (optional)
rm -rf tiga.db recordings/
```

## Troubleshooting

### Recording Not Created

- Check config.yaml: `recording.enabled: true`
- Check logs: `grep "recording" tiga.log`
- Check storage path: `ls -la recordings/k8s_node/$(date +%Y-%m-%d)/`

### Audit Event Not Logged

- Check logs: `grep "audit" tiga.log`
- Verify async logger is running: `grep "AsyncLogger" tiga.log`
- Check database: `sqlite3 tiga.db "SELECT COUNT(*) FROM audit_events WHERE subsystem='kubernetes';"`

### Performance Issues

- Check async audit buffer size: `audit.buffer_size` (should be >= 1000)
- Check database indexes: `sqlite3 tiga.db ".schema audit_events"`
- Enable query logging: `LOG_LEVEL=debug ./bin/tiga`

## Success Checklist

- [ ] Node terminal recording works (Asciinema v2 format)
- [ ] Container terminal recording works (Asciinema v2 format)
- [ ] Recording file path follows convention: `./recordings/k8s_{node|pod}/{YYYY-MM-DD}/{id}.cast`
- [ ] Recording metadata contains cluster_id, node_name/pod_name
- [ ] Playback works (Web UI + asciinema CLI)
- [ ] Resource creation audit logged (CreateResource)
- [ ] Resource update audit logged (UpdateResource with change_summary)
- [ ] Resource deletion audit logged (DeleteResource)
- [ ] Terminal access audit logged (NodeTerminalAccess, PodTerminalAccess)
- [ ] Audit events link to recordings (recording_id field)
- [ ] Audit filters work (subsystem, action, resource_type, user, time range)
- [ ] Pagination works correctly
- [ ] Audit statistics accurate
- [ ] Cleanup dry run works
- [ ] Performance targets met (< 100ms terminal, < 50ms audit, < 500ms query)
- [ ] Frontend UI functional (filters, playback, details)

---

**Next Steps**:
1. Run automated contract tests: `go test ./tests/contract/k8s/...`
2. Run integration tests: `go test ./tests/integration/k8s/...`
3. Review implementation against spec.md requirements
4. Performance benchmarking with 100+ concurrent terminals
