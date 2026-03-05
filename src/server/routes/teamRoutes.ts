import { Router, type Request, type Response } from "express";
import { v4 as uuidv4 } from "uuid";
import { authenticate } from "../middleware/authMiddleware.js";
import { logAudit } from "../db/audit.js";
import { getDb } from "../db/schema.js";

const router = Router();

// Create a team
router.post("/", authenticate, (req: Request, res: Response) => {
  try {
    const userId = req.user!.userId;
    const { name } = req.body as { name?: string };

    if (!name) {
      res.status(400).json({ error: "name is required" });
      return;
    }

    const db = getDb();
    const id = uuidv4();

    db.prepare("INSERT INTO teams (id, name, created_by) VALUES (?, ?, ?)").run(id, name, userId);

    // Creator is auto-added as team admin
    db.prepare(
      "INSERT INTO team_members (id, team_id, user_id, role, added_by) VALUES (?, ?, ?, 'admin', ?)"
    ).run(uuidv4(), id, userId, userId);

    logAudit(userId, "team.create", "team", id, `Created team: ${name}`, req.ip ?? null);

    res.status(201).json({ team: { id, name, created_by: userId } });
  } catch (err) {
    const message = err instanceof Error ? err.message : "Failed to create team";
    res.status(400).json({ error: message });
  }
});

// List teams the user belongs to
router.get("/", authenticate, (req: Request, res: Response) => {
  const userId = req.user!.userId;
  const userRole = req.user!.role;
  const db = getDb();

  let teams;
  if (userRole === "admin") {
    teams = db.prepare(
      `SELECT t.*, (SELECT COUNT(*) FROM team_members WHERE team_id = t.id) as member_count
       FROM teams t ORDER BY t.name`
    ).all();
  } else {
    teams = db.prepare(
      `SELECT t.*, (SELECT COUNT(*) FROM team_members WHERE team_id = t.id) as member_count
       FROM teams t
       JOIN team_members tm ON tm.team_id = t.id AND tm.user_id = ?
       ORDER BY t.name`
    ).all(userId);
  }

  res.json({ teams });
});

// Get team details with members
router.get("/:id", authenticate, (req: Request, res: Response) => {
  const teamId = req.params.id as string;
  const db = getDb();

  const team = db.prepare("SELECT * FROM teams WHERE id = ?").get(teamId) as { id: string; name: string } | undefined;
  if (!team) {
    res.status(404).json({ error: "Team not found" });
    return;
  }

  const members = db.prepare(
    `SELECT u.id, u.email, u.name, tm.role, tm.created_at as joined_at
     FROM team_members tm
     JOIN users u ON u.id = tm.user_id
     WHERE tm.team_id = ?
     ORDER BY tm.created_at`
  ).all(teamId);

  res.json({ team, members });
});

// Add a member to a team
router.post("/:id/members", authenticate, (req: Request, res: Response) => {
  try {
    const userId = req.user!.userId;
    const teamId = req.params.id as string;
    const { email, role } = req.body as { email?: string; role?: string };

    if (!email) {
      res.status(400).json({ error: "email is required" });
      return;
    }

    const db = getDb();

    // Check requester is team admin or server admin
    const requesterMembership = db.prepare(
      "SELECT role FROM team_members WHERE team_id = ? AND user_id = ?"
    ).get(teamId, userId) as { role: string } | undefined;

    const requesterUser = db.prepare("SELECT role FROM users WHERE id = ?").get(userId) as { role: string } | undefined;

    if (requesterMembership?.role !== "admin" && requesterUser?.role !== "admin") {
      res.status(403).json({ error: "Only team admins can add members" });
      return;
    }

    const targetUser = db.prepare("SELECT id, email FROM users WHERE email = ?").get(email) as { id: string; email: string } | undefined;
    if (!targetUser) {
      res.status(404).json({ error: `User not found: ${email}` });
      return;
    }

    const memberRole = role === "admin" ? "admin" : "member";

    db.prepare(
      `INSERT INTO team_members (id, team_id, user_id, role, added_by)
       VALUES (?, ?, ?, ?, ?)
       ON CONFLICT(team_id, user_id) DO UPDATE SET role = excluded.role`
    ).run(uuidv4(), teamId, targetUser.id, memberRole, userId);

    const team = db.prepare("SELECT name FROM teams WHERE id = ?").get(teamId) as { name: string };

    logAudit(userId, "team.add_member", "team", teamId,
      `Added ${targetUser.email} to team "${team.name}" as ${memberRole}`, req.ip ?? null);

    res.json({ message: `Added ${targetUser.email} to team as ${memberRole}` });
  } catch (err) {
    const message = err instanceof Error ? err.message : "Failed to add member";
    res.status(500).json({ error: message });
  }
});

// Remove a member from a team
router.delete("/:id/members/:userId", authenticate, (req: Request, res: Response) => {
  const requesterId = req.user!.userId;
  const teamId = req.params.id as string;
  const targetUserId = req.params.userId as string;
  const db = getDb();

  const requesterMembership = db.prepare(
    "SELECT role FROM team_members WHERE team_id = ? AND user_id = ?"
  ).get(teamId, requesterId) as { role: string } | undefined;

  const requesterUser = db.prepare("SELECT role FROM users WHERE id = ?").get(requesterId) as { role: string } | undefined;

  if (requesterMembership?.role !== "admin" && requesterUser?.role !== "admin") {
    res.status(403).json({ error: "Only team admins can remove members" });
    return;
  }

  db.prepare("DELETE FROM team_members WHERE team_id = ? AND user_id = ?").run(teamId, targetUserId);

  logAudit(requesterId, "team.remove_member", "team", teamId,
    `Removed user ${targetUserId} from team`, req.ip ?? null);

  res.json({ message: "Member removed" });
});

export default router;
