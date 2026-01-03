import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import * as z from 'zod'
import { Button, Input, Form, FormControl, FormField, FormItem, FormLabel, FormMessage, BaseModal } from '@/components/ui'
import { Save, AlertTriangle } from 'lucide-react'

// Schema 정의
const editNameSchema = z.object({
  name: z.string().trim().min(1, '이름을 입력해주세요.'),
})

type EditNameFormValues = z.infer<typeof editNameSchema>

interface EditNameModalProps {
  isOpen: boolean
  onClose: () => void
  onSave: (newName: string) => void
  type: 'room' | 'user' | 'member'
  id: string
  currentName: string
}

export default function EditNameModal({
  isOpen,
  onClose,
  onSave,
  type,
  id,
  currentName,
}: EditNameModalProps) {
  const form = useForm<EditNameFormValues>({
    resolver: zodResolver(editNameSchema),
    defaultValues: {
      name: currentName,
    },
  })

  // 모달이 열릴 때마다 form 리셋
  useEffect(() => {
    if (isOpen) {
      form.reset({ name: currentName })
    }
  }, [isOpen, currentName, form])

  const onSubmit = (data: EditNameFormValues) => {
    onSave(data.name)
    onClose()
  }

  const getTitle = () => {
    switch (type) {
      case 'room': return '방 이름 수정'
      case 'user': return '사용자 이름 수정'
      case 'member': return '멤버 이름 수정'
      default: return '이름 수정'
    }
  }

  const showIdWarning = type !== 'member' && /^\d+$/.test(id)

  return (
    <BaseModal isOpen={isOpen} onClose={onClose} title={getTitle()} showHeaderBorder>
      <Form {...form}>
        <form onSubmit={(e) => void form.handleSubmit(onSubmit)(e)} className="space-y-4">
          <div className="bg-slate-50 p-3 rounded-lg border border-slate-100 mb-4">
            <div className="text-xs text-slate-500 font-medium mb-1">ID (변경 불가)</div>
            <div className="text-sm font-mono text-slate-700">{id}</div>
          </div>

          {showIdWarning && (
            <div className="bg-amber-50 p-3 rounded-lg border border-amber-100 flex items-start gap-2 mb-4">
              <AlertTriangle size={16} className="text-amber-500 mt-0.5 shrink-0" />
              <div className="text-xs text-amber-700 leading-snug">
                현재 ID가 사용 중입니다. 이름을 설정하면 ID 대신 이름이 표시됩니다.
              </div>
            </div>
          )}

          <FormField
            control={form.control}
            name="name"
            render={({ field }) => (
              <FormItem>
                <FormLabel>새로운 이름</FormLabel>
                <FormControl>
                  <Input placeholder="이름을 입력하세요" {...field} />
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
