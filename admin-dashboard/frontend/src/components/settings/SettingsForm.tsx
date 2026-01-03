import { useEffect } from 'react'
import { useForm, type Resolver, type SubmitHandler, type FieldErrors } from 'react-hook-form'
import * as z from 'zod'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { settingsApi } from '@/api'
import { queryKeys } from '@/api/queryKeys'
import type { SettingsResponse } from '@/types'
import { Card, Button, Input, Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui'
import { Settings as SettingsIcon, Save, Loader2, Check } from 'lucide-react'
import toast from 'react-hot-toast'

// Settings Schema Definition
const settingsSchema = z.object({
    alarmAdvanceMinutes: z.coerce.number()
        .min(1, { message: "최소 1분 이상이어야 합니다." })
        .max(60, { message: "최대 60분까지만 설정 가능합니다." }),
})

type SettingsFormValues = z.infer<typeof settingsSchema>

const settingsFormResolver: Resolver<SettingsFormValues> = (values) => {
    const parsed = settingsSchema.safeParse(values)
    if (parsed.success) {
        return { values: parsed.data, errors: {} }
    }

    const errors: FieldErrors<SettingsFormValues> = {}
    for (const issue of parsed.error.issues) {
        if (issue.path[0] === 'alarmAdvanceMinutes') {
            errors.alarmAdvanceMinutes = {
                type: issue.code,
                message: issue.message,
            }
        }
    }

    return { values: {}, errors }
}

interface SettingsFormProps {
    initialData?: SettingsResponse
}

export const SettingsForm = ({ initialData }: SettingsFormProps) => {
    const queryClient = useQueryClient()

    const { data: settingsData } = useQuery({
        queryKey: queryKeys.settings.all,
        queryFn: settingsApi.get,
        initialData,
    })

    // 초기값 타입을 안전하게 처리
    const defaultAlarmMinutes = settingsData?.settings?.alarmAdvanceMinutes ?? initialData?.settings?.alarmAdvanceMinutes ?? 5

    const form = useForm<SettingsFormValues>({
        resolver: settingsFormResolver,
        defaultValues: {
            alarmAdvanceMinutes: defaultAlarmMinutes,
        },
    })

    const { isDirty } = form.formState
    const { reset } = form

    // 서버 데이터로 폼 동기화 (단, 사용자가 수정 중이 아닐 때만)
    useEffect(() => {
        if (settingsData?.settings && !isDirty) {
            reset({
                alarmAdvanceMinutes: settingsData.settings.alarmAdvanceMinutes,
            })
        }
    }, [settingsData, reset, isDirty])

    const updateMutation = useMutation({
        mutationFn: settingsApi.update,
        onSuccess: (_, variables) => {
            void queryClient.invalidateQueries({ queryKey: queryKeys.settings.all })
            reset(variables)
            toast.success('설정이 성공적으로 저장되었습니다.')
        },
        onError: (err: Error) => {
            toast.error(`설정 저장 실패: ${err.message}`)
        }
    })

    const onSubmit: SubmitHandler<SettingsFormValues> = (data) => {
        updateMutation.mutate(data)
    }

    return (
        <Card>
            <Card.Header className="flex flex-row items-center gap-2 border-b border-slate-100 pb-4">
                <SettingsIcon className="text-slate-600" size={20} />
                <h3 className="text-lg font-bold text-slate-800">시스템 설정</h3>
            </Card.Header>

            <Card.Body className="space-y-6 pt-6">
                <Form {...form}>
                    <form
                        onSubmit={(event) => { void form.handleSubmit(onSubmit)(event) }}
                        className="space-y-6"
                    >
                        <div>
                            <h4 className="text-sm font-bold text-slate-900 mb-4 border-l-2 border-primary pl-3">알림 옵션</h4>

                            <div className="bg-slate-50 rounded-lg p-5 border border-slate-100 hover:border-slate-200 transition-colors">
                                <FormField
                                    control={form.control}
                                    name="alarmAdvanceMinutes"
                                    render={({ field }) => (
                                        <FormItem>
                                            <FormLabel>알람 사전 알림 시간</FormLabel>
                                            <div className="flex items-center gap-3">
                                                <FormControl>
                                                    <Input
                                                        type="number"
                                                        className="w-24 bg-white text-center font-medium"
                                                        {...field}
                                                    />
                                                </FormControl>
                                                <span className="text-sm font-medium text-slate-600">분 전 알림</span>
                                            </div>
                                            <FormDescription>
                                                방송 시작 몇 분 전에 채팅방으로 알람을 전송할지 설정합니다.
                                            </FormDescription>
                                            <FormMessage />
                                        </FormItem>
                                    )}
                                />
                            </div>
                        </div>

                        <div className="flex justify-end pt-2">
                            <Button
                                type="submit"
                                disabled={!isDirty || updateMutation.isPending}
                                className="gap-2"
                            >
                                {updateMutation.isPending ? (
                                    <Loader2 size={16} className="animate-spin" />
                                ) : isDirty ? (
                                    <Save size={16} />
                                ) : (
                                    <Check size={16} />
                                )}
                                {updateMutation.isPending ? '저장 중...' : isDirty ? '변경 사항 저장' : '저장됨'}
                            </Button>
                        </div>
                    </form>
                </Form>
            </Card.Body>
        </Card>
    )
}
