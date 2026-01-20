# Syncase Silent App

**Syncase Silent App** is a background companion tool for the Syncase ecosystem — a secure file synchronization and cloud backup platform built for teams and organizations with role-based access control.
This lightweight client runs silently in the background, ensuring that files are automatically synced across devices in real-time without interrupting the user’s workflow.

---

## Features

* **Automatic File Syncing**
  Seamlessly uploads and downloads files from your assigned folders in the background.

* **Hierarchical Role-Based Access**
  Syncs only folders and files permitted by your assigned role (Admin, Manager, User, etc.) as defined on the Syncase Server.

* **Secure Connection**
  All communications are encrypted using HTTPS/TLS. Authentication tokens are validated before every sync cycle.

* **Offline-First Architecture**
  Changes are queued locally and synced automatically when internet connectivity is restored.

* **Cross-Platform Ready**
  Built in Go, using the Walk GUI for Windows and headless (silent) mode for background execution.

* **Intelligent Conflict Handling**
  Automatically detects file version conflicts and handles them safely with timestamped backups.

---

## Architecture Overview

```
+---------------------+
|   Syncase Server    |
|  (PostgreSQL, API)  |
+---------▲-----------+
          |
      Secure HTTPS
          |
+---------▼-----------+
|  Syncase Silent App |
|  (Local Agent)      |
+---------------------+
|  Role Validation    |
|  File Watcher       |
|  Background Sync    |
|  Local Cache Store  |
+---------------------+
```

The Silent App connects to the Syncase API to:

1. Authenticate the user with email & password.
2. Fetch their allowed folder paths from the server.
3. Start the file watcher and sync engine.
4. Continuously monitor file changes and sync them securely.

---

## Installation

Installation instructions will be provided once public distribution is available.

---

## Configuration

Edit the configuration file located at:

```
C:\Users<username>\AppData\Roaming\SyncaseSilent\config.json
```

Example:

```json
{
  "server_url": "https://api.syncase.app",
  "email": "user@example.com",
  "token": "your_generated_access_token",
  "sync_interval": 60,
  "log_level": "info"
}
```

| Key             | Description                                   |
| --------------- | --------------------------------------------- |
| `server_url`    | The Syncase API base URL                      |
| `email`         | User email for authentication                 |
| `token`         | JWT access token (auto-generated after login) |
| `sync_interval` | Sync interval in seconds                      |
| `log_level`     | Logging verbosity (info, debug, error)        |

---

## Developer Setup

### Prerequisites

* Go 1.21+
* PostgreSQL (for server-side testing)
* Git

### Run Locally

```bash
git clone https://github.com/<your-org>/syncase-silent-app.git
cd syncase-silent-app
go mod tidy
go run main.go
```

### Build Binary

```bash
go build -o SyncaseSilentApp.exe
```

---

## Tech Stack

* **Language:** Go (Golang)
* **GUI Framework:** Walk (Windows)
* **Database (Server-side):** PostgreSQL
* **File Watching:** fsnotify
* **Encryption:** TLS + AES256
* **Role Management:** Hierarchical Role-Based Access Control (RBAC)

---

## Security & Privacy

* All credentials are securely stored and encrypted locally.
* Files are transmitted over HTTPS using end-to-end encryption.
* Admins have complete control over user access and folder permissions.
* No user activity is shared with third parties.

---

## Future Plans

* Cross-platform builds for macOS and Linux.
* GUI dashboard for admin monitoring.
* Real-time notifications for sync events.
* Integration with cloud storage providers (AWS S3, Google Drive, OneDrive).

---
