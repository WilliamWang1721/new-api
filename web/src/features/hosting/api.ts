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
import type {
  AdminPermissionMatrix,
  PermissionCatalog,
} from '@/lib/admin-permissions'
import { api } from '@/lib/api'

import type {
  HostingAgent,
  HostingApiResponse,
  HostingCreateResult,
  HostingHook,
  HostingIncident,
  HostingSessionPayload,
  HostingStatus,
  HostingToken,
} from './types'

export async function getHostingStatus() {
  const res = await api.get<HostingApiResponse<HostingStatus>>(
    '/api/hosting/admin/status'
  )
  return res.data
}

export async function listHostingAgents() {
  const res = await api.get<HostingApiResponse<HostingAgent[]>>(
    '/api/hosting/admin/agents'
  )
  return res.data
}

export async function createHostingAgent(payload: Record<string, unknown>) {
  const res = await api.post<HostingApiResponse<HostingCreateResult>>(
    '/api/hosting/admin/agents',
    payload
  )
  return res.data
}

export async function updateHostingAgent(
  id: number,
  payload: Record<string, unknown>
) {
  const res = await api.put<HostingApiResponse<HostingAgent>>(
    `/api/hosting/admin/agents/${id}`,
    payload
  )
  return res.data
}

export async function deleteHostingAgent(id: number) {
  const res = await api.delete<HostingApiResponse<null>>(
    `/api/hosting/admin/agents/${id}`
  )
  return res.data
}

export async function createHostingAgentToken(
  id: number,
  payload: { name?: string; allow_ips?: string }
) {
  const res = await api.post<
    HostingApiResponse<{ secret: string; token: { token_prefix: string } }>
  >(`/api/hosting/admin/agents/${id}/tokens`, payload)
  return res.data
}

export async function listHostingAgentTokens(id: number) {
  const res = await api.get<HostingApiResponse<HostingToken[]>>(
    `/api/hosting/admin/agents/${id}/tokens`
  )
  return res.data
}

export async function rotateHostingAgentToken(
  id: number,
  tokenId: number,
  payload: { allow_ips?: string } = {}
) {
  const res = await api.post<
    HostingApiResponse<{ secret: string; token: HostingToken }>
  >(`/api/hosting/admin/agents/${id}/tokens/${tokenId}/rotate`, payload)
  return res.data
}

export async function revokeHostingAgentToken(id: number, tokenId: number) {
  const res = await api.delete<HostingApiResponse<null>>(
    `/api/hosting/admin/agents/${id}/tokens/${tokenId}`
  )
  return res.data
}

export async function setHostingPanelEnabled(enabled: boolean) {
  const res = await api.put<
    HostingApiResponse<{
      enabled: boolean
      state: string
      env_enabled: boolean
      panel_enabled: boolean
    }>
  >('/api/hosting/admin/enabled', { enabled })
  return res.data
}

export async function testHostingBrain(payload: {
  base_url: string
  api_key: string
  model: string
  timeout_sec?: number
  extra_headers?: string
  api_type?: string
}) {
  const res = await api.post<
    HostingApiResponse<{ ok: boolean; message: string }>
  >('/api/hosting/admin/brain/test', payload)
  return res.data
}

export async function listHostingIncidents() {
  const res = await api.get<HostingApiResponse<HostingIncident[]>>(
    '/api/hosting/admin/incidents'
  )
  return res.data
}

export async function updateHostingIncident(id: number, status: string) {
  const res = await api.put<HostingApiResponse<HostingIncident>>(
    `/api/hosting/admin/incidents/${id}`,
    { status }
  )
  return res.data
}

export async function listHostingHooks() {
  const res = await api.get<HostingApiResponse<HostingHook[]>>(
    '/api/hosting/admin/hooks'
  )
  return res.data
}

export async function updateHostingHook(
  id: number,
  payload: { enabled?: boolean; wake_mode?: string }
) {
  const res = await api.put<HostingApiResponse<HostingHook>>(
    `/api/hosting/admin/hooks/${id}`,
    payload
  )
  return res.data
}

export async function deleteHostingHook(id: number) {
  const res = await api.delete<HostingApiResponse<null>>(
    `/api/hosting/admin/hooks/${id}`
  )
  return res.data
}

export async function getHostingSession(id: number) {
  const res = await api.get<HostingApiResponse<HostingSessionPayload>>(
    `/api/hosting/admin/agents/${id}/session`
  )
  return res.data
}

export async function exportHostingSession(id: number) {
  const res = await api.get<HostingApiResponse<{ export: string }>>(
    `/api/hosting/admin/agents/${id}/session/export`
  )
  return res.data
}

export async function getHostingCost(id: number) {
  const res = await api.get<
    HostingApiResponse<{ tokens_used: number; wakes: number; day: string }>
  >(`/api/hosting/admin/agents/${id}/cost`)
  return res.data
}

export async function rotateHostingSession(id: number) {
  const res = await api.post<HostingApiResponse<HostingAgent>>(
    `/api/hosting/admin/agents/${id}/session/rotate`
  )
  return res.data
}

export async function getHostingPermissionCatalog(): Promise<PermissionCatalog> {
  const res = await api.get('/api/authz/catalog')
  return {
    resources: res.data?.data?.resources ?? [],
    roles: res.data?.data?.roles ?? [],
  }
}

export async function updateHostingAgentPermissions(
  id: number,
  permissions: AdminPermissionMatrix
) {
  const res = await api.put<HostingApiResponse<AdminPermissionMatrix>>(
    `/api/hosting/admin/agents/${id}/permissions`,
    permissions
  )
  return res.data
}
