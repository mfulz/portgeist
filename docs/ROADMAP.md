# 📍 Portgeist Roadmap

## 🔹 Version 0.2.x – Access Control & Remote Config

### ✅ Auth & Roles (Fine-grained Access Control)
| Feature                                 | Status     | Description |
|----------------------------------------|------------|-------------|
| Authentication profiles (`admin`, `manage`, `view`) | 🟦 Planned | Each control user is assigned a role with specific permissions |
| Role-based access enforcement per command | 🟦 Planned | Role evaluation enforced in control server and CLI logic |
| IPC protocol extension for permission model | 🟦 Planned | Roles and scopes integrated into command handling metadata |

---

### ✅ Remote Configuration Management (via `geistctl`)
| Feature                                 | Status     | Description |
|----------------------------------------|------------|-------------|
| Manage Hosts (add/remove/update)       | 🟦 Planned | Commands like `geistctl config host add ...` |
| Manage Proxies (add/remove/default/setactive) | 🟦 Planned | Remote reconfiguration of proxies and bindings |
| Manage Logins (accounts/credentials)   | 🟦 Planned | Token/user configuration remotely controlled |
| Manage Controls (socket/tcp bindings)  | 🟦 Planned | Add, remove or edit control interfaces dynamically |

---

## 🔹 Future Versions

### 🧠 Stability / Observability
- Daemon status/health reporting
- Persistent proxy autostart states
- JSON-based structured logging

### 🧩 Extensibility & Plugins
- Backend plugin registry (dynamic loading)
- Role-aware audit log with user action tracking
- Norn-style capability abstraction layer

