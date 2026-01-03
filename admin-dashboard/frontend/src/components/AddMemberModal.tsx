import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import * as z from 'zod'
import { Button, Input, Form, FormControl, FormField, FormItem, FormLabel, FormMessage, BaseModal } from '@/components/ui'
import { UserPlus, Save } from 'lucide-react'

// Schema 정의
const addMemberSchema = z.object({
    name: z.string().trim().min(1, '멤버 이름을 입력해주세요.'),
    channelId: z.string().trim().min(24, 'ID 형식이 올바르지 않습니다 (최소 24자).'),
    nameKo: z.string().trim().optional(),
    nameJa: z.string().trim().optional(),
})

type AddMemberFormValues = z.infer<typeof addMemberSchema>

interface AddMemberModalProps {
    isOpen: boolean
    onClose: () => void
    onAdd: (member: AddMemberFormValues) => void
}

export default function AddMemberModal({
    isOpen,
    onClose,
    onAdd,
}: AddMemberModalProps) {
    const form = useForm<AddMemberFormValues>({
        resolver: zodResolver(addMemberSchema),
        defaultValues: {
            name: '',
            channelId: '',
            nameKo: '',
            nameJa: '',
        },
    })

    // 모달이 열릴 때마다 form 리셋
    useEffect(() => {
        if (isOpen) {
            form.reset({
                name: '',
                channelId: '',
                nameKo: '',
                nameJa: '',
            })
        }
    }, [isOpen, form])

    const onSubmit = (data: AddMemberFormValues) => {
        onAdd(data)
        onClose()
    }

    const title = (
        <span className="flex items-center gap-2">
            <UserPlus className="text-sky-600" size={20} />
            새 멤버 추가
        </span>
    )

    return (
        <BaseModal isOpen={isOpen} onClose={onClose} title={title} maxWidth="lg" showHeaderBorder>
            <Form {...form}>
                <form onSubmit={(e) => void form.handleSubmit(onSubmit)(e)} className="space-y-4">

                    <FormField
                        control={form.control}
                        name="name"
                        render={({ field }) => (
                            <FormItem>
                                <FormLabel>멤버 이름 (기본)</FormLabel>
                                <FormControl>
                                    <Input placeholder="예: Hoshimachi Suisei" {...field} />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        )}
                    />

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

                    <div className="grid grid-cols-2 gap-4">
                        <FormField
                            control={form.control}
                            name="nameKo"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel className="text-slate-500">한국어 이름 (선택)</FormLabel>
                                    <FormControl>
                                        <Input placeholder="예: 호시마치 스이세이" {...field} />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={form.control}
                            name="nameJa"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel className="text-slate-500">일본어 이름 (선택)</FormLabel>
                                    <FormControl>
                                        <Input placeholder="예: 星街すいせい" {...field} />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                    </div>

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
                            className="gap-2 bg-sky-600 hover:bg-sky-700"
                        >
                            <Save size={16} /> 추가하기
                        </Button>
                    </div>
                </form>
            </Form>
        </BaseModal>
    )
}
