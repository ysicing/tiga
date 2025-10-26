/**
 * Version information for Tiga server and agent
 */
export interface VersionInfo {
  /**
   * Version number
   * Format: v1.2.3-a1b2c3d (with tag) or 20251026-a1b2c3d (without tag) or "dev"
   */
  version: string;

  /**
   * Build timestamp in RFC3339 format
   * Example: "2025-10-26T12:34:56Z" or "unknown"
   */
  build_time: string;

  /**
   * Git commit ID (7-character short hash)
   * Format: 7-character hexadecimal string (e.g., "a1b2c3d") or "0000000"
   */
  commit_id: string;
}
