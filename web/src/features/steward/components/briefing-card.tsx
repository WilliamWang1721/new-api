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

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

type BriefingCardProps = {
  text: string
  onRefresh: () => void
  busy: boolean
}

export function BriefingCard(props: BriefingCardProps) {
  const { t } = useTranslation()
  if (!props.text) {
    return null
  }
  return (
    <Card size='sm'>
      <CardHeader className='flex flex-row items-start justify-between gap-2'>
        <CardTitle className='text-sm'>{t('Status briefing')}</CardTitle>
        <Button
          variant='outline'
          size='sm'
          disabled={props.busy}
          onClick={props.onRefresh}
        >
          {t('Refresh briefing')}
        </Button>
      </CardHeader>
      <CardContent className='text-sm whitespace-pre-wrap'>
        {props.text}
      </CardContent>
    </Card>
  )
}
