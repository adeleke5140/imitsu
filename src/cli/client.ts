import fs from "fs";
import path from "path";
import os from "os";

const CONFIG_DIR = path.join(os.homedir(), ".imitsu");
const CONFIG_FILE = path.join(CONFIG_DIR, "config.json");

interface Config {
  serverUrl: string;
  token: string | null;
  email: string | null;
}

function loadConfig(): Config {
  const defaults: Config = { serverUrl: "http://localhost:3100", token: null, email: null };
  if (!fs.existsSync(CONFIG_FILE)) return defaults;
  try {
    return { ...defaults, ...JSON.parse(fs.readFileSync(CONFIG_FILE, "utf-8")) };
  } catch {
    return defaults;
  }
}

function saveConfig(config: Config): void {
  if (!fs.existsSync(CONFIG_DIR)) {
    fs.mkdirSync(CONFIG_DIR, { recursive: true, mode: 0o700 });
  }
  fs.writeFileSync(CONFIG_FILE, JSON.stringify(config, null, 2), { mode: 0o600 });
}

export function getConfig(): Config {
  return loadConfig();
}

export function setServer(url: string): void {
  const config = loadConfig();
  config.serverUrl = url.replace(/\/$/, "");
  saveConfig(config);
}

export function setAuth(token: string, email: string): void {
  const config = loadConfig();
  config.token = token;
  config.email = email;
  saveConfig(config);
}

export function clearAuth(): void {
  const config = loadConfig();
  config.token = null;
  config.email = null;
  saveConfig(config);
}

interface ApiOptions {
  method?: string;
  body?: unknown;
  auth?: boolean;
}

export async function api<T = unknown>(endpoint: string, options: ApiOptions = {}): Promise<T> {
  const config = loadConfig();
  const { method = "GET", body, auth = true } = options;

  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (auth && config.token) {
    headers["Authorization"] = `Bearer ${config.token}`;
  }

  const res = await fetch(`${config.serverUrl}${endpoint}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : null,
  });

  const data = await res.json() as T & { error?: string };

  if (!res.ok) {
    throw new Error((data as { error?: string }).error || `Request failed: ${res.status}`);
  }

  return data;
}
