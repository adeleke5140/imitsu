import { Router, type Request, type Response } from "express";
import { registerUser, loginUser } from "../auth/auth.js";
import { logAudit } from "../db/audit.js";
import { authenticate, requireAdmin } from "../middleware/authMiddleware.js";
import { getDb } from "../db/schema.js";

const router = Router();

// Register - first user becomes admin
router.post("/register", async (req: Request, res: Response) => {
  try {
    const { email, name, password } = req.body as {
      email?: string;
      name?: string;
      password?: string;
    };

    if (!email || !name || !password) {
      res.status(400).json({ error: "email, name, and password are required" });
      return;
    }

    if (password.length < 12) {
      res.status(400).json({ error: "Password must be at least 12 characters" });
      return;
    }

    const db = getDb();
    const userCount = (db.prepare("SELECT COUNT(*) as count FROM users").get() as { count: number }).count;
    const role = userCount === 0 ? "admin" : "member";

    const user = await registerUser(email, name, password, role);
    logAudit(user.id, "user.register", "user", user.id, null, req.ip ?? null);

    res.status(201).json({ user });
  } catch (err) {
    const message = err instanceof Error ? err.message : "Registration failed";
    res.status(400).json({ error: message });
  }
});

router.post("/login", async (req: Request, res: Response) => {
  try {
    const { email, password } = req.body as { email?: string; password?: string };

    if (!email || !password) {
      res.status(400).json({ error: "email and password are required" });
      return;
    }

    const result = await loginUser(email, password);
    logAudit(result.user.id, "user.login", "user", result.user.id, null, req.ip ?? null);

    res.json({ user: result.user, token: result.token });
  } catch (err) {
    const message = err instanceof Error ? err.message : "Login failed";
    res.status(401).json({ error: message });
  }
});

router.get("/me", authenticate, (req: Request, res: Response) => {
  const db = getDb();
  const user = db
    .prepare("SELECT id, email, name, role, created_at FROM users WHERE id = ?")
    .get(req.user!.userId);
  res.json({ user });
});

// List users (admin only)
router.get("/users", authenticate, requireAdmin, (_req: Request, res: Response) => {
  const db = getDb();
  const users = db
    .prepare("SELECT id, email, name, role, created_at FROM users ORDER BY created_at")
    .all();
  res.json({ users });
});

export default router;
