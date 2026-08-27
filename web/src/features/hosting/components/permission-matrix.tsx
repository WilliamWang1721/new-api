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
import { useTranslation } from 'react-i18next'

import { Checkbox } from '@/components/ui/checkbox'
import type {
  AdminPermissionMatrix,
  PermissionCatalog,
} from '@/lib/admin-permissions'

import { SECRET_VIEW_ACTION } from '../lib/permissions'

type PermissionMatrixEditorProps = {
  catalog: PermissionCatalog
  permissions: AdminPermissionMatrix
  onChange: (next: AdminPermissionMatrix) => void
}

export function PermissionMatrixEditor(props: PermissionMatrixEditorProps) {
  const { t } = useTranslation()

  return (
    <div className='space-y-2'>
      {props.catalog.resources.map((resource) => (
        <div key={resource.resource} className='space-y-2 rounded-md border p-3'>
          <div className='text-sm font-medium'>{t(resource.label_key)}</div>
          <div className='space-y-2'>
            {resource.actions.map((option) => {
              const locked = option.action === SECRET_VIEW_ACTION
              return (
                <label key={option.action} className='flex items-start gap-3'>
                  <Checkbox
                    checked={
                      !locked &&
                      props.permissions[resource.resource]?.[option.action] ===
                        true
                    }
                    disabled={locked}
                    onCheckedChange={(checked) => {
                      if (locked) {
                        return
                      }
                      props.onChange({
                        ...props.permissions,
                        [resource.resource]: {
                          ...props.permissions[resource.resource],
                          [option.action]: checked === true,
                        },
                      })
                    }}
                  />
                  <span className='flex flex-col gap-1'>
                    <span className='text-sm font-medium'>
                      {t(option.label_key)}
                    </span>
                    <span className='text-muted-foreground text-xs'>
                      {t(option.description_key)}
                    </span>
                  </span>
                </label>
              )
            })}
          </div>
        </div>
      ))}
    </div>
  )
}
