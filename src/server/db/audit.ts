import { v4 as uuidv4 } from "uuid";
import { getDb } from "./schema.js";

export interface AuditEntry {
  id: string;
  user_id: string | null;
  action: string;
  resource_type: string;
  resource_id: string | null;
  details: string | null;
  ip_address: string | null;
  created_at: string;
}

export function logAudit(
  userId: string | null,
  action: string,
  resourceType: string,
  resourceId: string | null = null,
  details: string | null = null,
  ipAddress: string | null = null
): void {
  const db = getDb();
  db.prepare(
    `INSERT INTO audit_log (id, user_id, action, resource_type, resource_id, details, ip_address)
     VALUES (?, ?, ?, ?, ?, ?, ?)`
  ).run(uuidv4(), userId, action, resourceType, resourceId, details, ipAddress);
}

export function getAuditLogs(
  limit = 50,
  offset = 0,
  userId?: string,
  resourceType?: string
): AuditEntry[] {
  const db = getDb();
  let query = "SELECT * FROM audit_log WHERE 1=1";
  const params: unknown[] = [];

  if (userId) {
    query += " AND user_id = ?";
    params.push(userId);
  }
  if (resourceType) {
    query += " AND resource_type = ?";
    params.push(resourceType);
  }

  query += " ORDER BY created_at DESC LIMIT ? OFFSET ?";
  params.push(limit, offset);

  return db.prepare(query).all(...params) as AuditEntry[];
}
