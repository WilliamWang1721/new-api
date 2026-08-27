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
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import type {
  AdminPermissionMatrix,
  PermissionCatalog,
} from '@/lib/admin-permissions'

import { testHostingBrain } from '../api'
import type { HostingAgentForm } from '../lib/agent-form'
import { applyRecommendedPermissions } from '../lib/permissions'
import type { HostingAgent } from '../types'

import { PermissionMatrixEditor } from './permission-matrix'

type AgentEditorDialogProps = {
  open: boolean
  editing: HostingAgent | null
  form: HostingAgentForm
  permissions: AdminPermissionMatrix
  catalog: PermissionCatalog
  template: AdminPermissionMatrix
  secret: string
  saving: boolean
  onOpenChange: (open: boolean) => void
  onFormChange: (form: HostingAgentForm) => void
  onPermissionsChange: (permissions: AdminPermissionMatrix) => void
  onSave: () => void
}

export function AgentEditorDialog(props: AgentEditorDialogProps) {
  const { t } = useTranslation()

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='max-h-[90vh] overflow-y-auto sm:max-w-2xl'>
        <DialogHeader>
          <DialogTitle>
            {props.editing
              ? t('Edit hosting agent')
              : t('Create hosting agent')}
          </DialogTitle>
          <DialogDescription>
            {t(
              'Choose a brain source, grant tools, and optionally issue a hat- token.'
            )}
          </DialogDescription>
        </DialogHeader>
        <div className='grid gap-3'>
          <div className='grid gap-1'>
            <Label>{t('Name')}</Label>
            <Input
              value={props.form.name}
              onChange={(event) =>
                props.onFormChange({ ...props.form, name: event.target.value })
              }
            />
          </div>
          <div className='flex items-center justify-between'>
            <Label>{t('Enabled')}</Label>
            <Switch
              checked={props.form.enabled}
              onCheckedChange={(checked) =>
                props.onFormChange({ ...props.form, enabled: checked })
              }
            />
          </div>
          <div className='grid gap-1'>
            <Label>{t('Brain source')}</Label>
            <Select
              value={props.form.brain_source}
              onValueChange={(value) => {
                if (value !== 'internal_channel' && value !== 'dedicated') {
                  return
                }
                props.onFormChange({ ...props.form, brain_source: value })
              }}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value='internal_channel'>
                  {t('Internal channel')}
                </SelectItem>
                <SelectItem value='dedicated'>
                  {t('Dedicated upstream')}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className='grid gap-1'>
            <Label>{t('Model')}</Label>
            <Input
              value={props.form.brain_model}
              onChange={(event) =>
                props.onFormChange({
                  ...props.form,
                  brain_model: event.target.value,
                })
              }
            />
          </div>
          {props.form.brain_source === 'internal_channel' ? (
            <>
              <div className='grid gap-1'>
                <Label>{t('Group')}</Label>
                <Input
                  value={props.form.brain_group}
                  onChange={(event) =>
                    props.onFormChange({
                      ...props.form,
                      brain_group: event.target.value,
                    })
                  }
                />
              </div>
              <div className='grid gap-1'>
                <Label>{t('Pinned channel ID')}</Label>
                <Input
                  type='number'
                  value={props.form.brain_channel_id}
                  onChange={(event) =>
                    props.onFormChange({
                      ...props.form,
                      brain_channel_id: Number(event.target.value),
                    })
                  }
                />
              </div>
            </>
          ) : (
            <>
              <div className='grid gap-1'>
                <Label>{t('Base URL')}</Label>
                <Input
                  value={props.form.dedicated_base_url}
                  onChange={(event) =>
                    props.onFormChange({
                      ...props.form,
                      dedicated_base_url: event.target.value,
                    })
                  }
                />
              </div>
              <div className='grid gap-1'>
                <Label>{t('API key')}</Label>
                <Input
                  type='password'
                  value={props.form.dedicated_api_key}
                  placeholder={
                    props.editing
                      ? t('Leave blank to keep the current key')
                      : ''
                  }
                  onChange={(event) =>
                    props.onFormChange({
                      ...props.form,
                      dedicated_api_key: event.target.value,
                    })
                  }
                />
              </div>
              <div className='grid gap-1'>
                <Label>{t('API type')}</Label>
                <Input
                  value={props.form.dedicated_api_type}
                  onChange={(event) =>
                    props.onFormChange({
                      ...props.form,
                      dedicated_api_type: event.target.value,
                    })
                  }
                />
              </div>
              <div className='grid gap-1'>
                <Label>{t('Extra headers')}</Label>
                <Textarea
                  value={props.form.dedicated_headers}
                  placeholder={t('JSON object or Header: value lines')}
                  onChange={(event) =>
                    props.onFormChange({
                      ...props.form,
                      dedicated_headers: event.target.value,
                    })
                  }
                />
              </div>
              <Button
                type='button'
                variant='outline'
                onClick={async () => {
                  const res = await testHostingBrain({
                    base_url: props.form.dedicated_base_url,
                    api_key: props.form.dedicated_api_key,
                    model: props.form.brain_model,
                    extra_headers: props.form.dedicated_headers,
                    api_type: props.form.dedicated_api_type,
                    timeout_sec: props.form.dedicated_timeout_sec,
                  })
                  if (res.data?.ok) {
                    toast.success(t('Brain connectivity test succeeded'))
                  } else {
                    toast.error(
                      res.data?.message || t('Brain connectivity test failed')
                    )
                  }
                }}
              >
                {t('Test connectivity')}
              </Button>
            </>
          )}
          <div className='grid gap-1'>
            <Label>{t('Handoff user ID')}</Label>
            <Input
              type='number'
              value={props.form.handoff_user_id}
              onChange={(event) =>
                props.onFormChange({
                  ...props.form,
                  handoff_user_id: Number(event.target.value),
                })
              }
            />
          </div>
          <div className='flex items-center justify-between'>
            <Label>{t('Allow agent-created hooks')}</Label>
            <Switch
              checked={props.form.allow_agent_hooks}
              onCheckedChange={(checked) =>
                props.onFormChange({
                  ...props.form,
                  allow_agent_hooks: checked,
                })
              }
            />
          </div>
          <div className='flex items-center justify-between'>
            <Label>{t('Dry run')}</Label>
            <Switch
              checked={props.form.dry_run}
              onCheckedChange={(checked) =>
                props.onFormChange({ ...props.form, dry_run: checked })
              }
            />
          </div>
          <div className='grid gap-1'>
            <Label>{t('Daily token budget')}</Label>
            <Input
              type='number'
              value={props.form.daily_token_budget}
              onChange={(event) =>
                props.onFormChange({
                  ...props.form,
                  daily_token_budget: Number(event.target.value),
                })
              }
            />
          </div>
          <div className='grid gap-1'>
            <Label>{t('Wake merge window (seconds)')}</Label>
            <Input
              type='number'
              value={props.form.wake_merge_window_sec}
              onChange={(event) =>
                props.onFormChange({
                  ...props.form,
                  wake_merge_window_sec: Number(event.target.value),
                })
              }
            />
          </div>
          <div className='grid gap-1'>
            <Label>{t('Max wakes per hour')}</Label>
            <Input
              type='number'
              value={props.form.max_wakes_per_hour}
              onChange={(event) =>
                props.onFormChange({
                  ...props.form,
                  max_wakes_per_hour: Number(event.target.value),
                })
              }
            />
          </div>
          <div className='grid gap-1'>
            <Label>{t('Max actions per incident')}</Label>
            <Input
              type='number'
              value={props.form.max_actions_per_incident}
              onChange={(event) =>
                props.onFormChange({
                  ...props.form,
                  max_actions_per_incident: Number(event.target.value),
                })
              }
            />
          </div>
          <div className='grid gap-1'>
            <Label>{t('Context window')}</Label>
            <Input
              type='number'
              value={props.form.context_window}
              onChange={(event) =>
                props.onFormChange({
                  ...props.form,
                  context_window: Number(event.target.value),
                })
              }
            />
          </div>
          <div className='grid gap-1'>
            <Label>{t('Reserve tokens')}</Label>
            <Input
              type='number'
              value={props.form.reserve_tokens}
              onChange={(event) =>
                props.onFormChange({
                  ...props.form,
                  reserve_tokens: Number(event.target.value),
                })
              }
            />
          </div>
          <div className='grid gap-1'>
            <Label>{t('Keep recent tokens')}</Label>
            <Input
              type='number'
              value={props.form.keep_recent_tokens}
              onChange={(event) =>
                props.onFormChange({
                  ...props.form,
                  keep_recent_tokens: Number(event.target.value),
                })
              }
            />
          </div>
          <div className='grid gap-1'>
            <Label>{t('Default channel groups')}</Label>
            <Input
              value={props.form.default_channel_groups}
              onChange={(event) =>
                props.onFormChange({
                  ...props.form,
                  default_channel_groups: event.target.value,
                })
              }
            />
          </div>
          {!props.editing ? (
            <>
              <div className='flex items-center justify-between'>
                <Label>{t('Issue token on create')}</Label>
                <Switch
                  checked={props.form.issue_token}
                  onCheckedChange={(checked) =>
                    props.onFormChange({ ...props.form, issue_token: checked })
                  }
                />
              </div>
              {props.form.issue_token ? (
                <div className='grid gap-1'>
                  <Label>{t('Token IP allowlist')}</Label>
                  <Input
                    value={props.form.token_allow_ips}
                    placeholder={t('Optional comma-separated IPs or CIDRs')}
                    onChange={(event) =>
                      props.onFormChange({
                        ...props.form,
                        token_allow_ips: event.target.value,
                      })
                    }
                  />
                </div>
              ) : null}
            </>
          ) : null}
          <Button
            type='button'
            variant='outline'
            onClick={() =>
              props.onPermissionsChange(
                applyRecommendedPermissions(props.catalog, props.template)
              )
            }
          >
            {t('Fill recommended channel permissions')}
          </Button>
          <div className='space-y-2'>
            <h3 className='text-sm font-medium'>{t('Permission matrix')}</h3>
            <PermissionMatrixEditor
              catalog={props.catalog}
              permissions={props.permissions}
              onChange={props.onPermissionsChange}
            />
          </div>
          {props.secret ? (
            <div className='grid gap-1'>
              <Label>
                {t('Copy this token now. It will not be shown again.')}
              </Label>
              <Textarea readOnly value={props.secret} />
            </div>
          ) : null}
        </div>
        <DialogFooter>
          <Button
            variant='outline'
            onClick={() => props.onOpenChange(false)}
          >
            {t('Cancel')}
          </Button>
          <Button onClick={props.onSave} disabled={props.saving}>
            {t('Save')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
