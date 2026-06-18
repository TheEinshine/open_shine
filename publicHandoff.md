# Open Shine — Project Handoff (public)

This document briefs a fresh AI session (or any reader) on the `open_shine` project. All credentials, IPs, hostnames, and usernames have been redacted — replace the `<...>` placeholders with real values from your own environment. Never commit real secrets to the repo.

## What this project is

A small Go web service (`open_shine`) running on an always-on home Ubuntu laptop server. It serves a trivial HTTP endpoint and, in the background, emails a "heartbeat" **status report** on an interval — a dark-themed HTML table of host metrics (CPU, memory, storage, load, uptime), Go-runtime stats, and a "log stack" of recent send history. Email config (recipient, interval, subject, on/off) lives in MySQL; secrets live in environment variables. The repo auto-deploys to the server via a poll-based GitOps loop.

## Infrastructure

- **Repo:** `github.com/TheEinshine/open_shine` (public, GitHub)
- **Go module:** `github.com/TheEinshine/open_shine`, `go 1.26`
- **Dependency:** `github.com/go-sql-driver/mysql v1.10.0` (+ `filippo.io/edwards25519` indirect). `go.sum` must be committed.
- **Server:** Ubuntu 24.04 laptop on the local network (acting as an always-on server).
- **Repo path on server:** `/home/<user>/<workspace>/open_shine`
- **Stack on server:** Tailscale (remote access), SSH, systemd, Air `v1.65.x` (hot-reload), Go, MySQL/MariaDB.

### Deployment loop (GitOps, poll-based)

Develop on a separate machine → `git commit` → `git push` to `main`. On the server, a systemd timer runs every minute:

1. `open-shine-updater.timer` fires (every 1 min) → `open-shine-updater.service` → `~/scripts/update-open-shine.sh`
2. Script does `git fetch origin`, compares local vs remote commit
3. If changed: `git reset --hard origin/main`
4. Air detects file changes → rebuilds → restarts the Go app
5. New version live

`open-shine.service` runs Air, which runs the Go app. Air keeps the last good build running if a new build fails (logs `failed to build, error: exit status 1`).

**Deploy implications:**

- Only `origin/main` deploys. A PR branch will NOT go live until merged (desirable — review before auto-deploy).
- `git reset --hard` every minute wipes any file edited directly on the server. The server is a pure mirror of GitHub.
- Each redeploy restarts the app, re-firing the immediate startup email and resetting the interval clock.
- Shutdown is signal-driven (SIGINT/SIGTERM): `systemctl stop/restart open-shine.service` now drains in-flight HTTP requests (10s deadline) and closes the DB pool before exit, rather than hard-killing. For Air's own per-build restarts to be graceful too, set `send_interrupt = true` (with a `kill_delay`) in `.air.toml`; otherwise Air hard-kills the old process and the drain is skipped (harmless for this workload).
- If dependencies change, run `go get ...` + `go mod tidy` on the dev machine and commit `go.mod` + `go.sum`, or Air's build on the server fails.

## Code layout

```
open_shine/
├── main.go                  # HTTP service on :8080 + background mail loop goroutine
├── mailer/mailer.go         # SMTP send via stdlib net/smtp; multipart text+HTML; creds from env
├── db/db.go                 # MySQL: Open, Migrate, Seed, GetSettings, LogSend, RecentLogs, Close
├── sysstat/                 # Host + runtime metrics (no deps)
│   ├── sysstat.go           #   shared types + Collect()
│   ├── sysstat_linux.go     #   //go:build linux — reads /proc + statfs
│   └── sysstat_other.go     #   //go:build !linux — dev stub (metrics unavailable)
├── report/report.go         # Renders Stats + log history → dark HTML table + text fallback
├── go.mod
└── go.sum
```

**Critical:** Go requires one package per directory. `mailer`, `db`, `sysstat`, and `report` each live in their own subfolder. Having multiple top-level packages' `.go` files flat in the repo root fails to build (`found packages main and mailer`).

**Metrics are dependency-free:** host metrics come straight from the Linux `/proc` filesystem (`/proc/stat`, `/proc/meminfo`, `/proc/uptime`, `/proc/loadavg`) and a `statfs` syscall — no third-party library. Build tags keep it cross-platform: on non-Linux dev machines `sysstat_other.go` reports host metrics as unavailable (runtime stats still work), so `go build` succeeds everywhere while production (Linux) gets the real numbers.

### main.go behavior

- Serves `GET /` → `Shine's Service v4 is running` and `GET /healthz` → `ok` on `:8080`, via an explicit `http.Server` with timeouts (`ReadHeaderTimeout 5s`, `ReadTimeout`/`WriteTimeout 15s`, `IdleTimeout 60s`) so a slow or stalled client can't pin a connection open indefinitely (slow-loris protection).
- **Graceful shutdown:** `main` watches SIGINT/SIGTERM via `signal.NotifyContext`. On signal it calls `srv.Shutdown` (10s deadline) to drain in-flight requests, the mail loop stops at its next checkpoint, the DB pool is closed, and the process exits cleanly (`log.Fatalf` only on a real, unexpected server error — not on the normal `http.ErrServerClosed`).
- `startMailLoop(ctx)` runs as a goroutine: loads SMTP config from env, opens DB, runs `Migrate()` then `Seed()`, then loops: read `mail_settings`, if `enabled && recipient != ""` build + send the report + log to `mail_log`, then wait `interval_mins` (min 1, default 10). Sends immediately on first iteration. The wait is **interruptible** (`select` on `ctx.Done()` vs a timer) — a shutdown signal stops the loop at once instead of blocking for up to a full interval.
- `sendHeartbeat` calls `sysstat.Collect()` (host + runtime metrics) and `store.RecentLogs(8)`, renders both via `report.RenderHTML`/`RenderText`, and sends as one multipart email. The log stack shows **prior** sends — the current send is recorded to `mail_log` only after it completes.
- If env/DB/SMTP is misconfigured it logs `mailer disabled: ...` and the HTTP service keeps running (app does not crash).

### mailer package

- `Config` holds SMTP host/port/user/pass/from, all from env. Recipient is NOT here — it comes from the DB.
- `SendMessage(Message{To, Subject, Text, HTML})` is the main entry: when `HTML` is set it sends `multipart/alternative` (text part first, then HTML); otherwise plain text. `Send(to, subject, body)` is a thin plain-text wrapper. Transport is `smtp.PlainAuth` + `smtp.SendMail` (port 587 STARTTLS); Go's `DotWriter` handles CRLF + dot-stuffing.
- Header values are run through `sanitizeHeader` (strips CR/LF) so a DB-sourced subject/recipient can't inject extra headers.

### db package

- `Open()` — connects via DSN built from `DB_*` env, pings.
- `Migrate()` — runs an ordered slice of idempotent `CREATE TABLE IF NOT EXISTS` statements. Safe on every boot. Add new tables/columns by appending to the `migrations` slice.
- `Seed(defaultRecipient)` — `INSERT IGNORE` the single `mail_settings` row (id=1) with `recipient = defaultRecipient` (passed as `SMTP_USER` from main). Runs once; never overwrites manual changes.
- `GetSettings()`, `LogSend(status, errMsg)`, `Close()` (releases the connection pool — called via `defer` on mail-loop exit / shutdown).
- `RecentLogs(limit)` — returns up to `limit` newest `mail_log` rows (`[]LogEntry`) for the report's log stack.

## Database schema

Database name: `open_shine`.

```sql
-- empty for now; auth deferred
users (
  id INT AUTO_INCREMENT PK,
  name VARCHAR(255) NOT NULL,
  email VARCHAR(255) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NULL,   -- ready for hashing later, no migration needed
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
)

mail_settings (        -- single row, id=1
  id INT PK,
  recipient VARCHAR(255) NOT NULL,
  interval_mins INT DEFAULT 10,
  subject VARCHAR(255) DEFAULT 'Open Shine heartbeat',
  enabled BOOLEAN DEFAULT TRUE
)

mail_log (
  id INT AUTO_INCREMENT PK,
  sent_at DATETIME NOT NULL,
  status VARCHAR(20) NOT NULL,   -- 'ok' | 'error'
  error TEXT
)
```

Change behavior at runtime with SQL (next tick picks it up, no redeploy):

```sql
UPDATE mail_settings SET recipient='<recipient>' WHERE id=1;
UPDATE mail_settings SET interval_mins=30 WHERE id=1;   -- applies after current sleep ends
UPDATE mail_settings SET enabled=FALSE WHERE id=1;      -- pause without stopping the app
SELECT sent_at, status, error FROM mail_log ORDER BY id DESC LIMIT 20;
```

## Environment / secrets

File: `/etc/open-shine.env`, `chmod 600`, owner `root:root`. Loaded into the service via `EnvironmentFile=/etc/open-shine.env` in the `[Service]` block of `open-shine.service`. Editing it requires `systemctl restart open-shine.service` to take effect (env is read at process start).

```
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=<db-user>            # case-sensitive
DB_PASS=<db-password>
DB_NAME=open_shine
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=<gmail-address>
SMTP_PASS=<16-char Gmail App Password, NO spaces>
# SMTP_FROM optional; defaults to SMTP_USER
```

**Secrets policy:** SMTP and DB passwords stay in env, NOT in the DB. If the MySQL instance is reachable beyond localhost (e.g. exposed on the LAN/Tailscale), anything with the DB login could read a secret stored in a table — whereas the env file is root-only `chmod 600`. Keep secrets in env. Use a strong DB password and restrict MySQL's bind address / grants to what you actually need.

## Decisions made

- **SMTP vs service:** Use plain Gmail SMTP (free, ~500 emails/day; 144/day at a 10-min interval is fine). No transactional provider needed because mail goes to the operator's own inbox; deliverability features (SPF/DKIM/reputation) only matter when emailing other people. Host/port/creds are env-driven so switching to Brevo/Mailgun later is config-only, no code change.
- **Config split:** secrets → env; tunable non-secret data (recipient, interval, subject, enabled) → DB.
- **Seeder:** structure (`Migrate`) and default data (`Seed`) are separate methods so schema can grow cleanly. `users` table scaffolded now, auth deferred.
- **Server hardening:** explicit `http.Server` timeouts + signal-driven graceful shutdown, chosen over the stdlib `http.ListenAndServe` defaults (no timeouts, hard exit) because this is a long-running, always-on service. A redeploy/restart drains cleanly and a slow client can't hold a connection open. The mail loop and HTTP server share one cancellation `context` so a single signal coordinates both.

## Gotchas to avoid

1. **Flat file layout fails to build** — packages must be in `mailer/` and `db/` subdirs.
2. **Missing `go.sum`** — run `go mod tidy` and commit it, or Air build fails on the server.
3. **Seed runs once** — `Seed` uses `INSERT IGNORE`, so the `mail_settings` recipient is fixed on first run. If env had a placeholder address when the row was first created, later env fixes won't update it. Correct it with `UPDATE mail_settings SET recipient=... WHERE id=1;`.
4. **Gmail App Password required** — a normal Gmail password fails with `535 5.7.8 BadCredentials`. Enable 2-Step Verification, then generate an App Password (16 chars, spaces removed) at `myaccount.google.com/apppasswords`.
5. **`mysql -p` spacing** — `mysql -u user -p value` treats `value` as the database name. The password must be glued to the flag: `-p<password>`.
6. **Match the right DB account/host** — the app connects over `127.0.0.1` (= `localhost` to MySQL), so the grant must cover `'<user>'@'localhost'`. A network client (e.g. HeidiSQL over LAN) may match a different host like `'<user>'@'%'`.
7. **Interval change latency** — a changed `interval_mins` applies after the current sleep finishes, not instantly.

## Outstanding / next steps

1. **Push** the current `db/db.go` + `main.go` (Migrate/Seed + server hardening: timeouts, `/healthz`, graceful shutdown); let Air rebuild.
2. **Verify schema:** `SHOW TABLES; DESCRIBE users;` → expect `users`, `mail_settings`, `mail_log`.
3. **Set recipient:** `UPDATE mail_settings SET recipient='<real recipient>' WHERE id=1;` then restart and confirm a real email arrives.
4. **Confirm logs:** journal shows `mail loop started` → `heartbeat email sent to <recipient>`; `mail_log` has an `ok` row.

### Likely future work

- Wire up `users` CRUD (insert/read).
- Add auth later: hash into `password_hash` (e.g. bcrypt), login flow.
- `/healthz` exists now (returns `ok` — liveness only). Optional enhancement: a richer endpoint that reads the last `mail_log` row so `curl localhost:8080/healthz` reports last send status (timestamp + ok/error).
- Optional: sender display name / from-address as DB-configurable.
- Harden DB credentials and MySQL bind address / grants.
- Reliability: disable laptop suspend (`sudo systemctl mask sleep.target suspend.target hibernate.target hybrid-sleep.target`); set BIOS "Power On After AC Loss" for auto-recovery after an outage.
- Consider whether email-every-10-min is the right design vs a pull-based uptime monitor (e.g. healthchecks.io / UptimeRobot) that alerts on _missing_ heartbeats.

## Recovery characteristics (by design)

SSH disconnect / closed terminal → survives (systemd). Air crash → systemd restarts. App crash → Air restarts. Reboot → systemd starts everything. Public IP change → Tailscale handles it. Lid closed → continues (suspend must also be disabled — see above). Power outage → needs BIOS "Power On After AC Loss".
