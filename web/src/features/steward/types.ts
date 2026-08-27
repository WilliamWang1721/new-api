/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
    11|but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
export type StewardStatus = {
  ready: boolean
  needs_brain: boolean
  panel_enabled: boolean
  env_enabled: boolean
  default_agent_id: number
  pending_approvals: number
  can_review: boolean
  permission_preset: string
  auto_review_mode: string
  briefing_mode: string
  practice_mode: boolean
}

export type StewardSettings = {
  enabled: boolean
  brain_source: 'internal_channel' | 'dedicated'
  brain_model: string
  brain_group: string
  brain_channel_id: number
  dedicated_base_url: string
  dedicated_api_type: string
  dedicated_key_set: boolean
  permission_preset: 'watch' | 'operate' | 'full'
  auto_review_mode: 'off' | 'conservative' | 'balanced' | 'aggressive'
  briefing_mode: 'off' | 'every_open' | 'daily'
  daily_token_budget: number
  dry_run: boolean
  needs_brain: boolean
  agent_id: number
  pending_approvals: number
}

export type StewardApproval = {
  id: number
  kind: string
  status: string
  tool_name: string
  risk: string
  summary: string
  reason: string
  requester_user_id: number
  target_user_id: number
  created_at: number
}

export type StewardSessionEntry = {
  id: number
  seq: number
  role: string
  name: string
  content: string
  created_at: number
}

export type StewardApiResponse<T> = {
  success: boolean
  message?: string
  data: T
}
