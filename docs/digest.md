### 🚀 Go Supervisor (gosv) - Project Digest

#### 🌐 Core Information
1. **GitHub**: [github.com/kolkov/gosv](https://github.com/kolkov/gosv)
2. **Language**: Go 1.22+
3. **Status**: Beta (core features complete)
4. **Documentation**: English (code comments, docs, issues)
5. **Key Features**:
   - Process management with auto-restart
   - YAML configuration
   - TUI interface (tview)
   - CLI control (start/stop/restart/status)
   - gRPC API for remote management
   - Foreground mode for single process
   - Cross-platform (Windows/Linux)
   - Graceful shutdown
   - Config reload (SIGHUP)
   - Log buffering and display

---

#### 📂 Project Structure
```bash
gosv/
├── api/
│   ├── gosv/           # Generated gRPC code
│   │   ├── supervisor.pb.go
│   │   └── supervisor_grpc.pb.go
│   └── supervisor.proto # Protobuf definition
├── cmd/
│   ├── client/         # gRPC client CLI
│   │   └── main.go
│   └── gosv/           # Main CLI entrypoint
│       └── main.go
├── internal/
│   ├── api/            # gRPC server implementation
│   │   └── server.go
│   ├── config/         # YAML config loader
│   │   └── config.go
│   ├── process/        # Process management
│   │   └── manager.go
│   ├── service/        # Service interfaces
│   │   ├── adapter.go
│   │   └── service.go
│   └── supervisor/     # Core logic and TUI
│       └── supervisor.go
├── gosv.yaml           # Sample config
├── go.mod
├── go.sum
└── README.md           # Project documentation
```

---

#### ✅ Completed Milestones
1. **Core Supervisor Engine**
   - Process lifecycle management
   - Auto-restart with exponential backoff
   - Status monitoring
2. **Configuration System**
   - YAML config parsing
   - Auto-config creation
3. **TUI Interface**
   - Real-time process monitoring
   - Log display
   - Keyboard controls
4. **gRPC API**
   - Start/Stop/Restart processes
   - Status reporting
   - Reflection support
5. **CLI Client**
   - Remote process control
   - Status checks

---

#### 🔧 Development Quickstart
```bash
# 1. Clone repository
git clone https://github.com/kolkov/gosv
cd gosv

# 2. Install dependencies
go mod tidy

# 3. Generate gRPC code (Windows)
third_party\protoc_gen.cmd

# 4. Build binaries
go build -o gosv.exe ./cmd/gosv
go build -o client.exe ./cmd/client

# 5. Run supervisor
./gosv -c gosv.yaml -grpc-port 50051 -debug

# 6. Control processes
./client localhost:50051 status
./client localhost:50051 start web-server
```

---

#### 🚀 Next Steps
1. **HTTP Gateway**  
   Add RESTful interface via gRPC-Gateway
   ```bash
   mkdir internal/api/gateway
   ```
2. **Authentication**  
   Implement TLS and JWT for gRPC API
3. **Prometheus Metrics**  
   Add monitoring endpoint
4. **Cluster Mode**  
   Research Raft consensus for HA
5. **Windows Service**  
   Create service wrapper for production

---

#### 📚 Documentation Standards
1. **Code Comments**:
   ```go
   // StartProcess initiates a managed process
   // Errors: process not found, already running
   func (s *Supervisor) StartProcess(name string) error
   ```
2. **Git Commits**:
   ```bash
   git commit -m "feat: add process health checks"
   git commit -m "fix: resolve status race condition"
   ```
3. **PR Guidelines**:
   - English descriptions
   - Link related issues
   - Update README.md for new features

---

#### ⚙️ Workflow for New Features
```bash
# Create feature branch
git checkout -b feat/http-gateway

# Implement changes
# ...

# Run tests
go test ./...

# Update documentation
nano README.md

# Commit and push
git commit -m "feat: implement HTTP gateway"
git push origin feat/http-gateway

# Create PR on GitHub
```

---

### ▶️ Start Developing Now
[![Open in GitHub Codespaces](https://img.shields.io/badge/Open_in_Codespaces-181717?style=for-the-badge&logo=github)](https://github.com/kolkov/gosv)  
[Open New Terminal Session](command:workbench.action.terminal.new)