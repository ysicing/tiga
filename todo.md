# Tiga Development TODO List

> Last Updated: 2025-10-07
> Branch: 002-nezha-webssh
> Priority: WebSSH Terminal & Service Probe Features

## 🎯 Current Sprint Focus

### 🖥️ WebSSH Terminal Implementation (6 days total)

#### Phase 1: Backend Enhancement (2 days)

##### WebSocket Handler (`internal/api/handlers/webssh_handler.go`)
- [ ] Complete HandleWebSocket method implementation
  - [ ] Define WebSocket message protocol (JSON format)
  - [ ] Handle resize events from frontend
  - [ ] Implement ping/pong for connection keep-alive
  - [ ] Add graceful connection close handling
- [ ] Implement connection pool management
  - [ ] Maximum connections per user limit
  - [ ] Connection timeout (default: 30 minutes)
  - [ ] Connection cleanup on disconnect
- [ ] Add error handling and reconnection mechanism
  - [ ] Handle network interruptions
  - [ ] Automatic reconnection with backoff
  - [ ] Error message propagation to frontend
- [ ] Add session authentication and authorization
  - [ ] Verify user permissions for host access
  - [ ] Session token validation
  - [ ] Rate limiting for connections

##### Terminal Manager (`internal/services/host/terminal_manager.go`)
- [ ] Complete PTY session management
  - [ ] Create PTY with proper size
  - [ ] Handle PTY resize events
  - [ ] Manage PTY lifecycle
- [ ] Implement input/output stream processing
  - [ ] Handle stdin from WebSocket
  - [ ] Stream stdout/stderr to WebSocket
  - [ ] Buffer management for large outputs
- [ ] Add session timeout handling
  - [ ] Idle timeout detection (configurable)
  - [ ] Warning before timeout
  - [ ] Graceful session termination
- [ ] Implement session recording (optional)
  - [ ] Record terminal sessions to file
  - [ ] Playback capability
  - [ ] Storage management

##### Agent Manager (`internal/services/host/agent_manager.go`)
- [ ] Implement Agent SSH proxy protocol
  - [ ] Define gRPC service for SSH proxy
  - [ ] Message format for SSH commands
  - [ ] Stream handling for terminal I/O
- [ ] Forward SSH connections to Agent
  - [ ] Connection request routing
  - [ ] Bidirectional stream forwarding
  - [ ] Connection state tracking
- [ ] Add security authentication mechanism
  - [ ] Agent authentication
  - [ ] End-to-end encryption
  - [ ] Command whitelisting/blacklisting

#### Phase 2: Agent Implementation (2 days)

##### SSH Proxy Service (`cmd/tiga-agent/ssh_proxy.go`) - NEW FILE
- [ ] Create SSH proxy service structure
  ```go
  type SSHProxy struct {
    maxConnections int
    connections    map[string]*SSHConnection
    mu            sync.RWMutex
  }
  ```
- [ ] Implement SSH connection handling
  - [ ] Receive SSH requests from server
  - [ ] Create local SSH client connection
  - [ ] Handle authentication (key/password)
  - [ ] Manage connection lifecycle
- [ ] Implement bidirectional data forwarding
  - [ ] Forward stdin to SSH session
  - [ ] Stream stdout/stderr back to server
  - [ ] Handle control sequences properly
- [ ] Add security mechanisms
  - [ ] Command execution restrictions
  - [ ] Audit logging for all commands
  - [ ] Connection encryption using TLS
  - [ ] Rate limiting per connection
- [ ] Implement error handling
  - [ ] Connection failure recovery
  - [ ] Graceful disconnection
  - [ ] Error reporting to server

##### Agent Integration (`cmd/tiga-agent/main.go`)
- [ ] Integrate SSH proxy into agent main loop
- [ ] Add configuration for SSH proxy
  - [ ] Enable/disable SSH proxy
  - [ ] Maximum concurrent connections
  - [ ] Allowed command whitelist
- [ ] Add health checks for SSH service
- [ ] Implement metrics collection for SSH sessions

#### Phase 3: Frontend Optimization (1 day)

##### Terminal UI (`ui/src/pages/hosts/host-ssh-page.tsx`)
- [ ] Optimize xterm.js configuration
  - [ ] Fine-tune scrollback buffer size
  - [ ] Configure cursor style and blinking
  - [ ] Set appropriate font family and size
  - [ ] Enable/disable sound
- [ ] Implement adaptive terminal sizing
  - [ ] Detect container size changes
  - [ ] Calculate rows/cols based on container
  - [ ] Send resize events to backend
  - [ ] Handle mobile responsive layout
- [ ] Add copy/paste support
  - [ ] Right-click context menu
  - [ ] Ctrl+C/Ctrl+V handling
  - [ ] Selection highlighting
  - [ ] Clipboard API integration
- [ ] Implement terminal themes
  - [ ] Predefined themes (dark, light, solarized, etc.)
  - [ ] Custom theme configuration
  - [ ] Theme persistence in localStorage
  - [ ] Real-time theme switching

##### Connection Management UI
- [ ] Add connection status indicator
  - [ ] Connected/Disconnected states
  - [ ] Reconnecting animation
  - [ ] Latency display
  - [ ] Data transfer indicators
- [ ] Implement automatic reconnection
  - [ ] Exponential backoff strategy
  - [ ] Maximum retry attempts
  - [ ] User notification on failure
- [ ] Enhance error notifications
  - [ ] User-friendly error messages
  - [ ] Actionable error suggestions
  - [ ] Error log viewer
- [ ] Add session management features
  - [ ] Multiple tab support
  - [ ] Session history
  - [ ] Quick connect shortcuts

#### Phase 4: Integration Testing (1 day)

##### End-to-End Tests
- [ ] Connection establishment test
  - [ ] Successful connection flow
  - [ ] Authentication failure handling
  - [ ] Permission denied scenarios
- [ ] Multi-session concurrency test
  - [ ] Open 100+ concurrent sessions
  - [ ] Memory usage monitoring
  - [ ] CPU usage monitoring
- [ ] Network disruption recovery test
  - [ ] Simulate network interruption
  - [ ] Verify automatic reconnection
  - [ ] Check session state preservation
- [ ] Performance stress test
  - [ ] Large output handling (e.g., cat large file)
  - [ ] Rapid input testing
  - [ ] Long-running session stability

##### Security Tests
- [ ] Command injection prevention
- [ ] Session hijacking prevention
- [ ] Rate limiting verification
- [ ] Authentication bypass attempts

---

### 📊 Service Probe Implementation (9 days total)

#### Phase 1: Backend Enhancement (2 days)

##### Probe Scheduler (`internal/services/monitor/probe_scheduler.go`)
- [ ] Implement HTTP/HTTPS probe
  - [ ] GET/POST/HEAD methods
  - [ ] Custom headers support
  - [ ] Response code validation
  - [ ] Response time measurement
  - [ ] Content validation (regex)
  - [ ] SSL certificate validation
- [ ] Implement TCP port probe
  - [ ] Connection establishment test
  - [ ] Custom payload sending
  - [ ] Response validation
  - [ ] Connection timeout handling
- [ ] Implement ICMP ping probe
  - [ ] Packet loss calculation
  - [ ] RTT measurement
  - [ ] Jitter calculation
  - [ ] Multi-packet statistics
- [ ] Add custom script probe
  - [ ] Script execution framework
  - [ ] Timeout enforcement
  - [ ] Output parsing
  - [ ] Exit code validation

##### Probe Result Model (`internal/models/service_probe_result.go`)
- [ ] Enhance data model
  ```go
  type ServiceProbeResult struct {
    ID           uuid.UUID
    MonitorID    uuid.UUID
    AgentID      uuid.UUID
    Status       string // up/down/degraded
    ResponseTime int64  // microseconds
    StatusCode   int
    ErrorMessage string
    Metadata     JSONB // additional probe-specific data
    ProbeAt      time.Time
  }
  ```
- [ ] Add response time statistics
  - [ ] P50/P95/P99 percentiles
  - [ ] Average response time
  - [ ] Standard deviation
- [ ] Implement availability calculation
  - [ ] Uptime percentage
  - [ ] MTBF (Mean Time Between Failures)
  - [ ] MTTR (Mean Time To Recovery)
- [ ] Add trend analysis
  - [ ] Response time trends
  - [ ] Availability trends
  - [ ] Anomaly detection

##### Service Repository (`internal/repository/service_repository.go`)
- [ ] Optimize probe result storage
  - [ ] Batch insert for results
  - [ ] Data retention policies
  - [ ] Automatic old data cleanup
- [ ] Implement time-series queries
  - [ ] Range queries with aggregation
  - [ ] Downsampling for long ranges
  - [ ] Moving averages
- [ ] Add statistical aggregation
  - [ ] Hourly/daily/weekly summaries
  - [ ] Service group aggregations
  - [ ] Cross-service comparisons
- [ ] Create availability reports
  - [ ] SLA compliance calculation
  - [ ] Incident detection
  - [ ] Outage duration tracking

#### Phase 2: Agent Probe Executor (2 days)

##### Probe Executor (`cmd/tiga-agent/prober/executor.go`) - NEW FILE
- [ ] Create probe executor framework
  ```go
  type ProbeExecutor struct {
    httpClient  *http.Client
    maxWorkers  int
    workQueue   chan ProbeTask
    results     chan ProbeResult
  }
  ```
- [ ] Implement distributed probe execution
  - [ ] Receive probe tasks from server
  - [ ] Queue management with priorities
  - [ ] Concurrent execution with worker pool
  - [ ] Result batching for efficiency
- [ ] Report probe results to server
  - [ ] Real-time result streaming
  - [ ] Batch upload for efficiency
  - [ ] Retry on failure
  - [ ] Result compression

##### HTTP Probe (`cmd/tiga-agent/prober/http.go`) - NEW FILE
- [ ] Implement HTTP/HTTPS probing
  - [ ] Support all HTTP methods
  - [ ] Custom headers and body
  - [ ] Follow redirects (configurable)
  - [ ] Cookie jar support
- [ ] Add authentication support
  - [ ] Basic authentication
  - [ ] Bearer token
  - [ ] Custom authentication headers
- [ ] Implement response validation
  - [ ] Status code checks
  - [ ] Response body regex matching
  - [ ] JSON path validation
  - [ ] Response time thresholds

##### TCP Probe (`cmd/tiga-agent/prober/tcp.go`) - NEW FILE
- [ ] Implement TCP connection testing
  - [ ] Connection establishment
  - [ ] TLS support
  - [ ] Custom handshake protocols
- [ ] Add protocol-specific probes
  - [ ] Redis PING
  - [ ] MySQL connection test
  - [ ] PostgreSQL connection test
  - [ ] MongoDB connection test

##### DNS Probe (`cmd/tiga-agent/prober/dns.go`) - NEW FILE
- [ ] Implement DNS resolution testing
  - [ ] A/AAAA record lookup
  - [ ] MX/TXT/CNAME queries
  - [ ] Response time measurement
  - [ ] Multiple DNS server support

##### Script Probe (`cmd/tiga-agent/prober/script.go`) - NEW FILE
- [ ] Implement custom script execution
  - [ ] Shell script support
  - [ ] Python script support
  - [ ] Output parsing
  - [ ] Resource limits (CPU/Memory)

#### Phase 3: Frontend Development (3 days)

##### Service Monitor List (`ui/src/pages/hosts/service-monitor-list.tsx`) - NEW FILE
- [ ] Create list page component
  - [ ] Table view with sorting/filtering
  - [ ] Card view for overview
  - [ ] Status indicators (up/down/degraded)
  - [ ] Quick actions menu
- [ ] Implement CRUD operations
  - [ ] Create monitor dialog
  - [ ] Edit monitor form
  - [ ] Delete with confirmation
  - [ ] Bulk operations support
- [ ] Add search and filters
  - [ ] Search by name/URL/host
  - [ ] Filter by status
  - [ ] Filter by type (HTTP/TCP/etc)
  - [ ] Filter by tag/group
- [ ] Implement batch operations
  - [ ] Enable/disable multiple monitors
  - [ ] Delete multiple monitors
  - [ ] Export/import configurations

##### Service Monitor Detail (`ui/src/pages/hosts/service-monitor-detail.tsx`) - NEW FILE
- [ ] Create detail page layout
  - [ ] Monitor configuration display
  - [ ] Current status card
  - [ ] Recent probe results table
  - [ ] Alert configuration section
- [ ] Add availability visualization
  - [ ] Availability percentage gauge
  - [ ] Uptime/downtime timeline
  - [ ] Incident list with duration
  - [ ] SLA compliance indicator
- [ ] Implement response time charts
  - [ ] Line chart for response times
  - [ ] Histogram distribution
  - [ ] Percentile trends
  - [ ] Comparison with baseline
- [ ] Add probe result logs
  - [ ] Paginated log viewer
  - [ ] Log filtering by status
  - [ ] Export logs feature
  - [ ] Real-time log streaming

##### Service Monitor Form (`ui/src/components/hosts/service-monitor-form.tsx`) - NEW FILE
- [ ] Create comprehensive form
  - [ ] Basic information (name, description)
  - [ ] Probe type selection
  - [ ] Target configuration
  - [ ] Schedule configuration
- [ ] Add probe-specific fields
  - [ ] HTTP: URL, method, headers, body
  - [ ] TCP: host, port, timeout
  - [ ] ICMP: host, packet count
  - [ ] Script: script content, interpreter
- [ ] Implement validation rules
  - [ ] URL format validation
  - [ ] Port range validation
  - [ ] Cron expression validation
  - [ ] Script syntax checking
- [ ] Add advanced options
  - [ ] Retry configuration
  - [ ] Timeout settings
  - [ ] Alert thresholds
  - [ ] Notification channels

##### Availability Chart (`ui/src/components/hosts/availability-chart.tsx`) - NEW FILE
- [ ] Create chart component
  - [ ] Availability heatmap (calendar view)
  - [ ] Response time line chart
  - [ ] Status timeline visualization
  - [ ] Statistical summary cards
- [ ] Add interactivity
  - [ ] Zoom and pan
  - [ ] Tooltip with details
  - [ ] Click to view incident
  - [ ] Export chart as image
- [ ] Implement real-time updates
  - [ ] WebSocket subscription
  - [ ] Smooth animations
  - [ ] Auto-refresh toggle
  - [ ] Notification on status change

#### Phase 4: Visualization & Alerting (2 days)

##### Monitoring Dashboard (`ui/src/pages/hosts/monitoring-dashboard.tsx`) - NEW FILE
- [ ] Create dashboard layout
  - [ ] Service status overview grid
  - [ ] Real-time alert feed
  - [ ] Top issues widget
  - [ ] Performance metrics summary
- [ ] Implement service topology
  - [ ] Interactive service map
  - [ ] Dependency visualization
  - [ ] Impact analysis view
  - [ ] Group by tags/categories
- [ ] Add real-time updates
  - [ ] WebSocket connection for live data
  - [ ] Status change animations
  - [ ] Alert notifications
  - [ ] Sound alerts (optional)

##### Alert Configuration (`ui/src/pages/hosts/alert-rules.tsx`)
- [ ] Enhance alert rule builder
  - [ ] Visual rule builder
  - [ ] Condition templates
  - [ ] Multi-condition support
  - [ ] Schedule-based muting
- [ ] Configure notification channels
  - [ ] Email configuration
  - [ ] Webhook setup
  - [ ] Slack integration
  - [ ] DingTalk integration
- [ ] Implement alert testing
  - [ ] Test rule execution
  - [ ] Preview notifications
  - [ ] Dry run mode
- [ ] Add alert history
  - [ ] Alert timeline
  - [ ] Alert analytics
  - [ ] False positive tracking
  - [ ] Alert acknowledgment

##### Alert Event Handler (`internal/services/alert/probe_alert_handler.go`) - NEW FILE
- [ ] Create alert evaluation engine
  - [ ] Rule condition evaluation
  - [ ] Alert state machine
  - [ ] Flapping detection
  - [ ] Alert deduplication
- [ ] Implement notification dispatcher
  - [ ] Channel selection logic
  - [ ] Retry with backoff
  - [ ] Notification templating
  - [ ] Rate limiting

---

## 🔧 Technical Debt & Improvements

### Code Quality
- [ ] Add unit tests for WebSSH components
  - [ ] Session manager tests
  - [ ] Terminal manager tests
  - [ ] WebSocket handler tests
- [ ] Add unit tests for probe components
  - [ ] Probe executor tests
  - [ ] Scheduler tests
  - [ ] Alert handler tests
- [ ] Add integration tests
  - [ ] End-to-end WebSSH flow
  - [ ] Probe execution flow
  - [ ] Alert triggering flow
- [ ] Improve error handling
  - [ ] Consistent error types
  - [ ] Error wrapping with context
  - [ ] User-friendly error messages

### Performance Optimization
- [ ] Optimize WebSocket connection pooling
- [ ] Implement probe result data compression
- [ ] Add caching for frequently accessed data
- [ ] Optimize database queries with indexes
- [ ] Implement connection multiplexing for Agent

### Security Enhancements
- [ ] Add rate limiting for WebSSH connections
- [ ] Implement session recording encryption
- [ ] Add probe result data encryption at rest
- [ ] Implement API request signing for Agent
- [ ] Add security headers for all endpoints

### Documentation
- [ ] Write WebSSH user guide
- [ ] Create service probe configuration guide
- [ ] Document Agent installation process
- [ ] Add API documentation for new endpoints
- [ ] Create troubleshooting guide

---

## 📈 Success Metrics

### WebSSH Terminal
- [ ] Support 100+ concurrent sessions
- [ ] Connection latency < 100ms
- [ ] 99.9% connection stability
- [ ] Support for all common terminal operations
- [ ] Zero security vulnerabilities

### Service Probe
- [ ] Support 1000+ monitored services
- [ ] Probe execution accuracy > 99.99%
- [ ] Alert false positive rate < 1%
- [ ] Data retention for 90 days
- [ ] Query response time < 500ms

---

## 🚀 Quick Start Tasks (Do First!)

### Today's Focus
1. [ ] Complete WebSocket message protocol design
2. [ ] Implement basic HTTP probe functionality
3. [ ] Create frontend page skeletons

### Tomorrow's Focus
1. [ ] Implement Agent SSH proxy basic structure
2. [ ] Add probe result storage optimization
3. [ ] Design UI components for service monitor

### This Week's Goals
1. [ ] Working WebSSH prototype
2. [ ] Basic probe execution with HTTP support
3. [ ] Frontend pages with mock data

---

## 📝 Notes

### Dependencies
- xterm.js (already installed)
- React Query for data fetching
- Recharts for visualization
- golang.org/x/crypto/ssh for SSH client
- robfig/cron for probe scheduling

### Known Issues
- WebSocket connection drops on network change
- Agent reconnection sometimes fails
- Probe results table needs indexing

### References
- [xterm.js documentation](https://xtermjs.org/docs/)
- [SSH Protocol RFC](https://tools.ietf.org/html/rfc4254)
- [Prometheus metric types](https://prometheus.io/docs/concepts/metric_types/)

---

## 🎯 Definition of Done

### For Each Feature
- [ ] Code implemented and reviewed
- [ ] Unit tests written and passing
- [ ] Integration tests passing
- [ ] Documentation updated
- [ ] UI/UX reviewed and approved
- [ ] Performance benchmarks met
- [ ] Security review completed
- [ ] Deployed to staging environment

---

*Last updated: 2025-10-07 by Claude*
*Next review: After Phase 1 completion*