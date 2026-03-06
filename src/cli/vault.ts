#!/usr/bin/env node

import { Command } from "commander";
import chalk from "chalk";
import readline from "readline";
import fs from "fs";
import path from "path";
import { api, setAuth, clearAuth, getConfig, setServer } from "./client.js";

const program = new Command();

function readPassword(prompt: string): Promise<string> {
  const rl = readline.createInterface({ input: process.stdin, output: process.stderr });
  return new Promise((resolve) => {
    // Disable echo for password input
    process.stderr.write(prompt);
    const stdin = process.stdin;
    const wasRaw = stdin.isRaw;
    if (stdin.isTTY) stdin.setRawMode(true);

    let password = "";
    const onData = (ch: Buffer) => {
      const c = ch.toString();
      if (c === "\n" || c === "\r") {
        stdin.removeListener("data", onData);
        if (stdin.isTTY) stdin.setRawMode(wasRaw ?? false);
        rl.close();
        process.stderr.write("\n");
        resolve(password);
      } else if (c === "\u0003") {
        process.exit(1);
      } else if (c === "\u007f" || c === "\b") {
        password = password.slice(0, -1);
      } else {
        password += c;
      }
    };
    stdin.on("data", onData);
  });
}

program
  .name("imitsu")
  .description("imitsu - Team secret manager")
  .version("0.0.2");

// Configure server URL
program
  .command("server <url>")
  .description("Set the vault server URL")
  .action((url: string) => {
    setServer(url);
    console.log(chalk.green(`Server set to ${url}`));
  });

// Register
program
  .command("register")
  .description("Create a new account")
  .requiredOption("-e, --email <email>", "Your email")
  .requiredOption("-n, --name <name>", "Your name")
  .action(async (opts: { email: string; name: string }) => {
    try {
      const password = await readPassword("Password (min 12 chars): ");
      const confirm = await readPassword("Confirm password: ");
      if (password !== confirm) {
        console.log(chalk.red("Passwords do not match"));
        process.exit(1);
      }

      const data = await api<{ user: { id: string; email: string; role: string } }>("/api/auth/register", {
        method: "POST",
        body: { email: opts.email, name: opts.name, password },
        auth: false,
      });

      console.log(chalk.green(`Registered! Role: ${data.user.role}`));
      if (data.user.role === "admin") {
        console.log(chalk.yellow("You are the first user — you've been made admin."));
      }
    } catch (err) {
      console.error(chalk.red((err as Error).message));
      process.exit(1);
    }
  });

// Login
program
  .command("login <email>")
  .description("Login to the vault")
  .action(async (email: string) => {
    try {
      const password = await readPassword("Password: ");
      const data = await api<{ user: { email: string; role: string }; token: string }>("/api/auth/login", {
        method: "POST",
        body: { email, password },
        auth: false,
      });

      setAuth(data.token, email);
      console.log(chalk.green(`Logged in as ${data.user.email} (${data.user.role})`));
    } catch (err) {
      console.error(chalk.red((err as Error).message));
      process.exit(1);
    }
  });

// Logout
program
  .command("logout")
  .description("Clear saved credentials")
  .action(() => {
    clearAuth();
    console.log(chalk.green("Logged out"));
  });

// Who am I
program
  .command("whoami")
  .description("Show current user")
  .action(async () => {
    try {
      const config = getConfig();
      if (!config.token) {
        console.log(chalk.yellow("Not logged in. Run: vault login <email>"));
        return;
      }
      const data = await api<{ user: { email: string; name: string; role: string } }>("/api/auth/me");
      console.log(`${data.user.name} <${data.user.email}> (${data.user.role})`);
    } catch (err) {
      console.error(chalk.red((err as Error).message));
      process.exit(1);
    }
  });

// Set a secret
program
  .command("set <name> [value]")
  .description("Create or update a secret")
  .option("-c, --category <category>", "Secret category", "general")
  .option("-t, --team <team>", "Share with team (by name)")
  .action(async (name: string, value: string | undefined, opts: { category: string; team?: string }) => {
    try {
      if (!value) {
        value = await readPassword("Secret value: ");
      }

      // Check if secret exists by name
      const list = await api<{ secrets: { id: string; name: string }[] }>("/api/secrets");
      const existing = list.secrets.find((s) => s.name === name);

      let secretId: string;
      if (existing) {
        await api(`/api/secrets/${existing.id}`, { method: "PUT", body: { value } });
        secretId = existing.id;
        console.log(chalk.green(`Updated: ${name}`));
      } else {
        const created = await api<{ secret: { id: string } }>("/api/secrets", {
          method: "POST",
          body: { name, value, category: opts.category },
        });
        secretId = created.secret.id;
        console.log(chalk.green(`Created: ${name}`));
      }

      if (opts.team) {
        const teams = await api<{ teams: { id: string; name: string }[] }>("/api/teams");
        const team = teams.teams.find((t) => t.name === opts.team);
        if (!team) {
          console.error(chalk.red(`Team not found: ${opts.team}`));
          process.exit(1);
        }
        await api(`/api/secrets/${secretId}/share-team`, {
          method: "POST",
          body: { team_id: team.id, permission: "read" },
        });
        console.log(chalk.dim(`Shared with team "${opts.team}" (read access)`));
      }
    } catch (err) {
      console.error(chalk.red((err as Error).message));
      process.exit(1);
    }
  });

// Get a secret
program
  .command("get <name>")
  .description("Retrieve a secret value")
  .action(async (name: string) => {
    try {
      const list = await api<{ secrets: { id: string; name: string }[] }>("/api/secrets");
      const secret = list.secrets.find((s) => s.name === name);
      if (!secret) {
        console.error(chalk.red(`Secret not found: ${name}`));
        process.exit(1);
      }

      const data = await api<{ secret: { value: string } }>(`/api/secrets/${secret.id}`);
      // Output just the value so it can be piped
      process.stdout.write(data.secret.value);
      if (process.stdout.isTTY) process.stdout.write("\n");
    } catch (err) {
      console.error(chalk.red((err as Error).message));
      process.exit(1);
    }
  });

// List secrets
program
  .command("list")
  .alias("ls")
  .description("List all accessible secrets")
  .action(async () => {
    try {
      const data = await api<{
        secrets: { id: string; name: string; category: string; version: number; updated_at: string }[];
      }>("/api/secrets");

      if (data.secrets.length === 0) {
        console.log(chalk.yellow("No secrets found"));
        return;
      }

      console.log(chalk.bold("NAME".padEnd(30) + "CATEGORY".padEnd(15) + "VER".padEnd(6) + "UPDATED"));
      console.log("─".repeat(70));
      for (const s of data.secrets) {
        console.log(
          s.name.padEnd(30) +
            s.category.padEnd(15) +
            `v${s.version}`.padEnd(6) +
            s.updated_at
        );
      }
    } catch (err) {
      console.error(chalk.red((err as Error).message));
      process.exit(1);
    }
  });

// Delete a secret
program
  .command("rm <name>")
  .description("Delete a secret")
  .action(async (name: string) => {
    try {
      const list = await api<{ secrets: { id: string; name: string }[] }>("/api/secrets");
      const secret = list.secrets.find((s) => s.name === name);
      if (!secret) {
        console.error(chalk.red(`Secret not found: ${name}`));
        process.exit(1);
      }

      await api(`/api/secrets/${secret.id}`, { method: "DELETE" });
      console.log(chalk.green(`Deleted: ${name}`));
    } catch (err) {
      console.error(chalk.red((err as Error).message));
      process.exit(1);
    }
  });

// Share a secret
program
  .command("share <name>")
  .description("Share a secret with a teammate")
  .requiredOption("-u, --user <email>", "Teammate's email")
  .option("-p, --permission <level>", "Permission: read, write, or admin", "read")
  .action(async (name: string, opts: { user: string; permission: string }) => {
    try {
      // Find secret
      const list = await api<{ secrets: { id: string; name: string }[] }>("/api/secrets");
      const secret = list.secrets.find((s) => s.name === name);
      if (!secret) {
        console.error(chalk.red(`Secret not found: ${name}`));
        process.exit(1);
      }

      // Find user
      const users = await api<{ users: { id: string; email: string }[] }>("/api/auth/users");
      const target = users.users.find((u) => u.email === opts.user);
      if (!target) {
        console.error(chalk.red(`User not found: ${opts.user}`));
        process.exit(1);
      }

      await api(`/api/secrets/${secret.id}/share`, {
        method: "POST",
        body: { user_id: target.id, permission: opts.permission },
      });

      console.log(chalk.green(`Shared "${name}" with ${opts.user} (${opts.permission})`));
    } catch (err) {
      console.error(chalk.red((err as Error).message));
      process.exit(1);
    }
  });

// List users (admin)
program
  .command("users")
  .description("List all users (admin only)")
  .action(async () => {
    try {
      const data = await api<{
        users: { id: string; email: string; name: string; role: string; created_at: string }[];
      }>("/api/auth/users");

      console.log(chalk.bold("EMAIL".padEnd(30) + "NAME".padEnd(20) + "ROLE".padEnd(10) + "JOINED"));
      console.log("─".repeat(75));
      for (const u of data.users) {
        console.log(u.email.padEnd(30) + u.name.padEnd(20) + u.role.padEnd(10) + u.created_at);
      }
    } catch (err) {
      console.error(chalk.red((err as Error).message));
      process.exit(1);
    }
  });

// Audit logs (admin)
program
  .command("audit")
  .description("View audit logs (admin only)")
  .option("-l, --limit <n>", "Number of entries", "20")
  .action(async (opts: { limit: string }) => {
    try {
      const data = await api<{
        logs: { action: string; resource_type: string; resource_id: string; details: string; created_at: string; user_id: string }[];
      }>(`/api/audit?limit=${opts.limit}`);

      if (data.logs.length === 0) {
        console.log(chalk.yellow("No audit logs"));
        return;
      }

      console.log(chalk.bold("TIME".padEnd(22) + "ACTION".padEnd(20) + "DETAILS"));
      console.log("─".repeat(75));
      for (const log of data.logs) {
        console.log(
          log.created_at.padEnd(22) +
            log.action.padEnd(20) +
            (log.details || "—")
        );
      }
    } catch (err) {
      console.error(chalk.red((err as Error).message));
      process.exit(1);
    }
  });

// === IMPORT / EXPORT ===

function parseEnvFile(content: string): { name: string; value: string }[] {
  const entries: { name: string; value: string }[] = [];
  for (const line of content.split("\n")) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#")) continue;
    const eqIdx = trimmed.indexOf("=");
    if (eqIdx === -1) continue;
    const name = trimmed.slice(0, eqIdx).trim();
    let value = trimmed.slice(eqIdx + 1).trim();
    // Strip surrounding quotes
    if ((value.startsWith('"') && value.endsWith('"')) || (value.startsWith("'") && value.endsWith("'"))) {
      value = value.slice(1, -1);
    }
    if (name && value) entries.push({ name, value });
  }
  return entries;
}

program
  .command("import <file>")
  .description("Bulk import secrets from a .env file")
  .option("-c, --category <category>", "Category for imported secrets", "general")
  .option("-t, --team <team>", "Share with team (by name)")
  .action(async (file: string, opts: { category: string; team?: string }) => {
    try {
      const filePath = path.resolve(file);
      if (!fs.existsSync(filePath)) {
        console.error(chalk.red(`File not found: ${filePath}`));
        process.exit(1);
      }

      const content = fs.readFileSync(filePath, "utf-8");
      const entries = parseEnvFile(content);

      if (entries.length === 0) {
        console.log(chalk.yellow("No secrets found in file"));
        return;
      }

      console.log(chalk.dim(`Parsed ${entries.length} entries from ${path.basename(file)}`));

      let teamId: string | undefined;
      if (opts.team) {
        const teams = await api<{ teams: { id: string; name: string }[] }>("/api/teams");
        const team = teams.teams.find((t) => t.name === opts.team);
        if (!team) {
          console.error(chalk.red(`Team not found: ${opts.team}. Create it first with: vault team create <name>`));
          process.exit(1);
        }
        teamId = team.id;
      }

      const data = await api<{ created: string[]; updated: string[]; total: number }>("/api/secrets/import", {
        method: "POST",
        body: { secrets: entries, category: opts.category, team_id: teamId },
      });

      if (data.created.length > 0) {
        console.log(chalk.green(`Created: ${data.created.join(", ")}`));
      }
      if (data.updated.length > 0) {
        console.log(chalk.yellow(`Updated: ${data.updated.join(", ")}`));
      }
      console.log(chalk.green(`Done. ${data.total} secrets imported.`));
      if (teamId) {
        console.log(chalk.dim(`Shared with team "${opts.team}" (read access)`));
      }
    } catch (err) {
      console.error(chalk.red((err as Error).message));
      process.exit(1);
    }
  });

program
  .command("export [file]")
  .description("Export secrets as .env format")
  .option("-c, --category <category>", "Filter by category")
  .action(async (file: string | undefined, opts: { category?: string }) => {
    try {
      const data = await api<{ secrets: { name: string; value: string; category: string }[] }>("/api/secrets/export");

      let secrets = data.secrets;
      if (opts.category) {
        secrets = secrets.filter((s) => s.category === opts.category);
      }

      if (secrets.length === 0) {
        console.error(chalk.yellow("No secrets to export"));
        return;
      }

      const envContent = secrets.map((s) => `${s.name}=${s.value}`).join("\n") + "\n";

      if (file) {
        const outPath = path.resolve(file);
        fs.writeFileSync(outPath, envContent, { mode: 0o600 });
        console.log(chalk.green(`Exported ${secrets.length} secrets to ${outPath}`));
      } else {
        process.stdout.write(envContent);
      }
    } catch (err) {
      console.error(chalk.red((err as Error).message));
      process.exit(1);
    }
  });

// === TEAM COMMANDS ===

const teamCmd = program
  .command("team")
  .description("Team management commands");

teamCmd
  .command("create <name>")
  .description("Create a new team")
  .action(async (name: string) => {
    try {
      const data = await api<{ team: { id: string; name: string } }>("/api/teams", {
        method: "POST",
        body: { name },
      });
      console.log(chalk.green(`Team created: ${data.team.name}`));
    } catch (err) {
      console.error(chalk.red((err as Error).message));
      process.exit(1);
    }
  });

teamCmd
  .command("list")
  .alias("ls")
  .description("List your teams")
  .action(async () => {
    try {
      const data = await api<{
        teams: { id: string; name: string; member_count: number; created_at: string }[];
      }>("/api/teams");

      if (data.teams.length === 0) {
        console.log(chalk.yellow("No teams. Create one: vault team create <name>"));
        return;
      }

      console.log(chalk.bold("TEAM".padEnd(25) + "MEMBERS".padEnd(10) + "CREATED"));
      console.log("─".repeat(55));
      for (const t of data.teams) {
        console.log(t.name.padEnd(25) + String(t.member_count).padEnd(10) + t.created_at);
      }
    } catch (err) {
      console.error(chalk.red((err as Error).message));
      process.exit(1);
    }
  });

teamCmd
  .command("add <team> <email>")
  .description("Add a user to a team")
  .option("-r, --role <role>", "member or admin", "member")
  .action(async (teamName: string, email: string, opts: { role: string }) => {
    try {
      const teams = await api<{ teams: { id: string; name: string }[] }>("/api/teams");
      const team = teams.teams.find((t) => t.name === teamName);
      if (!team) {
        console.error(chalk.red(`Team not found: ${teamName}`));
        process.exit(1);
      }

      const data = await api<{ message: string }>(`/api/teams/${team.id}/members`, {
        method: "POST",
        body: { email, role: opts.role },
      });

      console.log(chalk.green(data.message));
    } catch (err) {
      console.error(chalk.red((err as Error).message));
      process.exit(1);
    }
  });

teamCmd
  .command("members <team>")
  .description("List team members")
  .action(async (teamName: string) => {
    try {
      const teams = await api<{ teams: { id: string; name: string }[] }>("/api/teams");
      const team = teams.teams.find((t) => t.name === teamName);
      if (!team) {
        console.error(chalk.red(`Team not found: ${teamName}`));
        process.exit(1);
      }

      const data = await api<{
        members: { email: string; name: string; role: string; joined_at: string }[];
      }>(`/api/teams/${team.id}`);

      console.log(chalk.bold(`Team: ${teamName}\n`));
      console.log(chalk.bold("EMAIL".padEnd(30) + "NAME".padEnd(20) + "ROLE".padEnd(10) + "JOINED"));
      console.log("─".repeat(75));
      for (const m of data.members) {
        console.log(m.email.padEnd(30) + m.name.padEnd(20) + m.role.padEnd(10) + m.joined_at);
      }
    } catch (err) {
      console.error(chalk.red((err as Error).message));
      process.exit(1);
    }
  });

// Share with team (update existing share command alternative)
program
  .command("share-team <secretName> <teamName>")
  .description("Share a secret with an entire team")
  .option("-p, --permission <level>", "Permission: read, write, or admin", "read")
  .action(async (secretName: string, teamName: string, opts: { permission: string }) => {
    try {
      const [secretList, teamList] = await Promise.all([
        api<{ secrets: { id: string; name: string }[] }>("/api/secrets"),
        api<{ teams: { id: string; name: string }[] }>("/api/teams"),
      ]);

      const secret = secretList.secrets.find((s) => s.name === secretName);
      if (!secret) {
        console.error(chalk.red(`Secret not found: ${secretName}`));
        process.exit(1);
      }

      const team = teamList.teams.find((t) => t.name === teamName);
      if (!team) {
        console.error(chalk.red(`Team not found: ${teamName}`));
        process.exit(1);
      }

      const data = await api<{ message: string }>(`/api/secrets/${secret.id}/share-team`, {
        method: "POST",
        body: { team_id: team.id, permission: opts.permission },
      });

      console.log(chalk.green(data.message));
    } catch (err) {
      console.error(chalk.red((err as Error).message));
      process.exit(1);
    }
  });

program.parse();
