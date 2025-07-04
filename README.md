
<p align="center">
  <img src="assets/logo_portgeist.png" alt="PORTGEIST logo" width="200"/>
</p>

<h1 align="center">PORTGEIST</h1>
<p align="center"><em>Daemon-controlled dynamic proxy orchestration over SSH, VPN, or more.</em></p>

---

## 🧠 Overview

**PORTGEIST** is a modular proxy control system designed for hackers, developers, and network operators who need on-demand, remote-controlled SOCKS proxy endpoints over various backends like SSH, VPN tunnels, or future plugin support.

At its core, PORTGEIST is composed of two main components:

- `geistd`: The daemon that maintains active proxy endpoints and handles backend connectivity logic.
- `geistctl`: A CLI interface to list, start, stop, and manage proxy definitions and remote hosts.

---

## 🔩 Key Features

- **Configurable backends**: Define login credentials, remote hosts, and proxy mappings in a clean YAML config.
- **Dynamic fallback**: Each proxy can specify a default host and a fallback chain for failover.
- **Daemon lifecycle**: `geistd` manages persistent SOCKS5 tunnels and can auto-start proxies on launch.
- **Remote control**: `geistctl` allows local or remote command execution via future control interfaces.
- **Extensible**: Backend abstraction allows future support for VPN, WireGuard, or even TOR.

---

## 📦 Example Use Case

Configure your browser (or curl, or system) to use:
```
SOCKS5 127.0.0.1:8888
```

Then control it:
```bash
geistctl proxy -p proxy1 start
geistctl proxy -p proxy1 setactive -h zurich
geistctl proxy -p proxy1 status
```

---

## 📁 Project Layout

```bash
portgeist/
├── cmd/
│   ├── geistd/
│   └── geistctl/
├── internal/
│   ├── config/
│   ├── proxy/
│   └── backend/
├── assets/
│   └── logo_portgeist.png
└── README.md
```

---

## 💡 Status

🛠️ Under active development. Initial prototype targets SSH-based proxies, but backend is abstracted for broader tunneling logic in the future.

---

## 🔮 Name Origin

> **Portgeist** – a mischievous little ghost that haunts your localhost ports and tunnels traffic from the shadows.

---

## 📜 License

MIT – free to haunt your ports.
