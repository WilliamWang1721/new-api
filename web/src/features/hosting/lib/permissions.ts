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

export const SECRET_VIEW_ACTION = 'secret_view'

export function emptyPermissionMatrix(
  catalog: PermissionCatalog
): AdminPermissionMatrix {
  const matrix: AdminPermissionMatrix = {}
  for (const resource of catalog.resources) {
    matrix[resource.resource] = {}
    for (const action of resource.actions) {
      matrix[resource.resource][action.action] = false
    }
  }
  return matrix
}

export function applyRecommendedPermissions(
  catalog: PermissionCatalog,
  template: AdminPermissionMatrix
): AdminPermissionMatrix {
  const matrix = emptyPermissionMatrix(catalog)
  for (const [resource, actions] of Object.entries(template)) {
    if (!matrix[resource]) {
      matrix[resource] = {}
    }
    for (const [action, allowed] of Object.entries(actions)) {
      if (action === SECRET_VIEW_ACTION) {
        continue
      }
      matrix[resource][action] = allowed === true
    }
  }
  return matrix
}
