// Light client-side shape check only (26-char Crockford base32) — purely a
// UX nicety to avoid an obviously-malformed request; the server remains
// the authority on validity.
const ULID_RE = /^[0-9A-HJKMNP-TV-Z]{26}$/i;

export function looksLikeUlid(value: string): boolean {
  return ULID_RE.test(value);
}
