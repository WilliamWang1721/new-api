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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Textarea } from '@/components/ui/textarea'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import {
  decideStewardApproval,
  getStewardBriefing,
  getStewardSession,
  getStewardStatus,
  listStewardApprovals,
  postStewardChat,
  rotateStewardSession,
} from './api'
import { ApprovalCard } from './components/approval-card'
import { BriefingCard } from './components/briefing-card'
import type { StewardSessionEntry } from './types'

const PROMPT_KEYS = [
  'What needs attention right now?',
  'List channels that are down',
  'Help a user who needs more quota',
  'Walk me through first-time setup',
] as const

export function StewardPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const role = useAuthStore((state) => state.auth.user?.role)
  const canReview = (role ?? 0) >= ROLE.ADMIN
  const isRoot = role === ROLE.SUPER_ADMIN
  const [draft, setDraft] = useState('')

  const statusQuery = useQuery({
    queryKey: ['steward-status'],
    queryFn: getStewardStatus,
  })
  const sessionQuery = useQuery({
    queryKey: ['steward-session'],
    queryFn: getStewardSession,
  })
  const approvalsQuery = useQuery({
    queryKey: ['steward-approvals'],
    queryFn: () => listStewardApprovals('pending'),
  })
  const briefingQuery = useQuery({
    queryKey: ['steward-briefing'],
    queryFn: () => getStewardBriefing(false),
  })

  const status = statusQuery.data?.data
  const entries = sessionQuery.data?.data?.entries ?? []
  const approvals = approvalsQuery.data?.data ?? []
  const visibleEntries = entries.filter(
    (entry) => entry.role === 'user' || entry.role === 'assistant'
  )

  const sendMutation = useMutation({
    mutationFn: (message: string) => postStewardChat(message),
    onSuccess: (res) => {
      if (!res.success) {
        toast.error(res.message || t('The steward could not reply'))
        return
      }
      if (res.data.needs_brain) {
        toast.error(t('Set up an AI model in Steward Settings first.'))
      }
      queryClient.invalidateQueries({ queryKey: ['steward-session'] })
      queryClient.invalidateQueries({ queryKey: ['steward-approvals'] })
      queryClient.invalidateQueries({ queryKey: ['steward-status'] })
      setDraft('')
    },
    onError: () => {
      toast.error(t('The steward could not reply'))
    },
  })

  const send = (text: string) => {
    const message = text.trim()
    if (!message || sendMutation.isPending) return
    sendMutation.mutate(message)
  }

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>
        <span className='inline-flex min-w-0 items-center gap-2'>
          <span className='truncate'>{t('AI Steward')}</span>
          <Badge variant='outline' className='shrink-0'>
            {status?.ready ? t('Ready') : t('Not ready')}
          </Badge>
        </span>
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <div className='flex flex-wrap gap-2'>
          <Button
            variant='outline'
            size='sm'
            onClick={async () => {
              const res = await rotateStewardSession()
              if (res.success) {
                toast.success(t('New conversation started'))
                queryClient.invalidateQueries({ queryKey: ['steward-session'] })
              }
            }}
          >
            {t('New conversation')}
          </Button>
          {isRoot ? (
            <Button variant='outline' size='sm' render={<Link to='/hosting' />}>
              {t('Steward Settings')}
            </Button>
          ) : null}
        </div>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='flex h-full min-h-0 flex-col gap-3'>
          {status?.needs_brain ? (
            <Alert>
              <AlertTitle>{t('Set up the steward')}</AlertTitle>
              <AlertDescription>
                {t(
                  'Choose an AI model in Steward Settings, then come back and talk in plain language.'
                )}
              </AlertDescription>
            </Alert>
          ) : null}
          <BriefingCard
            text={briefingQuery.data?.data?.text ?? ''}
            busy={briefingQuery.isFetching}
            onRefresh={async () => {
              const res = await getStewardBriefing(true)
              if (!res.success) {
                toast.error(res.message || t('The steward could not reply'))
                return
              }
              queryClient.setQueryData(['steward-briefing'], res)
            }}
          />
          {approvals.length > 0 ? (
            <div className='grid gap-2'>
              {approvals.map((item) => (
                <ApprovalCard
                  key={item.id}
                  item={item}
                  canReview={canReview}
                  busy={false}
                  onDecide={async (id, approve) => {
                    const res = await decideStewardApproval(id, approve)
                    if (!res.success) {
                      toast.error(res.message || t('Failed to review request'))
                      return
                    }
                    toast.success(
                      approve ? t('Request approved') : t('Request denied')
                    )
                    queryClient.invalidateQueries({
                      queryKey: ['steward-approvals'],
                    })
                  }}
                />
              ))}
            </div>
          ) : null}
          <ScrollArea className='min-h-0 flex-1 rounded-md border'>
            <div className='space-y-3 p-3'>
              {visibleEntries.length === 0 ? (
                <div className='space-y-3'>
                  <p className='text-muted-foreground text-sm'>
                    {t(
                      'Ask the steward to check the API, change a setting, or review a user request.'
                    )}
                  </p>
                  <div className='flex flex-wrap gap-2'>
                    {PROMPT_KEYS.map((key) => (
                      <Button
                        key={key}
                        variant='outline'
                        size='sm'
                        onClick={() => send(t(key))}
                      >
                        {t(key)}
                      </Button>
                    ))}
                  </div>
                </div>
              ) : (
                visibleEntries.map((entry) => (
                  <ChatBubble key={entry.id} entry={entry} />
                ))
              )}
            </div>
          </ScrollArea>
          <div className='flex gap-2'>
            <Textarea
              value={draft}
              placeholder={t('Tell the steward what you need…')}
              className='min-h-16'
              onChange={(event) => setDraft(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter' && !event.shiftKey) {
                  event.preventDefault()
                  send(draft)
                }
              }}
            />
            <Button
              className='self-end'
              disabled={sendMutation.isPending || !draft.trim()}
              onClick={() => send(draft)}
            >
              {sendMutation.isPending ? t('Thinking…') : t('Send')}
            </Button>
          </div>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

function ChatBubble(props: { entry: StewardSessionEntry }) {
  const isUser = props.entry.role === 'user'
  return (
    <div className={isUser ? 'ml-8' : 'mr-8'}>
      <div
        className={
          isUser
            ? 'bg-primary text-primary-foreground rounded-lg px-3 py-2 text-sm'
            : 'bg-muted rounded-lg px-3 py-2 text-sm whitespace-pre-wrap'
        }
      >
        {props.entry.content}
      </div>
    </div>
  )
}
