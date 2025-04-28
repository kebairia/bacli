# bacli, Backup Manager — PostgreSQL & MongoDB

[![Go Report Card](https://goreportcard.com/badge/github.com/kebairia/backup)](https://goreportcard.com/report/github.com/kebairia/backup)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

---

**bacli** is a lightweight and extensible Go application that automates **backup** and **restore** operations for PostgreSQL and MongoDB databases.
It uses a simple YAML configuration to manage multiple instances, with structured logging and metadata tracking for production-grade reliability.

---

## ✨ Features

- **Automated backups and restores** (`pg_dump`, `pg_restore`, `mongodump`, `mongorestore`)
- **Structured logging** (JSON format)
- **Centralized metadata tracking** (backup duration, size, status)
- **Flexible YAML configuration** (global defaults + per-instance overrides)
- **Robust error handling** with clean recovery from failures

---

## 📂 Project Structure

```plaintext
.
├── bacli                # Compiled binary
├── cmd                  # CLI entrypoints (backup, restore, root commands)
│   ├── backup_cmd.go
│   ├── restore_cmd.go
│   └── root.go
├── configs              # Configuration files
│   └── config.yaml
├── internal             # Internal application packages
│   ├── backup           # Backup and restore logic (Postgres, MongoDB, MySQL)
│   ├── config           # YAML configuration loader
│   ├── logger           # Structured logger setup
│   └── operations       # Orchestration of backup and restore workflows
├── go.mod               # Go modules file
├── go.sum               # Go modules checksum file
├── LICENSE              # Project license
├── main.go              # Application entry point
├── Makefile             # Automation commands
└── README.md            # Project documentation
```

---

## 🚀 Quickstart

### 1. Define your configuration

```yaml
backup:
  output_dir: "./backups"
  timestamp_format: "2006-01-02_15-04-05"
defaults:
  postgres:
    host: "localhost"
    port: "5432"
    method: "custom"
  mongodb:
    host: "localhost"
    port: "27017"
    method: "archive"

postgres_instances:
  - username: "user1"
    password: "pass1"
    database: "db1"
    port: "5432"
    method: "plain"
  - username: "user2"
    password: "pass2"
    database: "db2"
    port: "5433"

mongodb_instances:
  - username: "root1"
    password: "secret1"
    database: "testdb1"
    port: "27017"
```

### 2. Run backup

```bash
./bacli backup --config ./configs/config.yaml
```

### 3. Run restore

```bash
./bacli restore --source metadata.json
```

Backup metadata will be saved automatically to `metadata.json`.

---

## 📜 Example Logs

### Successful Backup

```json
{"level":"info","time":"2025-04-28T16:30:00Z","msg":"backup started","database":"testdb1","engine":"mongodb","path":"./backups/mongodb/2025-04-28_testdb1"}
{"level":"info","time":"2025-04-28T16:31:00Z","msg":"backup completed","database":"testdb1","engine":"mongodb","path":"./backups/mongodb/2025-04-28_testdb1","duration":"15s"}
```

### Successful Restore

```json
{"level":"info","time":"2025-04-28T17:00:00Z","msg":"restore started","database":"testdb1","engine":"mongodb","source":"./backups/mongodb/2025-04-28_testdb1"}
{"level":"info","time":"2025-04-28T17:01:00Z","msg":"restore completed","database":"testdb1","engine":"mongodb","source":"./backups/mongodb/2025-04-28_testdb1","duration":"25s"}
```

### Backup Error

```json
{
  "level": "error",
  "time": "2025-04-28T16:32:00Z",
  "msg": "backup failed",
  "database": "testdb1",
  "engine": "mongodb",
  "path": "./backups/mongodb/2025-04-28_testdb1",
  "error": "mongodump: connection refused"
}
```

### Restore Error

```json
{
  "level": "error",
  "time": "2025-04-28T17:02:00Z",
  "msg": "restore failed",
  "database": "testdb1",
  "engine": "mongodb",
  "source": "./backups/mongodb/2025-04-28_testdb1",
  "error": "mongorestore: permission denied"
}
```

---

## 🗂 Example Metadata File (metadata.json)

```json
{
  "run_at": "2025-04-28T16:30:00Z",
  "backups": [
    {
      "name": "db10",
      "path": "None",
      "success": false,
      "error": "pg_dump failed: exit status 1",
      "started_at": "2025-04-28T18:08:31.122614594Z",
      "duration_ms": 8880553,
      "size_bytes": 0
    },
    {
      "name": "db1",
      "path": "./backups/postgres/2025-04-28_16-30-00-db1.dump",
      "success": true,
      "error": false,
      "started_at": "2025-04-28T16:32:00Z",
      "duration_ms": 18000,
      "size_bytes": 4096000
    }
  ]
}
```

---

## 🛠 Requirements

- Go 1.20+
- `psql`, `pg_dump`, `pg_restore` (PostgreSQL client tools)
- `mongodump`, `mongorestore` (MongoDB client tools)

---

## 🛡 License

This project is licensed under the [Apache 2.0 License](LICENSE).

---

## 📣 Contribution

Pull requests are welcome! For major changes, please open an issue first to discuss what you would like to change.

---

> _Simple, reliable backup automation for PostgreSQL and MongoDB._
