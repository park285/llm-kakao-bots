import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import * as z from 'zod'
import { Button, Input, Form, FormControl, FormField, FormItem, FormLabel, FormMessage, BaseModal } from '@/components/ui'
import { Save, Youtube } from 'lucide-react'

// Schema 정의
const channelEditSchema = z.object({
  channelId: z.string().trim().min(24, '채널 ID 형식이 올바르지 않습니다 (최소 24자).'),
})

type ChannelEditFormValues = z.infer<typeof channelEditSchema>

interface ChannelEditModalProps {
  isOpen: boolean
  onClose: () => void
  onSave: (newChannelId: string) => void
  memberId: number
  memberName: string
  currentChannelId: string
}

export default function ChannelEditModal({
  isOpen,
  onClose,
  onSave,
  memberId,
  memberName,
  currentChannelId,
}: ChannelEditModalProps) {
  const form = useForm<ChannelEditFormValues>({
    resolver: zodResolver(channelEditSchema),
    defaultValues: {
      channelId: currentChannelId,
    },
  })

  // 모달이 열릴 때마다 form 리셋
  useEffect(() => {
    if (isOpen) {
      form.reset({ channelId: currentChannelId })
    }
  }, [isOpen, currentChannelId, form])

  const onSubmit = (data: ChannelEditFormValues) => {
    onSave(data.channelId)
    onClose()
  }

  const title = (
    <span className="flex items-center gap-2">
      <Youtube className="text-red-600" size={20} />
      채널 ID 수정
    </span>
  )

  return (
    <BaseModal isOpen={isOpen} onClose={onClose} title={title} showHeaderBorder>
      <Form {...form}>
        <form onSubmit={(e) => void form.handleSubmit(onSubmit)(e)} className="space-y-4">
          <div className="bg-slate-50 p-3 rounded-lg border border-slate-100 mb-4 space-y-2">
            <div className="flex justify-between text-sm">
              <span className="text-slate-500">멤버 이름</span>
              <span className="font-bold text-slate-800">{memberName}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-slate-500">멤버 ID</span>
              <span className="font-mono text-slate-600">{memberId}</span>
            </div>
          </div>

          <FormField
            control={form.control}
            name="channelId"
            render={({ field }) => (
              <FormItem>
                <FormLabel>YouTube 채널 ID</FormLabel>
                <FormControl>
                  <Input placeholder="UC..." className="font-mono" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <div className="mt-6 flex justify-end gap-3 pt-2">
            <Button
              type="button"
              variant="outline"
              onClick={onClose}
            >
              취소
            </Button>
            <Button
              type="submit"
              disabled={!form.formState.isDirty}
              className="gap-2"
            >
              <Save size={16} /> 저장
            </Button>
          </div>
        </form>
      </Form>
    </BaseModal>
  )
}
