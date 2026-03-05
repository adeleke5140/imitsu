# imitsu

A team secret manager with AES-256-GCM encryption, role-based access control, team sharing, and audit logging.

## Quick Start

```bash
npm install
npm run build
```

### Start the server

```bash
IMITSU_MASTER_KEY="your-secret-master-key" IMITSU_JWT_SECRET="your-jwt-secret" npm start
```

The server runs on `http://localhost:3100` by default. Set `IMITSU_PORT` to change it.

### First-time setup

```bash
# First user becomes admin
imitsu register -e you@team.com -n "Your Name"
imitsu login you@team.com
```

## CLI Reference

### Auth

```bash
imitsu register -e <email> -n <name>     # Create account (prompts for password)
imitsu login <email>                      # Login (saves token locally)
imitsu logout                             # Clear session
imitsu whoami                             # Show current user
```

### Secrets

```bash
imitsu set <name> [value]                 # Create or update (prompts if no value given)
imitsu set <name> -c <category>           # Set with a category tag
imitsu set <name> [value] -t <team>       # Create and share with a team
imitsu get <name>                         # Print secret value (pipeable)
imitsu ls                                 # List all accessible secrets
imitsu rm <name>                          # Delete a secret
```

### Sharing

```bash
# Share with a specific user
imitsu share <name> -u <email>                     # read access (default)
imitsu share <name> -u <email> -p write             # write access
imitsu share <name> -u <email> -p admin             # full access

# Share with a team (all members get access)
imitsu share-team <secret> <team>
imitsu share-team <secret> <team> -p write
```

### Teams

```bash
imitsu team create <name>                 # Create a team
imitsu team ls                            # List your teams
imitsu team add <team> <email>            # Add a member
imitsu team members <team>                # List members
```

### Bulk Import / Export

```bash
# Import a .env file
imitsu import .env
imitsu import .env -c database            # Tag with category
imitsu import .env -t backend             # Import and share with a team

# Export secrets as .env
imitsu export                             # Print to stdout
imitsu export .env.local                  # Write to file
imitsu export -c database                 # Filter by category
```

### Admin

```bash
imitsu users                              # List all users
imitsu audit                              # View audit log
imitsu audit -l 50                        # Last 50 entries
```

## TUI (itui)

There's also an interactive terminal UI. See [src/tui/README.md](src/tui/README.md) for installation and usage.

```sh
curl -fsSL https://raw.githubusercontent.com/adeleke5140/imitsu/main/install.sh | sh
itui
```

## Typical Team Workflow

```bash
# 1. Admin sets up
imitsu register -e admin@company.com -n "Admin"
imitsu login admin@company.com
imitsu team create backend

# 2. Import your existing .env and share with the team
imitsu import .env.production -t backend -c production

# 3. New developer joins
imitsu register -e dev@company.com -n "New Dev"    # dev runs this
imitsu team add backend dev@company.com             # admin runs this

# 4. New dev immediately has access
imitsu login dev@company.com
imitsu ls
imitsu export .env.local
```

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `IMITSU_PORT` | `3100` | Server port |
| `IMITSU_MASTER_KEY` | — | Encryption master key. **Required in production.** |
| `IMITSU_JWT_SECRET` | — | JWT signing secret. **Required in production.** |
| `IMITSU_DB_PATH` | `./imitsu.db` | SQLite database path |

## Security

- **Encryption**: AES-256-GCM with per-secret salts and IVs, keys derived via HKDF-SHA512
- **Passwords**: Argon2id (65MB memory, 3 iterations)
- **Auth**: JWT tokens, 8-hour expiry
- **Access control**: Owner/admin/team/per-user permissions
- **Audit trail**: Every read, write, share, and delete is logged
- **Rate limiting**: 100 requests/minute per IP

### Production Checklist

- [ ] Set strong random values for `IMITSU_MASTER_KEY` and `IMITSU_JWT_SECRET`
- [ ] Run behind HTTPS (nginx, Caddy, or cloud load balancer)
- [ ] Back up `imitsu.db` regularly
- [ ] Restrict network access to the server

## Project Structure

```
src/
├── cli/                    # CLI client
│   ├── client.ts           # HTTP client + local config
│   └── vault.ts            # Command definitions
├── server/                 # API server
│   ├── index.ts            # Express app + rate limiting
│   ├── auth/auth.ts        # Registration, login, JWT
│   ├── crypto/encryption.ts # AES-256-GCM
│   ├── db/schema.ts        # SQLite schema
│   ├── db/audit.ts         # Audit logging
│   ├── middleware/          # Auth guards
│   └── routes/             # API endpoints
└── tui/                    # Interactive terminal UI (Go)
    ├── main.go             # Entry point
    ├── api/client.go       # API client
    └── ui/                 # Bubble Tea views
```

## License

ISC
