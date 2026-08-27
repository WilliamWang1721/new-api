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
  getHostingCost,
  getHostingPermissionCatalog,
  getHostingSession,
  getHostingStatus,
  listHostingAgents,
  listHostingHooks,
  listHostingIncidents,
  rotateHostingSession,
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
  const [selectedAgentId, setSelectedAgentId] = useState<number | null>(null)

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
        <Button onClick={openCreate}>{t('Create agent')}</Button>
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

          <Card>
            <CardHeader>
              <CardTitle>{t('Runtime snapshot')}</CardTitle>
              <CardDescription>
                {t('Non-AI monitoring summary. Auto-disabled channels: {{count}}', {
                  count: snapshot?.auto_disabled_count ?? 0,
                })}
              </CardDescription>
            </CardHeader>
            <CardContent className='text-muted-foreground text-sm'>
              {t('Hosting state')}: {hostingState}
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
                            })
                            if (res.success && res.data.secret) {
                              setSecret(res.data.secret)
                              toast.success(t('Token created'))
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
                      <CardDescription>{item.handoff_reason}</CardDescription>
                    </CardHeader>
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
                        </CardDescription>
                      </div>
                      <Switch
                        checked={hook.enabled}
                        onCheckedChange={async (checked) => {
                          await updateHostingHook(hook.id, { enabled: checked })
                          queryClient.invalidateQueries({
                            queryKey: ['hosting-hooks'],
                          })
                        }}
                      />
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
                    <span className='text-muted-foreground text-sm'>
                      {t('Today')}: {costQuery.data?.data?.tokens_used ?? 0}{' '}
                      {t('tokens')} / {costQuery.data?.data?.wakes ?? 0}{' '}
                      {t('wakes')}
                    </span>
                  </div>
                  <div className='space-y-2'>
                    {(sessionQuery.data?.data ?? []).map((entry) => (
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
