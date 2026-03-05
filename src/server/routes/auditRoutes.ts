import { Router, type Request, type Response } from "express";
import { authenticate, requireAdmin } from "../middleware/authMiddleware.js";
import { getAuditLogs } from "../db/audit.js";

const router = Router();

router.get("/", authenticate, requireAdmin, (req: Request, res: Response) => {
  const limit = parseInt(req.query.limit as string) || 50;
  const offset = parseInt(req.query.offset as string) || 0;
  const userId = req.query.user_id as string | undefined;
  const resourceType = req.query.resource_type as string | undefined;

  const logs = getAuditLogs(Math.min(limit, 200), offset, userId, resourceType);
  res.json({ logs, limit, offset });
});

export default router;
