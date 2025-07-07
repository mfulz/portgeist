# 🌀 Portgeist

**Portgeist** is a modular, CLI- and daemon-based proxy controller written in Go.  
It provides secure control over SSH-based proxy tunnels, remote command launching,  
and flexible control interfaces – with fully configurable authentication and logging.

---

## ✨ Features

- 🔐 Secure IPC interface (UNIX socket or TCP with token auth)
- 🚀 Launch subprocesses with `torsocks`, `proxychains`, or IP route enforcement
- 🔁 Dynamically switch active proxy routes per binding
- 🧩 Flexible backend support (e.g., SSH, VPN, chains)
- 📁 Modular config structure: `~/.portgeist/{geistd,geistctl,launch}.yaml`
- 📊 Centralized structured logging with level control and file output
- 🧠 Extensible architecture (backends, control modes, launch logic)

---

## 📁 Directory Structure

```bash
~/.portgeist/
├── geistd/
│   └── geistd.yaml         # Daemon config
├── geistctl/
│   ├── config.yaml         # CLI config
│   └── launch.yaml         # Launcher profiles
└── logs/
    └── portgeist.log       # Centralized log file (if enabled)
```

---

## 🚦 Roadmap

See [docs/ROADMAP.md](docs/ROADMAP.md) for upcoming features and milestones.

---

## 🛠 Development

- Build: `make build` (or `go build ./cmd/geistd`, `./cmd/geistctl`)
- Run daemon: `go run ./cmd/geistd/main.go`
- Use CLI: `go run ./cmd/geistctl/main.go ...`

---

## 📜 License

MIT
