// Types mirror docs/api.md. Unknown response fields are ignored.

export type Visibility = 'public' | 'private';
export type ImageFormat = 'jpeg' | 'png' | 'gif' | 'webp' | 'avif';
export type ImageState = 'active' | 'trashed';
export type Scope = 'upload' | 'read' | 'manage';

export interface Links {
  url: string;
  markdown: string;
  html: string;
  bbcode: string;
}
export type LinkFormat = keyof Links;

export type ImageSource =
  | { kind: 'web' }
  | { kind: 'token'; tokenId: number; tokenName: string }
  | { kind: 'cli' };

export interface Image {
  id: string;
  url: string;
  filename: string;
  mime: string;
  ext: string;
  size: number;
  width: number;
  height: number;
  frames: number;
  sha256: string;
  visibility: Visibility;
  state: ImageState;
  hasGps: boolean;
  hasThumb: boolean;
  broken: boolean;
  duplicate: boolean;
  source: ImageSource;
  tags: string[];
  createdAt: string;
  trashedAt: string | null;
  links: Links;
}

export interface ImagePage {
  items: Image[];
  nextCursor: string | null;
}

export type ImageSort = 'new' | 'old' | 'large' | 'small' | 'name';

export interface ImageQuery {
  state?: ImageState;
  q?: string;
  /** Any-of. */
  format?: ImageFormat[];
  visibility?: Visibility;
  /** All-of. */
  tag?: string[];
  /** web, cli or 	oken:<id>; any-of. */
  source?: string[];
  /** RFC 3339: created_at >= since, created_at < until. */
  since?: string;
  until?: string;
  sort?: ImageSort;
  limit?: number;
  cursor?: string;
}

export interface ImagePatch {
  visibility?: Visibility;
  filename?: string;
  tags?: string[];
}

export interface FacetSource {
  key: string;
  kind: ImageSource['kind'];
  tokenId?: number;
  tokenName?: string;
  count: number;
}

export interface Facets {
  total: number;
  bytes: number;
  formats: Record<ImageFormat, number>;
  visibility: Record<Visibility, number>;
  sources: FacetSource[];
  tags: { tag: string; count: number }[];
}

export type BatchAction = 'trash' | 'restore' | 'purge' | 'visibility';
export interface BatchResult {
  id: string;
  ok: boolean;
  code?: string;
}

export interface Token {
  id: number;
  name: string;
  prefix: string;
  scopes: Scope[];
  createdAt: string;
  expiresAt: string | null;
  lastUsedAt: string | null;
  revokedAt: string | null;
}

export interface TokenCreate {
  name: string;
  scopes: Scope[];
  expiresInDays: number | null;
}

export interface Settings {
  maxUploadBytes: number;
  maxPixels: number;
  allowedFormats: ImageFormat[];
  dedupe: boolean;
  defaultVisibility: Visibility;
  trashRetentionDays: number;
  uploadsPerMinute: number;
  sessionIdleMinutes: number;
  sessionAbsoluteHours: number;
  // read-only
  publicUrl: string;
  imageUrl: string;
  hardMaxUploadBytes: number;
}

export type WritableSettings = Omit<Settings, 'publicUrl' | 'imageUrl' | 'hardMaxUploadBytes'>;

export interface SessionInfo {
  authenticated: boolean;
  setupRequired: boolean;
  user: { name: string } | null;
  limits: {
    maxUploadBytes: number;
    allowedFormats: ImageFormat[];
    defaultVisibility: Visibility;
  } | null;
  version: string;
}

export interface StatusWarning {
  code: string;
  message: string;
}

export interface Status {
  version: string;
  commit: string;
  goVersion: string;
  sqliteVersion: string;
  startedAt: string;
  uptimeSeconds: number;
  images: { active: number; trashed: number; bytes: number; broken: number };
  database: { bytes: number; walBytes: number };
  disk: { freeBytes: number; totalBytes: number; watermarkBytes: number };
  dataDir: string;
  lastSnapshot: { at: string; path: string } | null;
  lastCheck: { at: string; mode: 'quick' | 'full'; ok: boolean; issues: number } | null;
  warnings: StatusWarning[];
}

export type CheckKind =
  | 'missing-file'
  | 'size-mismatch'
  | 'hash-mismatch'
  | 'orphan-file'
  | 'pending-move'
  | 'stale-temp';

export interface CheckIssue {
  kind: CheckKind;
  imageId: string | null;
  path: string | null;
  repaired: boolean;
}

export interface CheckReport {
  mode: 'quick' | 'full';
  startedAt: string;
  durationMs: number;
  ok: boolean;
  checked: number;
  issues: CheckIssue[];
}

export interface Snapshot {
  path: string;
  images: number;
  bytes: number;
  createdAt: string;
}

export interface AuditEntry {
  id: number;
  at: string;
  actor: string;
  action: string;
  target: string | null;
  detail: string | null;
  ip: string | null;
}

export interface AuditPage {
  items: AuditEntry[];
  nextCursor: string | null;
}
