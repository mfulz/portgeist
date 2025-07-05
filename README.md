
<p align="center">
  <img src="assets/logo_portgeist.png" alt="PORTGEIST logo" width="200"/>
</p>

<h1 align="center">PORTGEIST</h1>
<p align="center"><em>Daemon-controlled dynamic proxy orchestration over SSH, VPN, or more.</em></p>

---

## 🧠 Overview

**PORTGEIST** is a modular proxy control system designed for hackers, developers, and network operators who need on-demand, remote-controlled SOCKS proxy endpoints over various backends like SSH, VPN tunnels, or plugin modules.

At its core, PORTGEIST consists of two primary components:

- `geistd`: The daemon that maintains active proxy endpoints and manages backend logic and orchestration.
- `geistctl`: A CLI interface to list, start, stop, inspect and control proxy endpoints locally or remotely.

---

## 🔩 Key Features

- **Multi-backend support** (currently: `ssh_exec`, extensible)
- **Dynamic proxy fallback**: Automatic failover across defined hosts.
- **Flexible config**:
  - Proxies
  - Hosts
  - Daemon control interfaces (UNIX/TCP)
  - Per-backend settings and overrides
- **Per-proxy authentication**: Token-based control per user/proxy.
- **Remote daemon support**: Connect and authenticate against multiple `geistd` instances.
- **CLI flags for daemon override** (no need for config entries)
- **Modular backend abstraction**
- **Backend-specific runtime configuration** (e.g., additional SSH options)
- **Full JSON protocol interface**

---

## 🧪 Example Usage

Start and manage a proxy:

```bash
geistctl proxy start -p pp
geistctl proxy setactive -p pp -o zurich
geistctl proxy status -p pp
geistctl proxy info -p pp
```

Using a specific remote daemon:

```bash
geistctl proxy info -p pp -d server1 -u admin
```

Or manually:

```bash
geistctl proxy info -p pp --socket /tmp/alt.sock --token mytoken
```

---

## ⚙️ Configuration Overview

### 📂 `~/.portgeist/config.yaml`

```yaml
users:
  admin:
    token: "adminsecret"
  noob:
    token: "noobtoken"

daemons:
  local:
    socket: /tmp/portgeist.sock
  server1:
    tcp: 127.0.0.1:7142
```

---

### 📂 `config.yaml` (Daemon)

```yaml
control:
  logins:
    admin:
      token: "adminsecret"
    noob:
      token: "noobtoken"
  instances:
    - name: local
      mode: unix
      listen: /tmp/portgeist.sock
      enabled: true
      auth:
        enabled: false
    - name: remote
      mode: tcp
      listen: 127.0.0.1:7142
      enabled: true
      auth:
        enabled: true
        allowed:
          - admin
          - noob

proxies:
  pp:
    default: losangeles
    autostart: false
    allowed_controls: [admin]
    fallback:
      - duesseldorf
      - zurich

hosts:
  losangeles:
    address: losangeles.proxyhost.example.com
    port: 22
    login: pp
    backend: ssh_exec
  duesseldorf:
    address: duesseldorf.proxyhost.example.com
    port: 22
    login: pp
    backend: ssh_exec
    config:
      additional_flags:
        - "-o"
        - "StrictHostKeyChecking=no"

backends:
  ssh_exec:
    additional_flags:
      - "-o"
      - "ExitOnForwardFailure=yes"
```

---

## 🧩 Backend Configuration

Each backend may expose its own configuration fields.
For `ssh_exec`:

```yaml
backends:
  ssh_exec:
    additional_flags:
      - "-o"
      - "ExitOnForwardFailure=yes"
```

You can override backend config per host:

```yaml
hosts:
  zurich:
    address: ...
    backend: ssh_exec
    config:
      additional_flags:
        - "-o"
        - "Compression=yes"
```

---

## 📦 Project Layout

```
portgeist/
├── cmd/
│   ├── geistd/      # Daemon entrypoint
│   └── geistctl/    # CLI interface
├── internal/
│   ├── config/      # Config handling
│   ├── backend/     # Backend implementations
│   ├── proxy/       # Proxy logic
│   └── control/     # Control interfaces (unix/tcp)
├── interfaces/      # Backend interfaces
├── protocol/        # Protocol definitions
├── dispatch/        # Command dispatcher
├── assets/
│   └── logo_portgeist.png
└── README.md
```

---

## 🧙 Name Origin

> **Portgeist** – a mischievous little ghost that haunts your localhost ports and tunnels traffic from the shadows.

---

## 📜 License

MIT – free to haunt your ports.
