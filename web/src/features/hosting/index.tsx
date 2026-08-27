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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import {
  EMPTY_PERMISSION_CATALOG,
  type AdminPermissionMatrix,
} from '@/lib/admin-permissions'

import {
  createHostingAgent,
  createHostingAgentToken,
  deleteHostingAgent,
  deleteHostingHook,
  exportHostingSession,
  getHostingCost,
  getHostingPermissionCatalog,
  getHostingSession,
  getHostingStatus,
  listHostingAgentTokens,
  listHostingAgents,
  listHostingHooks,
  listHostingIncidents,
  rotateHostingAgentToken,
  rotateHostingSession,
  setHostingPanelEnabled,
  updateHostingAgent,
  updateHostingHook,
  updateHostingIncident,
} from './api'
import { AgentEditorDialog } from './components/agent-editor-dialog'
import {
  agentFormPayload,
  EMPTY_HOSTING_AGENT_FORM,
  formFromAgent,
  type HostingAgentForm,
} from './lib/agent-form'
import { applyRecommendedPermissions } from './lib/permissions'
import type { HostingAgent } from './types'

export function HostingPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<HostingAgent | null>(null)
  const [form, setForm] = useState<HostingAgentForm>(EMPTY_HOSTING_AGENT_FORM)
  const [permissions, setPermissions] = useState<AdminPermissionMatrix>({})
  const [secret, setSecret] = useState('')
  const [tokenAllowIps, setTokenAllowIps] = useState('')
  const [selectedAgentId, setSelectedAgentId] = useState<number | null>(null)
  const tokensQuery = useQuery({
    queryKey: ['hosting-tokens', selectedAgentId],
    queryFn: () => listHostingAgentTokens(selectedAgentId as number),
    enabled: selectedAgentId != null,
  })

  const statusQuery = useQuery({
    queryKey: ['hosting-status'],
    queryFn: getHostingStatus,
  })
  const agentsQuery = useQuery({
    queryKey: ['hosting-agents'],
    queryFn: listHostingAgents,
  })
  const incidentsQuery = useQuery({
    queryKey: ['hosting-incidents'],
    queryFn: listHostingIncidents,
  })
  const hooksQuery = useQuery({
    queryKey: ['hosting-hooks'],
    queryFn: listHostingHooks,
  })
  const catalogQuery = useQuery({
    queryKey: ['hosting-permission-catalog'],
    queryFn: getHostingPermissionCatalog,
  })
  const sessionQuery = useQuery({
    queryKey: ['hosting-session', selectedAgentId],
    queryFn: () => getHostingSession(selectedAgentId as number),
    enabled: selectedAgentId != null,
  })
  const costQuery = useQuery({
    queryKey: ['hosting-cost', selectedAgentId],
    queryFn: () => getHostingCost(selectedAgentId as number),
    enabled: selectedAgentId != null,
  })

  const catalog = catalogQuery.data ?? EMPTY_PERMISSION_CATALOG
  const template = statusQuery.data?.data?.template ?? {}
  const status = statusQuery.data?.data?.status
  const snapshot = statusQuery.data?.data?.snapshot
  const agents = agentsQuery.data?.data ?? []
  const incidents = incidentsQuery.data?.data ?? []
  const hooks = hooksQuery.data?.data ?? []
  const hostingState = status?.state ?? 'disabled'
  const envEnabled = status?.env_enabled !== false
  const panelEnabled = status?.panel_enabled !== false
  const sessionPayload = sessionQuery.data?.data
  const sessionEntries = sessionPayload?.entries ?? []
  const selectedAgent = agents.find((agent) => agent.id === selectedAgentId)

  const saveMutation = useMutation({
    mutationFn: async () => {
      const payload: Record<string, unknown> = {
        ...agentFormPayload(form),
        permissions,
      }
      if (editing) {
        return updateHostingAgent(editing.id, payload)
      }
      return createHostingAgent(payload)
    },
    onSuccess: (res) => {
      if (!res.success) {
        toast.error(res.message || t('Failed to save hosting agent'))
        return
      }
      toast.success(t('Hosting agent saved'))
      if ('token' in res.data && res.data.token) {
        setSecret(res.data.token)
      } else {
        setDialogOpen(false)
      }
      queryClient.invalidateQueries({ queryKey: ['hosting-agents'] })
    },
  })

  const openCreate = () => {
    setEditing(null)
    setForm(EMPTY_HOSTING_AGENT_FORM)
    setPermissions(applyRecommendedPermissions(catalog, template))
    setSecret('')
    setDialogOpen(true)
  }

  const openEdit = (agent: HostingAgent) => {
    setEditing(agent)
    setForm(formFromAgent(agent))
    setPermissions(agent.permissions ?? applyRecommendedPermissions(catalog, template))
    setSecret('')
    setDialogOpen(true)
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        <span className='inline-flex min-w-0 items-center gap-2'>
          <span className='truncate'>{t('Intelligent Hosting')}</span>
          <Badge variant='outline' className='shrink-0'>
            Root
          </Badge>
        </span>
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <div className='flex items-center gap-3'>
          <span className='text-muted-foreground text-sm'>
            {t('Hosting enabled')}
          </span>
          <Switch
            checked={panelEnabled && hostingState !== 'disabled'}
            disabled={!envEnabled}
            onCheckedChange={async (checked) => {
              const res = await setHostingPanelEnabled(checked)
              if (!res.success) {
                toast.error(res.message || t('Failed to update hosting switch'))
                return
              }
              queryClient.invalidateQueries({ queryKey: ['hosting-status'] })
              toast.success(
                checked ? t('Hosting enabled') : t('Hosting disabled')
              )
            }}
          />
          <Button onClick={openCreate}>{t('Create agent')}</Button>
        </div>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='space-y-4'>
          {hostingState !== 'ready' && (
            <Alert variant={hostingState === 'error' ? 'destructive' : 'default'}>
              <AlertTitle>
                {hostingState === 'error'
                  ? t('Hosting failed to start')
                  : t('Hosting is disabled')}
              </AlertTitle>
              <AlertDescription>
                {status?.error
                  ? status.error
                  : t(
                      'The API continues to serve requests. Enable hosting with HOSTING_ENABLED or fix the startup error.'
                    )}
              </AlertDescription>
            </Alert>
          )}

          {secret ? (
            <Alert>
              <AlertTitle>
                {t('Copy this token now. It will not be shown again.')}
              </AlertTitle>
              <AlertDescription>
                <Textarea readOnly value={secret} className='mt-2' />
              </AlertDescription>
            </Alert>
          ) : null}

          <div className='grid gap-1'>
            <span className='text-muted-foreground text-sm'>
              {t('Token IP allowlist')}
            </span>
            <Input
              value={tokenAllowIps}
              placeholder={t('Optional comma-separated IPs or CIDRs')}
              onChange={(event) => setTokenAllowIps(event.target.value)}
            />
          </div>

          <Card>
            <CardHeader>
              <CardTitle>{t('Runtime snapshot')}</CardTitle>
              <CardDescription>
                {t('Non-AI monitoring summary. Auto-disabled channels: {{count}}', {
                  count: snapshot?.auto_disabled_count ?? 0,
                })}
              </CardDescription>
            </CardHeader>
            <CardContent className='text-muted-foreground space-y-2 text-sm'>
              <div>
                {t('Hosting state')}: {hostingState}
              </div>
              <div>
                {t('Inspection tasks')}:{' '}
                {snapshot?.inspection_tasks
                  ? Object.entries(snapshot.inspection_tasks)
                      .map(([name, task]) => `${name}=${task.status}`)
                      .join(', ') || t('None')
                  : t('None')}
              </div>
              <div>
                {t('Host memory')}: {snapshot?.host_resources?.alloc_mb ?? 0} MB
              </div>
            </CardContent>
          </Card>

          <Tabs defaultValue='agents'>
            <TabsList>
              <TabsTrigger value='agents'>{t('Agents')}</TabsTrigger>
              <TabsTrigger value='incidents'>{t('Incidents')}</TabsTrigger>
              <TabsTrigger value='hooks'>{t('Hooks')}</TabsTrigger>
              <TabsTrigger value='session'>{t('Session')}</TabsTrigger>
            </TabsList>
            <TabsContent value='agents' className='space-y-3 pt-3'>
              {agents.length === 0 ? (
                <p className='text-muted-foreground text-sm'>
                  {t('No hosting agents yet')}
                </p>
              ) : (
                agents.map((agent) => (
                  <Card key={agent.id}>
                    <CardHeader className='flex flex-row items-start justify-between gap-2'>
                      <div>
                        <CardTitle>{agent.name}</CardTitle>
                        <CardDescription>
                          {agent.brain_source === 'dedicated'
                            ? t('Dedicated upstream')
                            : t('Internal channel')}{' '}
                          · {agent.brain_model || t('No model')} ·{' '}
                          {agent.token_prefixes?.join(', ') || t('No token')}
                        </CardDescription>
                      </div>
                      <div className='flex flex-wrap gap-2'>
                        <Button
                          variant='outline'
                          size='sm'
                          onClick={() => setSelectedAgentId(agent.id)}
                        >
                          {t('Inspect')}
                        </Button>
                        <Button
                          variant='outline'
                          size='sm'
                          onClick={() => openEdit(agent)}
                        >
                          {t('Edit')}
                        </Button>
                        <Button
                          variant='outline'
                          size='sm'
                          onClick={async () => {
                            const res = await createHostingAgentToken(agent.id, {
                              name: 'rotated',
                              allow_ips: tokenAllowIps,
                            })
                            if (res.success && res.data.secret) {
                              setSecret(res.data.secret)
                              toast.success(t('Token created'))
                              queryClient.invalidateQueries({
                                queryKey: ['hosting-tokens', agent.id],
                              })
                            } else {
                              toast.error(
                                res.message || t('Failed to create token')
                              )
                            }
                          }}
                        >
                          {t('Issue token')}
                        </Button>
                        <Button
                          variant='destructive'
                          size='sm'
                          onClick={async () => {
                            const res = await deleteHostingAgent(agent.id)
                            if (res.success) {
                              toast.success(t('Hosting agent deleted'))
                              queryClient.invalidateQueries({
                                queryKey: ['hosting-agents'],
                              })
                            }
                          }}
                        >
                          {t('Delete')}
                        </Button>
                      </div>
                    </CardHeader>
                  </Card>
                ))
              )}
            </TabsContent>
            <TabsContent value='incidents' className='space-y-3 pt-3'>
              {incidents.length === 0 ? (
                <p className='text-muted-foreground text-sm'>
                  {t('No hosting incidents')}
                </p>
              ) : (
                incidents.map((item) => (
                  <Card key={item.id}>
                    <CardHeader>
                      <CardTitle className='flex items-center gap-2'>
                        <Badge variant='outline'>{item.status}</Badge>
                        {item.summary}
                      </CardTitle>
                      <CardDescription>
                        {[item.source_event, item.brain_source, item.handoff_reason]
                          .filter(Boolean)
                          .join(' · ')}
                      </CardDescription>
                    </CardHeader>
                    {item.actions_json ? (
                      <CardContent className='text-muted-foreground text-sm whitespace-pre-wrap'>
                        {item.actions_json}
                      </CardContent>
                    ) : null}
                    <CardContent className='flex gap-2'>
                      <Button
                        size='sm'
                        variant='outline'
                        onClick={async () => {
                          await updateHostingIncident(item.id, 'auto_resolved')
                          queryClient.invalidateQueries({
                            queryKey: ['hosting-incidents'],
                          })
                        }}
                      >
                        {t('Confirm')}
                      </Button>
                      <Button
                        size='sm'
                        variant='outline'
                        onClick={async () => {
                          await updateHostingIncident(item.id, 'ignored')
                          queryClient.invalidateQueries({
                            queryKey: ['hosting-incidents'],
                          })
                        }}
                      >
                        {t('Ignore')}
                      </Button>
                    </CardContent>
                  </Card>
                ))
              )}
            </TabsContent>
            <TabsContent value='hooks' className='space-y-3 pt-3'>
              {hooks.length === 0 ? (
                <p className='text-muted-foreground text-sm'>
                  {t('No hosting hooks')}
                </p>
              ) : (
                hooks.map((hook) => (
                  <Card key={hook.id}>
                    <CardHeader className='flex flex-row items-center justify-between'>
                      <div>
                        <CardTitle>{hook.name}</CardTitle>
                        <CardDescription>
                          {hook.owner} · {hook.kind} · {hook.wake_mode}
                          {hook.cooldown_sec
                            ? ` · ${hook.cooldown_sec} ${t('seconds')}`
                            : ''}
                          {hook.next_fire_at
                            ? ` · ${new Date(hook.next_fire_at * 1000).toLocaleString()}`
                            : ''}
                        </CardDescription>
                      </div>
                      <div className='flex items-center gap-2'>
                      <Switch
                        checked={hook.enabled}
                        onCheckedChange={async (checked) => {
                          await updateHostingHook(hook.id, { enabled: checked })
                          queryClient.invalidateQueries({
                            queryKey: ['hosting-hooks'],
                          })
                        }}
                      />
                      {hook.owner !== 'system' ? (
                        <Button
                          variant='destructive'
                          size='sm'
                          onClick={async () => {
                            const res = await deleteHostingHook(hook.id)
                            if (res.success) {
                              toast.success(t('Hook deleted'))
                              queryClient.invalidateQueries({
                                queryKey: ['hosting-hooks'],
                              })
                            } else {
                              toast.error(
                                res.message || t('Failed to delete hook')
                              )
                            }
                          }}
                        >
                          {t('Delete')}
                        </Button>
                      ) : null}
                      </div>
                    </CardHeader>
                  </Card>
                ))
              )}
            </TabsContent>
            <TabsContent value='session' className='space-y-3 pt-3'>
              {selectedAgentId == null ? (
                <p className='text-muted-foreground text-sm'>
                  {t('Select an agent to inspect its session and cost')}
                </p>
              ) : (
                <>
                  <div className='flex flex-wrap items-center gap-2'>
                    <Button
                      variant='outline'
                      size='sm'
                      onClick={async () => {
                        await rotateHostingSession(selectedAgentId)
                        queryClient.invalidateQueries({
                          queryKey: ['hosting-session', selectedAgentId],
                        })
                        toast.success(t('New session started'))
                      }}
                    >
                      {t('Start new session')}
                    </Button>
                    <Button
                      variant='outline'
                      size='sm'
                      onClick={async () => {
                        const res = await exportHostingSession(selectedAgentId)
                        if (!res.success || !res.data?.export) {
                          toast.error(
                            res.message || t('Failed to export session')
                          )
                          return
                        }
                        const blob = new Blob([res.data.export], {
                          type: 'application/json',
                        })
                        const url = URL.createObjectURL(blob)
                        const link = document.createElement('a')
                        link.href = url
                        link.download = `hosting-session-${selectedAgentId}.json`
                        link.click()
                        URL.revokeObjectURL(url)
                        toast.success(t('Session exported'))
                      }}
                    >
                      {t('Export session')}
                    </Button>
                    <span className='text-muted-foreground text-sm'>
                      {t('Today')}: {costQuery.data?.data?.tokens_used ?? 0} /{' '}
                      {costQuery.data?.data?.daily_token_budget ??
                        selectedAgent?.daily_token_budget ??
                        0}{' '}
                      {t('tokens')} · {costQuery.data?.data?.wakes ?? 0} /{' '}
                      {costQuery.data?.data?.max_wakes_per_hour ??
                        selectedAgent?.max_wakes_per_hour ??
                        0}{' '}
                      {t('wakes')}
                      {sessionPayload?.token_occupancy != null ? (
                        <>
                          {' '}
                          · {t('Session')}: {sessionPayload.token_occupancy}{' '}
                          {t('tokens')}
                        </>
                      ) : null}
                    </span>
                  </div>
                  {(tokensQuery.data?.data ?? []).length > 0 ? (
                    <Card>
                      <CardHeader>
                        <CardTitle>{t('Tokens')}</CardTitle>
                      </CardHeader>
                      <CardContent className='space-y-2'>
                        {(tokensQuery.data?.data ?? []).map((token) => (
                          <div
                            key={token.id}
                            className='flex flex-wrap items-center justify-between gap-2 text-sm'
                          >
                            <span>
                              {token.name || t('Token')} · {token.token_prefix}{' '}
                              · {token.allow_ips || t('Any IP')}
                            </span>
                            <Button
                              variant='outline'
                              size='sm'
                              onClick={async () => {
                                const res = await rotateHostingAgentToken(
                                  selectedAgentId,
                                  token.id,
                                  { allow_ips: tokenAllowIps || token.allow_ips }
                                )
                                if (res.success && res.data.secret) {
                                  setSecret(res.data.secret)
                                  toast.success(t('Token rotated'))
                                  queryClient.invalidateQueries({
                                    queryKey: ['hosting-tokens', selectedAgentId],
                                  })
                                } else {
                                  toast.error(
                                    res.message || t('Failed to rotate token')
                                  )
                                }
                              }}
                            >
                              {t('Rotate token')}
                            </Button>
                          </div>
                        ))}
                      </CardContent>
                    </Card>
                  ) : null}
                  {sessionPayload?.last_compact_summary ||
                  selectedAgent?.last_compact_summary ? (
                    <Card>
                      <CardHeader>
                        <CardTitle>{t('Last compaction summary')}</CardTitle>
                      </CardHeader>
                      <CardContent className='whitespace-pre-wrap text-sm'>
                        {sessionPayload?.last_compact_summary ||
                          selectedAgent?.last_compact_summary}
                      </CardContent>
                    </Card>
                  ) : null}
                  <div className='space-y-2'>
                    {sessionEntries.map((entry) => (
                      <Card key={entry.id} size='sm'>
                        <CardHeader>
                          <CardTitle className='text-sm'>
                            {entry.role} {entry.name}
                          </CardTitle>
                          <CardDescription className='whitespace-pre-wrap'>
                            {entry.content}
                          </CardDescription>
                        </CardHeader>
                      </Card>
                    ))}
                  </div>
                </>
              )}
            </TabsContent>
          </Tabs>
        </div>
      </SectionPageLayout.Content>

      <AgentEditorDialog
        open={dialogOpen}
        editing={editing}
        form={form}
        permissions={permissions}
        catalog={catalog}
        template={template}
        secret={secret}
        saving={saveMutation.isPending}
        onOpenChange={setDialogOpen}
        onFormChange={setForm}
        onPermissionsChange={setPermissions}
        onSave={() => saveMutation.mutate()}
      />
    </SectionPageLayout>
  )
}
