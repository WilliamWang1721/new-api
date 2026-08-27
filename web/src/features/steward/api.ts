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
import { api } from '@/lib/api'

import type {
  StewardApiResponse,
  StewardApproval,
  StewardSessionEntry,
  StewardSettings,
  StewardStatus,
} from './types'

export async function getStewardStatus() {
  const res = await api.get<StewardApiResponse<StewardStatus>>(
    '/api/hosting/steward/status'
  )
  return res.data
}

export async function getStewardSession() {
  const res = await api.get<
    StewardApiResponse<{
      session_id: string
      entries: StewardSessionEntry[]
    }>
  >('/api/hosting/steward/session')
  return res.data
}

export async function postStewardChat(message: string) {
  const res = await api.post<
    StewardApiResponse<{
      session_id: string
      entries: StewardSessionEntry[]
      pending_approvals: StewardApproval[]
      needs_brain: boolean
      reply: string
    }>
  >('/api/hosting/steward/chat', { message })
  return res.data
}

export async function rotateStewardSession() {
  const res = await api.post<StewardApiResponse<{ session_id: string }>>(
    '/api/hosting/steward/session/rotate'
  )
  return res.data
}

export async function listStewardApprovals(status = 'pending') {
  const res = await api.get<StewardApiResponse<StewardApproval[]>>(
    '/api/hosting/steward/approvals',
    { params: { status } }
  )
  return res.data
}

export async function decideStewardApproval(
  id: number,
  approve: boolean,
  note = ''
) {
  const res = await api.post<StewardApiResponse<StewardApproval>>(
    `/api/hosting/steward/approvals/${id}/decide`,
    { approve, note }
  )
  return res.data
}

export async function getStewardBriefing(refresh = false) {
  const res = await api.get<
    StewardApiResponse<{
      text: string
      generated: boolean
      mode: string
      cached: boolean
    }>
  >('/api/hosting/steward/briefing', {
    params: refresh ? { refresh: '1' } : undefined,
  })
  return res.data
}

export async function getStewardSettings() {
  const res = await api.get<StewardApiResponse<StewardSettings>>(
    '/api/hosting/admin/steward'
  )
  return res.data
}

export async function saveStewardSettings(payload: Record<string, unknown>) {
  const res = await api.put<StewardApiResponse<StewardSettings>>(
    '/api/hosting/admin/steward',
    payload
  )
  return res.data
}
