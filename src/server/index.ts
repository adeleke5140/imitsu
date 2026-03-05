import express from "express";
import authRoutes from "./routes/authRoutes.js";
import secretRoutes from "./routes/secretRoutes.js";
import auditRoutes from "./routes/auditRoutes.js";
import teamRoutes from "./routes/teamRoutes.js";
import { getDb } from "./db/schema.js";

const app = express();
const PORT = parseInt(process.env.IMITSU_PORT || "3100");

app.use(express.json());

// Rate limiting (simple in-memory)
const requestCounts = new Map<string, { count: number; resetAt: number }>();
const RATE_LIMIT = 100;
const RATE_WINDOW = 60_000;

app.use((req, res, next) => {
  const key = req.ip ?? "unknown";
  const now = Date.now();
  const entry = requestCounts.get(key);

  if (!entry || now > entry.resetAt) {
    requestCounts.set(key, { count: 1, resetAt: now + RATE_WINDOW });
    next();
    return;
  }

  entry.count++;
  if (entry.count > RATE_LIMIT) {
    res.status(429).json({ error: "Too many requests" });
    return;
  }
  next();
});

// Routes
app.use("/api/auth", authRoutes);
app.use("/api/secrets", secretRoutes);
app.use("/api/audit", auditRoutes);
app.use("/api/teams", teamRoutes);

// Health check
app.get("/health", (_req, res) => {
  res.json({ status: "ok", timestamp: new Date().toISOString() });
});

// Initialize DB on startup
getDb();

app.listen(PORT, () => {
  console.log(`imitsu server running on port ${PORT}`);
  console.log(`Health check: http://localhost:${PORT}/health`);
});

export default app;
