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
import { describe, expect, it } from 'vitest'

import type { PermissionCatalog } from '@/lib/admin-permissions'

import {
  applyRecommendedPermissions,
  emptyPermissionMatrix,
} from '../lib/permissions'

const catalog: PermissionCatalog = {
  roles: [],
  resources: [
    {
      resource: 'channel',
      label_key: 'Channel Management',
      actions: [
        { action: 'read', label_key: 'Read channels', description_key: '' },
        {
          action: 'sensitive_write',
          label_key: 'Edit sensitive channel settings',
          description_key: '',
        },
        {
          action: 'secret_view',
          label_key: 'View channel secrets',
          description_key: '',
        },
      ],
    },
    {
      resource: 'log',
      label_key: 'Logs',
      actions: [
        { action: 'read', label_key: 'Read logs', description_key: '' },
      ],
    },
  ],
}

describe('hosting permission helpers', () => {
  it('starts from an empty matrix', () => {
    const matrix = emptyPermissionMatrix(catalog)
    expect(matrix.channel.read).toBe(false)
    expect(matrix.channel.secret_view).toBe(false)
    expect(matrix.log.read).toBe(false)
  })

  it('applies the recommended template without secret_view', () => {
    const matrix = applyRecommendedPermissions(catalog, {
      channel: { read: true, secret_view: true },
      log: { read: true },
    })
    expect(matrix.channel.read).toBe(true)
    expect(matrix.channel.secret_view).toBe(false)
    expect(matrix.channel.sensitive_write).toBe(false)
    expect(matrix.log.read).toBe(true)
  })
})
