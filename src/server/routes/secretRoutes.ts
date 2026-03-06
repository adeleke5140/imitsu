import { Router, type Request, type Response } from "express";
import { v4 as uuidv4 } from "uuid";
import { encrypt, decrypt } from "../crypto/encryption.js";
import { authenticate } from "../middleware/authMiddleware.js";
import { logAudit } from "../db/audit.js";
import { getDb } from "../db/schema.js";

const router = Router();

const MASTER_KEY = process.env.IMITSU_MASTER_KEY || "default-master-key-change-in-production";

interface SecretRow {
  id: string;
  name: string;
  encrypted_value: string;
  iv: string;
  auth_tag: string;
  salt: string;
  category: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  expires_at: string | null;
  version: number;
}

// Check if user can access a secret
function canAccess(userId: string, secretId: string, requiredPermission: "read" | "write" | "admin"): boolean {
  const db = getDb();

  // Owner always has access
  const secret = db.prepare("SELECT created_by FROM secrets WHERE id = ?").get(secretId) as { created_by: string } | undefined;
  if (secret?.created_by === userId) return true;

  // Check user role - admins have full access
  const user = db.prepare("SELECT role FROM users WHERE id = ?").get(userId) as { role: string } | undefined;
  if (user?.role === "admin") return true;

  const levels: Record<string, number> = { read: 1, write: 2, admin: 3 };

  // Check explicit user access grants
  const access = db.prepare("SELECT permission FROM secret_access WHERE secret_id = ? AND user_id = ?").get(secretId, userId) as { permission: string } | undefined;
  if (access && (levels[access.permission] ?? 0) >= (levels[requiredPermission] ?? 0)) return true;

  // Check team-based access
  const teamAccess = db.prepare(
    `SELECT sta.permission FROM secret_team_access sta
     JOIN team_members tm ON tm.team_id = sta.team_id
     WHERE sta.secret_id = ? AND tm.user_id = ?
     ORDER BY sta.permission DESC LIMIT 1`
  ).get(secretId, userId) as { permission: string } | undefined;
  if (teamAccess && (levels[teamAccess.permission] ?? 0) >= (levels[requiredPermission] ?? 0)) return true;

  return false;
}

// Create a secret
router.post("/", authenticate, (req: Request, res: Response) => {
  try {
    const { name, value, category, expires_at } = req.body as {
      name?: string;
      value?: string;
      category?: string;
      expires_at?: string;
    };

    if (!name) {
      res.status(400).json({ error: "name is required" });
      return;
    }

    const userId = req.user!.userId;
    const db = getDb();
    const id = uuidv4();
    const { encrypted, iv, authTag, salt } = encrypt(value ?? "", MASTER_KEY);

    db.prepare(
      `INSERT INTO secrets (id, name, encrypted_value, iv, auth_tag, salt, category, created_by, expires_at)
       VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
    ).run(id, name, encrypted, iv, authTag, salt, category ?? "general", userId, expires_at ?? null);

    // Store initial version
    db.prepare(
      `INSERT INTO secret_versions (id, secret_id, encrypted_value, iv, auth_tag, salt, version, created_by)
       VALUES (?, ?, ?, ?, ?, ?, 1, ?)`
    ).run(uuidv4(), id, encrypted, iv, authTag, salt, userId);

    logAudit(userId, "secret.create", "secret", id, `Created secret: ${name}`, req.ip ?? null);

    res.status(201).json({
      secret: { id, name, category: category ?? "general", created_by: userId, version: 1 },
    });
  } catch (err) {
    const message = err instanceof Error ? err.message : "Failed to create secret";
    res.status(500).json({ error: message });
  }
});

// List secrets the user can access
router.get("/", authenticate, (req: Request, res: Response) => {
  const userId = req.user!.userId;
  const userRole = req.user!.role;
  const db = getDb();

  let secrets;
  if (userRole === "admin") {
    secrets = db.prepare(
      `SELECT id, name, category, created_by, created_at, updated_at, expires_at, version
       FROM secrets ORDER BY updated_at DESC`
    ).all();
  } else {
    secrets = db.prepare(
      `SELECT DISTINCT s.id, s.name, s.category, s.created_by, s.created_at, s.updated_at, s.expires_at, s.version
       FROM secrets s
       LEFT JOIN secret_access sa ON s.id = sa.secret_id AND sa.user_id = ?
       LEFT JOIN secret_team_access sta ON s.id = sta.secret_id
       LEFT JOIN team_members tm ON tm.team_id = sta.team_id AND tm.user_id = ?
       WHERE s.created_by = ? OR sa.user_id IS NOT NULL OR tm.user_id IS NOT NULL
       ORDER BY s.updated_at DESC`
    ).all(userId, userId, userId);
  }

  res.json({ secrets });
});

// Bulk import secrets (from key=value pairs)
router.post("/import", authenticate, (req: Request, res: Response) => {
  try {
    const userId = req.user!.userId;
    const { secrets: entries, category, team_id: teamId } = req.body as {
      secrets: { name: string; value: string }[];
      category?: string;
      team_id?: string;
    };

    if (!entries || !Array.isArray(entries) || entries.length === 0) {
      res.status(400).json({ error: "secrets array is required" });
      return;
    }

    const db = getDb();
    const created: string[] = [];
    const updated: string[] = [];

    const insertSecret = db.transaction(() => {
      for (const entry of entries) {
        if (!entry.name) continue;

        // Find existing secret by name that the user owns, has write access to,
        // or belongs to the target team (if importing with a team)
        const existing = db.prepare(
          `SELECT DISTINCT s.id, s.version FROM secrets s
           LEFT JOIN secret_access sa ON s.id = sa.secret_id AND sa.user_id = ?
           LEFT JOIN secret_team_access sta ON s.id = sta.secret_id
           LEFT JOIN team_members tm ON tm.team_id = sta.team_id AND tm.user_id = ?
           WHERE s.name = ? AND (
             s.created_by = ?
             OR (sa.user_id IS NOT NULL AND sa.permission IN ('write', 'admin'))
             OR (tm.user_id IS NOT NULL AND sta.permission IN ('write', 'admin'))
             ${teamId ? "OR sta.team_id = ?" : ""}
           )
           LIMIT 1`
        ).get(...(teamId
          ? [userId, userId, entry.name, userId, teamId]
          : [userId, userId, entry.name, userId]
        )) as { id: string; version: number } | undefined;

        if (existing) {
          const { encrypted, iv, authTag, salt } = encrypt(entry.value, MASTER_KEY);
          const newVersion = existing.version + 1;
          db.prepare(
            `UPDATE secrets SET encrypted_value = ?, iv = ?, auth_tag = ?, salt = ?, version = ?,
             category = COALESCE(?, category), updated_at = datetime('now') WHERE id = ?`
          ).run(encrypted, iv, authTag, salt, newVersion, category, existing.id);

          db.prepare(
            `INSERT INTO secret_versions (id, secret_id, encrypted_value, iv, auth_tag, salt, version, created_by)
             VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
          ).run(uuidv4(), existing.id, encrypted, iv, authTag, salt, newVersion, userId);

          updated.push(entry.name);

          if (teamId) {
            db.prepare(
              `INSERT INTO secret_team_access (id, secret_id, team_id, permission, granted_by)
               VALUES (?, ?, ?, 'write', ?)
               ON CONFLICT(secret_id, team_id) DO UPDATE SET permission = 'write'`
            ).run(uuidv4(), existing.id, teamId, userId);
          }
        } else {
          const id = uuidv4();
          const { encrypted, iv, authTag, salt } = encrypt(entry.value, MASTER_KEY);

          db.prepare(
            `INSERT INTO secrets (id, name, encrypted_value, iv, auth_tag, salt, category, created_by)
             VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
          ).run(id, entry.name, encrypted, iv, authTag, salt, category ?? "general", userId);

          db.prepare(
            `INSERT INTO secret_versions (id, secret_id, encrypted_value, iv, auth_tag, salt, version, created_by)
             VALUES (?, ?, ?, ?, ?, ?, 1, ?)`
          ).run(uuidv4(), id, encrypted, iv, authTag, salt, userId);

          created.push(entry.name);

          if (teamId) {
            db.prepare(
              `INSERT INTO secret_team_access (id, secret_id, team_id, permission, granted_by)
               VALUES (?, ?, ?, 'write', ?)`
            ).run(uuidv4(), id, teamId, userId);
          }
        }
      }
    });

    insertSecret();

    logAudit(userId, "secret.import", "secret", null,
      `Imported ${created.length} new, ${updated.length} updated secrets`, req.ip ?? null);

    res.status(201).json({ created, updated, total: created.length + updated.length });
  } catch (err) {
    const message = err instanceof Error ? err.message : "Failed to import secrets";
    res.status(500).json({ error: message });
  }
});

// Bulk export secrets as key=value
router.get("/export", authenticate, (req: Request, res: Response) => {
  const userId = req.user!.userId;
  const userRole = req.user!.role;
  const db = getDb();

  let rows: SecretRow[];
  if (userRole === "admin") {
    rows = db.prepare("SELECT * FROM secrets ORDER BY name").all() as SecretRow[];
  } else {
    rows = db.prepare(
      `SELECT DISTINCT s.* FROM secrets s
       LEFT JOIN secret_access sa ON s.id = sa.secret_id AND sa.user_id = ?
       LEFT JOIN secret_team_access sta ON s.id = sta.secret_id
       LEFT JOIN team_members tm ON tm.team_id = sta.team_id AND tm.user_id = ?
       WHERE s.created_by = ? OR sa.user_id IS NOT NULL OR tm.user_id IS NOT NULL
       ORDER BY s.name`
    ).all(userId, userId, userId) as SecretRow[];
  }

  const secrets = rows.map((row) => ({
    name: row.name,
    value: decrypt(row.encrypted_value, row.iv, row.auth_tag, MASTER_KEY, row.salt),
    category: row.category,
  }));

  logAudit(userId, "secret.export", "secret", null,
    `Exported ${secrets.length} secrets`, req.ip ?? null);

  res.json({ secrets });
});

// Get a secret's value (decrypted)
router.get("/:id", authenticate, (req: Request, res: Response) => {
  const userId = req.user!.userId;
  const secretId = req.params.id as string;
  const db = getDb();

  const row = db.prepare("SELECT * FROM secrets WHERE id = ?").get(secretId) as SecretRow | undefined;
  if (!row) {
    res.status(404).json({ error: "Secret not found" });
    return;
  }

  if (!canAccess(userId, secretId, "read")) {
    res.status(403).json({ error: "Access denied" });
    return;
  }

  const value = decrypt(row.encrypted_value, row.iv, row.auth_tag, MASTER_KEY, row.salt);

  logAudit(userId, "secret.read", "secret", secretId, `Read secret: ${row.name}`, req.ip ?? null);

  res.json({
    secret: {
      id: row.id,
      name: row.name,
      value,
      category: row.category,
      created_by: row.created_by,
      version: row.version,
      expires_at: row.expires_at,
      created_at: row.created_at,
      updated_at: row.updated_at,
    },
  });
});

// Update a secret
router.put("/:id", authenticate, (req: Request, res: Response) => {
  try {
    const userId = req.user!.userId;
    const secretId = req.params.id as string;
    const { value, name, category } = req.body as {
      value?: string;
      name?: string;
      category?: string;
    };

    const db = getDb();
    const row = db.prepare("SELECT * FROM secrets WHERE id = ?").get(secretId) as SecretRow | undefined;

    if (!row) {
      res.status(404).json({ error: "Secret not found" });
      return;
    }

    if (!canAccess(userId, secretId, "write")) {
      res.status(403).json({ error: "Access denied" });
      return;
    }

    const newVersion = row.version + 1;

    if (value) {
      const { encrypted, iv, authTag, salt } = encrypt(value, MASTER_KEY);

      db.prepare(
        `UPDATE secrets SET encrypted_value = ?, iv = ?, auth_tag = ?, salt = ?, version = ?,
         name = COALESCE(?, name), category = COALESCE(?, category),
         updated_at = datetime('now') WHERE id = ?`
      ).run(encrypted, iv, authTag, salt, newVersion, name, category, secretId);

      // Store version history
      db.prepare(
        `INSERT INTO secret_versions (id, secret_id, encrypted_value, iv, auth_tag, salt, version, created_by)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
      ).run(uuidv4(), secretId, encrypted, iv, authTag, salt, newVersion, userId);
    } else {
      db.prepare(
        `UPDATE secrets SET name = COALESCE(?, name), category = COALESCE(?, category),
         updated_at = datetime('now') WHERE id = ?`
      ).run(name, category, secretId);
    }

    logAudit(userId, "secret.update", "secret", secretId, `Updated secret: ${row.name}`, req.ip ?? null);

    res.json({ message: "Secret updated", version: value ? newVersion : row.version });
  } catch (err) {
    const message = err instanceof Error ? err.message : "Failed to update secret";
    res.status(500).json({ error: message });
  }
});

// Delete a secret
router.delete("/:id", authenticate, (req: Request, res: Response) => {
  const userId = req.user!.userId;
  const secretId = req.params.id as string;
  const db = getDb();

  const row = db.prepare("SELECT name, created_by FROM secrets WHERE id = ?").get(secretId) as { name: string; created_by: string } | undefined;
  if (!row) {
    res.status(404).json({ error: "Secret not found" });
    return;
  }

  if (!canAccess(userId, secretId, "admin")) {
    res.status(403).json({ error: "Access denied - admin or owner required" });
    return;
  }

  db.prepare("DELETE FROM secrets WHERE id = ?").run(secretId);
  logAudit(userId, "secret.delete", "secret", secretId, `Deleted secret: ${row.name}`, req.ip ?? null);

  res.json({ message: "Secret deleted" });
});

// Share a secret with another user
router.post("/:id/share", authenticate, (req: Request, res: Response) => {
  try {
    const userId = req.user!.userId;
    const secretId = req.params.id as string;
    const { user_id: targetUserId, permission } = req.body as {
      user_id?: string;
      permission?: string;
    };

    if (!targetUserId || !permission) {
      res.status(400).json({ error: "user_id and permission are required" });
      return;
    }

    if (!["read", "write", "admin"].includes(permission)) {
      res.status(400).json({ error: "permission must be read, write, or admin" });
      return;
    }

    const db = getDb();

    const secret = db.prepare("SELECT name FROM secrets WHERE id = ?").get(secretId) as { name: string } | undefined;
    if (!secret) {
      res.status(404).json({ error: "Secret not found" });
      return;
    }

    if (!canAccess(userId, secretId, "admin")) {
      res.status(403).json({ error: "Only owners and admins can share secrets" });
      return;
    }

    const targetUser = db.prepare("SELECT id, email FROM users WHERE id = ?").get(targetUserId) as { id: string; email: string } | undefined;
    if (!targetUser) {
      res.status(404).json({ error: "Target user not found" });
      return;
    }

    db.prepare(
      `INSERT INTO secret_access (id, secret_id, user_id, permission, granted_by)
       VALUES (?, ?, ?, ?, ?)
       ON CONFLICT(secret_id, user_id) DO UPDATE SET permission = excluded.permission`
    ).run(uuidv4(), secretId, targetUserId, permission, userId);

    logAudit(
      userId, "secret.share", "secret", secretId,
      `Shared secret "${secret.name}" with user ${targetUser.email} (${permission})`,
      req.ip ?? null
    );

    res.json({ message: `Secret shared with ${targetUser.email} (${permission})` });
  } catch (err) {
    const message = err instanceof Error ? err.message : "Failed to share secret";
    res.status(500).json({ error: message });
  }
});

// Share a secret with a team
router.post("/:id/share-team", authenticate, (req: Request, res: Response) => {
  try {
    const userId = req.user!.userId;
    const secretId = req.params.id as string;
    const { team_id: teamId, permission } = req.body as {
      team_id?: string;
      permission?: string;
    };

    if (!teamId || !permission) {
      res.status(400).json({ error: "team_id and permission are required" });
      return;
    }

    if (!["read", "write", "admin"].includes(permission)) {
      res.status(400).json({ error: "permission must be read, write, or admin" });
      return;
    }

    const db = getDb();

    const secret = db.prepare("SELECT name FROM secrets WHERE id = ?").get(secretId) as { name: string } | undefined;
    if (!secret) {
      res.status(404).json({ error: "Secret not found" });
      return;
    }

    if (!canAccess(userId, secretId, "admin")) {
      res.status(403).json({ error: "Only owners and admins can share secrets" });
      return;
    }

    const team = db.prepare("SELECT id, name FROM teams WHERE id = ?").get(teamId) as { id: string; name: string } | undefined;
    if (!team) {
      res.status(404).json({ error: "Team not found" });
      return;
    }

    db.prepare(
      `INSERT INTO secret_team_access (id, secret_id, team_id, permission, granted_by)
       VALUES (?, ?, ?, ?, ?)
       ON CONFLICT(secret_id, team_id) DO UPDATE SET permission = excluded.permission`
    ).run(uuidv4(), secretId, teamId, permission, userId);

    logAudit(
      userId, "secret.share_team", "secret", secretId,
      `Shared secret "${secret.name}" with team "${team.name}" (${permission})`,
      req.ip ?? null
    );

    res.json({ message: `Secret shared with team "${team.name}" (${permission})` });
  } catch (err) {
    const message = err instanceof Error ? err.message : "Failed to share secret";
    res.status(500).json({ error: message });
  }
});

// Get version history
router.get("/:id/versions", authenticate, (req: Request, res: Response) => {
  const userId = req.user!.userId;
  const secretId = req.params.id as string;
  const db = getDb();

  if (!canAccess(userId, secretId, "read")) {
    res.status(403).json({ error: "Access denied" });
    return;
  }

  const versions = db.prepare(
    `SELECT id, version, created_by, created_at FROM secret_versions
     WHERE secret_id = ? ORDER BY version DESC`
  ).all(secretId);

  logAudit(userId, "secret.versions", "secret", secretId, null, req.ip ?? null);

  res.json({ versions });
});

// Deduplicate secrets - keeps the newest per name, deletes the rest (admin only)
router.post("/deduplicate", authenticate, (req: Request, res: Response) => {
  try {
    const userRole = req.user!.role;
    const userId = req.user!.userId;

    if (userRole !== "admin") {
      res.status(403).json({ error: "Admin only" });
      return;
    }

    const db = getDb();

    // Find duplicate names and the IDs to delete (keep the most recently updated)
    const dupes = db.prepare(
      `SELECT id, name FROM secrets
       WHERE id NOT IN (
         SELECT id FROM (
           SELECT id, ROW_NUMBER() OVER (PARTITION BY name ORDER BY updated_at DESC, version DESC) as rn
           FROM secrets
         ) WHERE rn = 1
       )`
    ).all() as { id: string; name: string }[];

    if (dupes.length === 0) {
      res.json({ message: "No duplicates found", deleted: 0 });
      return;
    }

    const deleteIds = dupes.map((d) => d.id);

    db.transaction(() => {
      for (const id of deleteIds) {
        db.prepare("DELETE FROM secrets WHERE id = ?").run(id);
      }
    })();

    const names = [...new Set(dupes.map((d) => d.name))];
    logAudit(userId, "secret.deduplicate", "secret", null,
      `Removed ${dupes.length} duplicates for: ${names.join(", ")}`, req.ip ?? null);

    res.json({ message: `Removed ${dupes.length} duplicates`, deleted: dupes.length, names });
  } catch (err) {
    const message = err instanceof Error ? err.message : "Failed to deduplicate";
    res.status(500).json({ error: message });
  }
});

export default router;
