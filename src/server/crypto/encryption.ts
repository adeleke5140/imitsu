import crypto from "crypto";

const ALGORITHM = "aes-256-gcm";
const IV_LENGTH = 16;
const KEY_LENGTH = 32;
const SALT_LENGTH = 32;

// Derive an encryption key from the vault master key using HKDF
function deriveKey(masterKey: string, salt: Buffer): Buffer {
  return Buffer.from(crypto.hkdfSync("sha512", masterKey, salt, "imitsu", KEY_LENGTH));
}

export function encrypt(
  plaintext: string,
  masterKey: string
): { encrypted: string; iv: string; authTag: string; salt: string } {
  const salt = crypto.randomBytes(SALT_LENGTH);
  const key = deriveKey(masterKey, salt);
  const iv = crypto.randomBytes(IV_LENGTH);

  const cipher = crypto.createCipheriv(ALGORITHM, key, iv);
  let encrypted = cipher.update(plaintext, "utf8", "base64");
  encrypted += cipher.final("base64");
  const authTag = cipher.getAuthTag();

  return {
    encrypted,
    iv: iv.toString("base64"),
    authTag: authTag.toString("base64"),
    salt: salt.toString("base64"),
  };
}

export function decrypt(
  encrypted: string,
  iv: string,
  authTag: string,
  masterKey: string,
  salt?: string
): string {
  // For backwards compat: if salt is embedded in the encrypted value
  const saltBuf = salt
    ? Buffer.from(salt, "base64")
    : crypto.randomBytes(SALT_LENGTH); // fallback shouldn't happen
  const key = deriveKey(masterKey, saltBuf);
  const decipher = crypto.createDecipheriv(
    ALGORITHM,
    key,
    Buffer.from(iv, "base64")
  );
  decipher.setAuthTag(Buffer.from(authTag, "base64"));

  let decrypted = decipher.update(encrypted, "base64", "utf8");
  decrypted += decipher.final("utf8");
  return decrypted;
}
