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
import type { HostingAgent } from '../types'

export type HostingAgentForm = {
  name: string
  enabled: boolean
  handoff_user_id: number
  brain_source: 'internal_channel' | 'dedicated'
  brain_model: string
  brain_group: string
  brain_channel_id: number
  dedicated_base_url: string
  dedicated_api_key: string
  dedicated_api_type: string
  dedicated_headers: string
  dedicated_timeout_sec: number
  token_allow_ips: string
  default_channel_groups: string
  daily_token_budget: number
  wake_merge_window_sec: number
  max_wakes_per_hour: number
  max_actions_per_incident: number
  context_window: number
  reserve_tokens: number
  keep_recent_tokens: number
  allow_agent_hooks: boolean
  dry_run: boolean
  issue_token: boolean
  apply_recommended_permissions: boolean
}

export const EMPTY_HOSTING_AGENT_FORM: HostingAgentForm = {
  name: '',
  enabled: true,
  handoff_user_id: 0,
  brain_source: 'internal_channel',
  brain_model: '',
  brain_group: 'default',
  brain_channel_id: 0,
  dedicated_base_url: '',
  dedicated_api_key: '',
  dedicated_api_type: 'openai',
  dedicated_headers: '',
  dedicated_timeout_sec: 60,
  token_allow_ips: '',
  default_channel_groups: 'default',
  daily_token_budget: 200000,
  wake_merge_window_sec: 60,
  max_wakes_per_hour: 10,
  max_actions_per_incident: 20,
  context_window: 128000,
  reserve_tokens: 20000,
  keep_recent_tokens: 20000,
  allow_agent_hooks: true,
  dry_run: false,
  issue_token: true,
  apply_recommended_permissions: true,
}

export function formFromAgent(agent: HostingAgent): HostingAgentForm {
  return {
    ...EMPTY_HOSTING_AGENT_FORM,
    name: agent.name,
    enabled: agent.enabled,
    handoff_user_id: agent.handoff_user_id,
    brain_source: agent.brain_source,
    brain_model: agent.brain_model,
    brain_group: agent.brain_group,
    brain_channel_id: agent.brain_channel_id,
    dedicated_base_url: agent.dedicated_base_url,
    dedicated_api_key: '',
    dedicated_api_type: agent.dedicated_api_type || 'openai',
    dedicated_headers: agent.dedicated_headers || '',
    dedicated_timeout_sec: agent.dedicated_timeout_sec,
    default_channel_groups: agent.default_channel_groups,
    daily_token_budget: agent.daily_token_budget,
    wake_merge_window_sec: agent.wake_merge_window_sec,
    max_wakes_per_hour: agent.max_wakes_per_hour,
    max_actions_per_incident: agent.max_actions_per_incident,
    context_window: agent.context_window,
    reserve_tokens: agent.reserve_tokens,
    keep_recent_tokens: agent.keep_recent_tokens,
    allow_agent_hooks: agent.allow_agent_hooks,
    dry_run: agent.dry_run,
    issue_token: false,
    apply_recommended_permissions: false,
  }
}

export function agentFormPayload(form: HostingAgentForm) {
  return {
    ...form,
    brain_channel_id: Number(form.brain_channel_id) || 0,
    handoff_user_id: Number(form.handoff_user_id) || 0,
  }
}
