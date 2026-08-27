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
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

import type { StewardApproval } from '../types'

type ApprovalCardProps = {
  item: StewardApproval
  canReview: boolean
  busy: boolean
  onDecide: (id: number, approve: boolean) => void
}

export function ApprovalCard(props: ApprovalCardProps) {
  const { t } = useTranslation()

  return (
    <Card size='sm'>
      <CardHeader className='space-y-1'>
        <CardTitle className='text-sm'>{props.item.summary}</CardTitle>
        <p className='text-muted-foreground text-xs'>
          {t('Needs confirmation')}
          {props.item.reason ? ` · ${props.item.reason}` : ''}
        </p>
      </CardHeader>
      {props.canReview ? (
        <CardContent className='flex gap-2'>
          <Button
            size='sm'
            disabled={props.busy}
            onClick={() => props.onDecide(props.item.id, true)}
          >
            {t('Approve')}
          </Button>
          <Button
            size='sm'
            variant='outline'
            disabled={props.busy}
            onClick={() => props.onDecide(props.item.id, false)}
          >
            {t('Deny')}
          </Button>
        </CardContent>
      ) : (
        <CardContent className='text-muted-foreground text-sm'>
          {t('An admin will review this request.')}
        </CardContent>
      )}
    </Card>
  )
}
