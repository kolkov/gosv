### Go Supervisor (GSV) - Project Digest

#### 📌 Core Information
1. **GitHub Repo**: [github.com/kolkov/gosv](https://github.com/kolkov/gosv)
2. **Language**: Go 1.22+
3. **Status**: Beta (core features complete)
4. **Documentation**: English (code comments, docs, issues)
5. **Key Features**:
   - Process management with auto-restart
   - YAML configuration
   - TUI interface (tview)
   - CLI control (start/stop/restart/status)
   - Foreground mode for single process
   - Cross-platform (Windows/Linux)
   - Graceful shutdown
   - Config reload (SIGHUP)
   - Log buffering and display

#### 🚀 Project Plan
| Stage             | Status     | Next Steps                     |
|-------------------|------------|--------------------------------|
| Core Supervisor   | ✅ Complete| Stabilize APIs                 |
| TUI Interface     | ✅ Complete| Add process control in TUI     |
| CLI Commands      | ✅ Complete| Improve help and docs          |
| HTTP API          | ⬜ Not Started| Design REST/gRPC endpoints   |
| WASM Plugins      | ⬜ Not Started| Define plugin interface      |
| Cluster Mode      | ⬜ Not Started| Research Raft consensus      |
| Monitoring        | ⬜ Not Started| Add Prometheus metrics       |
| Authentication    | ⬜ Not Started| Secure API access            |

#### 🏗️ Project Structure
```bash
gosv/
├── cmd/
│   └── gosv/           # Main CLI entrypoint
│       └── main.go
├── internal/
│   ├── config/         # YAML config loader
│   │   └── config.go
│   ├── process/        # Process management
│   │   └── manager.go
│   └── supervisor/     # Core logic and TUI
│       └── supervisor.go
├── gosv.yaml           # Sample config
├── go.mod
└── go.sum
```

#### 🔗 Key Dependencies
1. **TUI**: [github.com/rivo/tview](https://github.com/rivo/tview)
2. **Terminal**: [github.com/gdamore/tcell](https://github.com/gdamore/tcell)
3. **Colors**: [github.com/fatih/color](https://github.com/fatih/color)
4. **YAML**: [gopkg.in/yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3)

#### 💻 Development Notes
1. **Coding Standards**:
   - English comments only
   - `go fmt` enforced
   - Semantic versioning
2. **Build/Run**:
```bash
# Build
go build -o gosv.exe ./cmd/gosv

# Run (normal mode)
./gosv -c gosv.yaml

# Run (TUI mode)
./gosv -c gosv.yaml -tui

# CLI commands
./gosv -c gosv.yaml -status
./gosv -c gosv.yaml -start=web-server
./gosv -c gosv.yaml -stop=web-server
./gosv -c gosv.yaml -restart=web-server
./gosv -c gosv.yaml -run=web-server
./gosv -c gosv.yaml -list
```

#### 🚨 Known Issues
1. TUI may have rendering artifacts on some terminals
2. Windows signal handling limitations
3. No PID file support

#### 🔜 Immediate Next Steps
1. Implement HTTP API (REST)
2. Add Prometheus metrics endpoint
3. Create basic auth for API
4. Write integration tests

---

### 📄 Project Manifesto
We're building **gosv** as a modern, cloud-native process supervisor that combines:
- Simplicity of classic supervisors
- Scalability of container orchestrators
- Extensibility through plugins

**Design Principles**:
1. **One Binary**: Zero dependencies deployment
2. **DevOps Friendly**: Metrics, APIs, cloud integration
3. **Secure By Default**: Minimal attack surface
4. **Batteries Included**: Built-in useful features

---

### ▶️ Как продолжить разработку в новом окне

1. **Клонируйте репозиторий**:
   ```bash
   git clone https://github.com/kolkov/gosv
   cd gosv
   ```

2. **Обновите зависимости**:
   ```bash
   go mod tidy
   ```

3. **Запустите проект**:
   ```bash
   go run ./cmd/gosv -c gosv.yaml -tui
   ```

4. **Основные ветки**:
   - `main`: Стабильная версия
   - `dev`: Активная разработка
   ```bash
   git checkout dev
   ```

5. **Рабочий процесс**:
   ```bash
   # Создайте новую ветку для фичи
   git checkout -b feature/http-api
   
   # Реализуйте изменения
   # ...
   
   # Протестируйте
   go test ./...
   go build -o gosv.exe ./cmd/gosv
   
   # Зафиксируйте изменения (на английском)
   git commit -m "feat: implement HTTP API endpoints"
   
   # Запушьте ветку
   git push origin feature/http-api
   
   # Создайте Pull Request в GitHub
   ```

6. **Конвенции**:
   - Коммиты: `feat:`, `fix:`, `docs:`, `refactor:`
   - Пакеты: только английские названия
   - Комментарии: на английском, поясняющие сложные места

7. **Тестирование**:
   ```bash
   # Запуск всех тестов
   go test ./...
   
   # С код покрытием
   go test -coverprofile=coverage.out ./...
   go tool cover -html=coverage.out
   ```

8. **Документация**:
   - Обновляйте README.md для новых фич
   - Комментируйте публичные методы в стиле GoDoc
   - Вехи проекта в GitHub Projects

---

### 🔜 Что делать дальше
1. Реализовать HTTP API в новом пакете `internal/api/`
2. Добавить эндпоинты:
   - `GET /status` - JSON статус процессов
   - `POST /process/{name}/start` - запуск процесса
   - `POST /process/{name}/stop` - остановка процесса
3. Интегрировать Prometheus в `internal/metrics/`
4. Написать интеграционные тесты

Let's continue building! ➡️ [Open New Terminal Session](command:workbench.action.terminal.new)