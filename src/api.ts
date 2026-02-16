// API service for MemoMarket backend communication
// Each backend server = one channel. Client manages multiple channels.

export interface ServerInfo {
  name: string;
  description: string;
}

export interface Channel {
  id: string;       // local UUID
  url: string;      // backend server URL
  token: string;    // auth token for this server
  name: string;     // display name (fetched from server /api/info)
  description: string;
}

export interface PublishRulePackReq {
  name: string;
  description: string;
  version: string;
  system_prompt: string;
  rules: { title: string; update_rule: string }[];
  tags: string[];
}

export interface PublishMemoPackReq {
  name: string;
  description: string;
  version: string;
  memos: { title: string; content: string }[];
  tags: string[];
}

export interface ListResponse<T> {
  items: T[];
  total: number;
  page: number;
  limit: number;
}

export interface UserInfo {
  id: string;
  username: string;
  display_name: string;
  token?: string;
  created_at: string;
}

function headers(token?: string): Record<string, string> {
  const h: Record<string, string> = { "Content-Type": "application/json" };
  if (token) h["Authorization"] = `Bearer ${token}`;
  return h;
}

async function request<T>(baseUrl: string, method: string, path: string, body?: unknown, token?: string): Promise<T> {
  const url = baseUrl.replace(/\/$/, "");
  const res = await fetch(`${url}${path}`, {
    method,
    headers: headers(token),
    body: body ? JSON.stringify(body) : undefined,
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || `HTTP ${res.status}`);
  return data as T;
}

// ---- Per-channel API calls ----

export async function fetchServerInfo(baseUrl: string): Promise<ServerInfo> {
  return request(baseUrl, "GET", "/api/info");
}

export async function registerOnServer(baseUrl: string, username: string, displayName: string): Promise<UserInfo> {
  return request(baseUrl, "POST", "/api/register", { username, display_name: displayName });
}

export async function getMe(baseUrl: string, token: string): Promise<UserInfo> {
  return request(baseUrl, "GET", "/api/me", undefined, token);
}

export async function publishRulePack(baseUrl: string, token: string, req: PublishRulePackReq): Promise<unknown> {
  return request(baseUrl, "POST", "/api/rule-packs", req, token);
}

export async function publishMemoPack(baseUrl: string, token: string, req: PublishMemoPackReq): Promise<unknown> {
  return request(baseUrl, "POST", "/api/memo-packs", req, token);
}

export async function listRulePacks(baseUrl: string, params: { search?: string; tag?: string; page?: number; limit?: number } = {}): Promise<ListResponse<unknown>> {
  const p = new URLSearchParams();
  if (params.search) p.set("search", params.search);
  if (params.tag) p.set("tag", params.tag);
  p.set("page", String(params.page || 1));
  if (params.limit) p.set("limit", String(params.limit));
  return request(baseUrl, "GET", `/api/rule-packs?${p}`);
}

export async function listMemoPacks(baseUrl: string, params: { search?: string; tag?: string; page?: number; limit?: number } = {}): Promise<ListResponse<unknown>> {
  const p = new URLSearchParams();
  if (params.search) p.set("search", params.search);
  if (params.tag) p.set("tag", params.tag);
  p.set("page", String(params.page || 1));
  if (params.limit) p.set("limit", String(params.limit));
  return request(baseUrl, "GET", `/api/memo-packs?${p}`);
}
