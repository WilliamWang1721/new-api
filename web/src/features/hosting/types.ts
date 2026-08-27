/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import type { AdminPermissionMatrix } from '@/lib/admin-permissions'

export type HostingRuntimeState = 'disabled' | 'ready' | 'error'

export type HostingAgent = {
  id: number
  name: string
  enabled: boolean
  user_id: number
  handoff_user_id: number
  brain_source: 'internal_channel' | 'dedicated'
  brain_model: string
  brain_group: string
  brain_channel_id: number
  dedicated_base_url: string
  dedicated_api_type: string
  dedicated_timeout_sec: number
  dedicated_headers: string
  default_channel_groups: string
  max_actions_per_incident: number
  dry_run: boolean
  daily_token_budget: number
  wake_merge_window_sec: number
  max_wakes_per_hour: number
  allow_agent_hooks: boolean
  max_agent_hooks: number
  min_hook_interval_sec: number
  context_window: number
  reserve_tokens: number
  keep_recent_tokens: number
  session_id: string
  last_compact_summary: string
  remark: string
  permission_preset?: string
  auto_review_mode?: string
  is_default?: boolean
  dedicated_key_set: boolean
  permissions: AdminPermissionMatrix
  token_prefixes: string[]
}

export type HostingCreateResult = {
  agent: HostingAgent
  token?: string
}

export type HostingIncident = {
  id: number
  agent_id: number
  status: string
  source_hook_id: number
  source_event: string
  summary: string
  handoff_reason: string
  actions_json?: string
  brain_source?: string
  created_at: number
}

export type HostingHook = {
  id: number
  agent_id: number
  owner: string
  name: string
  enabled: boolean
  kind: string
  event_name: string
  wake_mode: string
  cooldown_sec: number
  next_fire_at: number
  system_key: string
}

export type HostingSessionEntry = {
  id: number
  seq: number
  role: string
  name: string
  content: string
  created_at: number
  token_count: number
}

export type HostingToken = {
  id: number
  name: string
  token_prefix: string
  allow_ips: string
  status: number
}

export type HostingSessionPayload = {
  entries: HostingSessionEntry[]
  session_id: string
  last_compact_summary: string
  token_occupancy?: number
}

export type HostingStatus = {
  status: {
    enabled: boolean
    state: HostingRuntimeState
    error?: string
    env_enabled?: boolean
    panel_enabled?: boolean
  }
  snapshot: {
    hosting: string
    auto_disabled_count: number
    recent_error_logs: Array<{
      id: number
      content: string
      channel_id: number
    }>
    inspection_tasks?: Record<string, { status: string; updated_at: number }>
    host_resources?: { alloc_mb: number }
  }
  template: AdminPermissionMatrix
}

export type HostingApiResponse<T> = {
  success: boolean
  message?: string
  data: T
}
