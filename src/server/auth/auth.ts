import argon2 from "argon2";
import jwt from "jsonwebtoken";
import { v4 as uuidv4 } from "uuid";
import { getDb } from "../db/schema.js";

const JWT_SECRET = process.env.IMITSU_JWT_SECRET || "change-me-in-production";
const JWT_EXPIRY = "8h";

export interface User {
  id: string;
  email: string;
  name: string;
  role: "admin" | "member";
  created_at: string;
}

export interface JwtPayload {
  userId: string;
  email: string;
  role: string;
}

export async function hashPassword(password: string): Promise<string> {
  return argon2.hash(password, {
    type: argon2.argon2id,
    memoryCost: 65536,
    timeCost: 3,
    parallelism: 4,
  });
}

export async function verifyPassword(hash: string, password: string): Promise<boolean> {
  return argon2.verify(hash, password);
}

export function generateToken(user: User): string {
  const payload: JwtPayload = {
    userId: user.id,
    email: user.email,
    role: user.role,
  };
  return jwt.sign(payload, JWT_SECRET, { expiresIn: JWT_EXPIRY });
}

export function verifyToken(token: string): JwtPayload {
  return jwt.verify(token, JWT_SECRET) as JwtPayload;
}

export async function registerUser(
  email: string,
  name: string,
  password: string,
  role: "admin" | "member" = "member"
): Promise<User> {
  const db = getDb();
  const existing = db.prepare("SELECT id FROM users WHERE email = ?").get(email);
  if (existing) {
    throw new Error("User with this email already exists");
  }

  const id = uuidv4();
  const passwordHash = await hashPassword(password);

  db.prepare(
    "INSERT INTO users (id, email, name, password_hash, role) VALUES (?, ?, ?, ?, ?)"
  ).run(id, email, name, passwordHash, role);

  return { id, email, name, role, created_at: new Date().toISOString() };
}

export async function loginUser(
  email: string,
  password: string
): Promise<{ user: User; token: string }> {
  const db = getDb();
  const row = db
    .prepare("SELECT id, email, name, password_hash, role, created_at FROM users WHERE email = ?")
    .get(email) as (User & { password_hash: string }) | undefined;

  if (!row) {
    throw new Error("Invalid email or password");
  }

  const valid = await verifyPassword(row.password_hash, password);
  if (!valid) {
    throw new Error("Invalid email or password");
  }

  const user: User = {
    id: row.id,
    email: row.email,
    name: row.name,
    role: row.role,
    created_at: row.created_at,
  };

  return { user, token: generateToken(user) };
}
