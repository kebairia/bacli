# Bacli, A Backup Manager — PostgreSQL & MongoDB

[![Go Report Card](https://goreportcard.com/badge/github.com/kebairia/backup)](https://goreportcard.com/report/github.com/kebairia/backup)
[![License: Apache2](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

---

**bacli** is a lightweight and extensible Go application that automates **backup** and **restore** operations for PostgreSQL and MongoDB databases.
It uses a simple YAML configuration to manage multiple instances, with structured logging and metadata tracking for production-grade reliability.

---

## ✨ Features

- **Automated backups and restores** (`psql`, `pg_dump`, `pg_restore`, `mongodump`, `mongorestore`)
- **Structured logging** (JSON format, Traefik-style)
- **Centralized metadata tracking** (backup duration, size, status)
- **Flexible YAML configuration** (global defaults + per-instance overrides)
- **Robust error handling** with clean recovery from failures

---

## 📂 Project Structure

```plaintext
/cmd            # CLI entrypoints (backup, restore commands)
/internal
    /backup     # Database implementations (Postgres, MongoDB)
    /config     # YAML config loading and parsing
    /logger     # Structured logging setup (Zap based)
    /operations # Backup and restore orchestration
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
bacli backup --config ./configs/config.yaml
```

### 3. Run restore

```bash
bacli restore --source metadata.json
```

Backup metadata will be saved automatically to `metadata.json`.

---

## 📜 Logs Example (Structured)

```json
{"level":"info","time":"2025-04-28T16:30:00Z","msg":"backup started","database":"testdb1","engine":"mongodb","path":"./backups/mongodb/2025-04-28_testdb1"}
{"level":"info","time":"2025-04-28T16:31:00Z","msg":"backup completed","database":"testdb1","engine":"mongodb","path":"./backups/mongodb/2025-04-28_testdb1","duration":"15s"}
```

---

## 🛠 Requirements

- Go 1.20+
- `pg_dump`, `pg_restore` (PostgreSQL client tools)
- `mongodump`, `mongorestore` (MongoDB client tools)

---

## 🛡 License

This project is licensed under the [Apache2 License](LICENSE).

---

## 📣 Contribution

Pull requests are welcome! For major changes, please open an issue first to discuss what you would like to change.

---

> _Simple, reliable backup automation for PostgreSQL and MongoDB._
