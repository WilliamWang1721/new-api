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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
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

import { testHostingBrain } from '@/features/hosting/api'
import { getStewardSettings, saveStewardSettings } from '@/features/steward/api'
import type { StewardSettings } from '@/features/steward/types'

const PRESETS = ['watch', 'operate', 'full'] as const
const REVIEW_MODES = ['off', 'conservative', 'balanced', 'aggressive'] as const
const BRIEFING_MODES = ['off', 'every_open', 'daily'] as const
const BUDGETS = [50000, 200000, 1000000] as const

function presetLabel(
  t: (key: string) => string,
  preset: (typeof PRESETS)[number]
) {
  if (preset === 'watch') return t('Watch only')
  if (preset === 'full') return t('Full helper')
  return t('Help me operate')
}

function reviewLabel(
  t: (key: string) => string,
  mode: (typeof REVIEW_MODES)[number]
) {
  if (mode === 'off') return t('Ask every time')
  if (mode === 'conservative') return t('Only auto-check')
  if (mode === 'aggressive') return t('Trust more')
  return t('Balanced')
}

function briefingLabel(
  t: (key: string) => string,
  mode: (typeof BRIEFING_MODES)[number]
) {
  if (mode === 'off') return t('Off')
  if (mode === 'daily') return t('Once a day')
  return t('When I open this page')
}

function budgetLabel(t: (key: string) => string, budget: number) {
  if (budget === 50000) return t('Light')
  if (budget === 1000000) return t('Heavy')
  return t('Normal')
}

export function SimpleStewardSetup() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const settingsQuery = useQuery({
    queryKey: ['steward-settings'],
    queryFn: getStewardSettings,
  })
  const [form, setForm] = useState<StewardSettings | null>(null)
  const [apiKey, setApiKey] = useState('')

  useEffect(() => {
    if (settingsQuery.data?.data) {
      const next = settingsQuery.data.data
      setForm({
        ...next,
        briefing_mode: next.briefing_mode || 'every_open',
        permission_preset: next.permission_preset || 'operate',
        auto_review_mode: next.auto_review_mode || 'balanced',
      })
    }
  }, [settingsQuery.data])

  const saveMutation = useMutation({
    mutationFn: () => {
      if (!form) {
        return Promise.reject(new Error('missing form'))
      }
      const payload: Record<string, unknown> = {
        enabled: form.enabled,
        brain_source: form.brain_source,
        brain_model: form.brain_model,
        brain_group: form.brain_group || 'default',
        brain_channel_id: form.brain_channel_id,
        dedicated_base_url: form.dedicated_base_url,
        dedicated_api_type: form.dedicated_api_type || 'openai',
        permission_preset: form.permission_preset,
        auto_review_mode: form.auto_review_mode,
        briefing_mode: form.briefing_mode,
        daily_token_budget: form.daily_token_budget,
        dry_run: form.dry_run,
      }
      if (apiKey.trim()) {
        payload.dedicated_api_key = apiKey.trim()
      }
      return saveStewardSettings(payload)
    },
    onSuccess: (res) => {
      if (!res.success) {
        toast.error(res.message || t('Failed to save steward settings'))
        return
      }
      toast.success(t('Steward settings saved'))
      setApiKey('')
      queryClient.invalidateQueries({ queryKey: ['steward-settings'] })
      queryClient.invalidateQueries({ queryKey: ['steward-status'] })
      queryClient.invalidateQueries({ queryKey: ['hosting-status'] })
    },
  })

  if (!form) {
    return (
      <p className='text-muted-foreground text-sm'>{t('Loading settings…')}</p>
    )
  }

  return (
    <div className='space-y-4'>
      <p className='text-muted-foreground text-sm'>
        {t(
          'You only need these options. Then talk to the steward in plain language on the AI Steward page.'
        )}
      </p>
      <Card>
        <CardHeader className='flex flex-row items-center justify-between'>
          <div>
            <CardTitle>{t('Turn on the steward')}</CardTitle>
            <CardDescription>
              {t('When this is off, you can still use the API as usual.')}
            </CardDescription>
          </div>
          <Switch
            checked={form.enabled}
            onCheckedChange={(checked) => setForm({ ...form, enabled: checked })}
          />
        </CardHeader>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t('Which AI should it use?')}</CardTitle>
          <CardDescription>
            {t('Pick a model you already have, or connect an outside AI.')}
          </CardDescription>
        </CardHeader>
        <CardContent className='grid gap-3'>
          <Select
            value={form.brain_source}
            onValueChange={(value) => {
              if (value !== 'internal_channel' && value !== 'dedicated') return
              setForm({ ...form, brain_source: value })
            }}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value='internal_channel'>
                {t('Use a model already in New API')}
              </SelectItem>
              <SelectItem value='dedicated'>
                {t('Connect an outside AI')}
              </SelectItem>
            </SelectContent>
          </Select>
          <div className='grid gap-1'>
            <Label>{t('Model name')}</Label>
            <Input
              value={form.brain_model}
              placeholder='gpt-4o-mini'
              onChange={(event) =>
                setForm({ ...form, brain_model: event.target.value })
              }
            />
          </div>
          {form.brain_source === 'internal_channel' ? (
            <p className='text-muted-foreground text-xs'>
              {t('The steward will use this model from your existing channels.')}
            </p>
          ) : (
            <>
              <div className='grid gap-1'>
                <Label>{t('AI server URL')}</Label>
                <Input
                  value={form.dedicated_base_url}
                  placeholder='https://api.openai.com'
                  onChange={(event) =>
                    setForm({ ...form, dedicated_base_url: event.target.value })
                  }
                />
              </div>
              <div className='grid gap-1'>
                <Label>{t('API key')}</Label>
                <Input
                  type='password'
                  value={apiKey}
                  placeholder={
                    form.dedicated_key_set
                      ? t('Leave blank to keep the current key')
                      : ''
                  }
                  onChange={(event) => setApiKey(event.target.value)}
                />
              </div>
              <Button
                type='button'
                variant='outline'
                onClick={async () => {
                  const res = await testHostingBrain({
                    base_url: form.dedicated_base_url,
                    api_key: apiKey,
                    model: form.brain_model,
                    api_type: form.dedicated_api_type || 'openai',
                  })
                  if (res.data?.ok) {
                    toast.success(t('Connection looks good'))
                  } else {
                    toast.error(
                      res.data?.message || t('Could not reach that AI server')
                    )
                  }
                }}
              >
                {t('Test connection')}
              </Button>
            </>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t('How much can it do by itself?')}</CardTitle>
          <CardDescription>
            {t(
              'Watch only reports. Helper can check, test, and change everyday settings. Full helper can also manage users after you confirm risky steps.'
            )}
          </CardDescription>
        </CardHeader>
        <CardContent className='flex flex-wrap gap-2'>
          {PRESETS.map((preset) => (
            <Button
              key={preset}
              variant={form.permission_preset === preset ? 'default' : 'outline'}
              onClick={() => setForm({ ...form, permission_preset: preset })}
            >
              {presetLabel(t, preset)}
            </Button>
          ))}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t('Auto Review')}</CardTitle>
          <CardDescription>
            {t(
              'Safe checks can run by themselves. Risky changes still wait for your Approve button.'
            )}
          </CardDescription>
        </CardHeader>
        <CardContent className='flex flex-wrap gap-2'>
          {REVIEW_MODES.map((mode) => (
            <Button
              key={mode}
              variant={form.auto_review_mode === mode ? 'default' : 'outline'}
              onClick={() => setForm({ ...form, auto_review_mode: mode })}
            >
              {reviewLabel(t, mode)}
            </Button>
          ))}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t('Status briefing')}</CardTitle>
          <CardDescription>
            {t(
              'When you open AI Steward, it can summarize what happened recently.'
            )}
          </CardDescription>
        </CardHeader>
        <CardContent className='flex flex-wrap gap-2'>
          {BRIEFING_MODES.map((mode) => (
            <Button
              key={mode}
              variant={form.briefing_mode === mode ? 'default' : 'outline'}
              onClick={() => setForm({ ...form, briefing_mode: mode })}
            >
              {briefingLabel(t, mode)}
            </Button>
          ))}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t('How much chatting per day?')}</CardTitle>
          <CardDescription>
            {t(
              'Practice mode lets you talk without changing the live system.'
            )}
          </CardDescription>
        </CardHeader>
        <CardContent className='flex flex-wrap items-center gap-2'>
          {BUDGETS.map((budget) => (
            <Button
              key={budget}
              variant={
                form.daily_token_budget === budget ? 'default' : 'outline'
              }
              onClick={() => setForm({ ...form, daily_token_budget: budget })}
            >
              {budgetLabel(t, budget)}
            </Button>
          ))}
          <div className='flex items-center gap-2 pl-2'>
            <span className='text-sm'>{t('Practice mode')}</span>
            <Switch
              checked={form.dry_run}
              onCheckedChange={(checked) =>
                setForm({ ...form, dry_run: checked })
              }
            />
          </div>
        </CardContent>
      </Card>

      <Button
        onClick={() => saveMutation.mutate()}
        disabled={saveMutation.isPending}
      >
        {t('Save steward settings')}
      </Button>
    </div>
  )
}
