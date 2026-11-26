import { useEffect, useMemo, useRef, type FC } from 'react'
import { Autocomplete, Box, Chip, TextField, Typography, type SxProps } from '@mui/material'
import { Controller, useFormContext, useWatch } from 'react-hook-form'

import type { FormValues } from './types'
import type { ICategoryModifier } from '@/features/categories/types/category'
import { fieldLabelSx } from './styles'
import { useAppSelector } from '@/hooks/redux'
import { getRealm } from '@/features/realms/realmSlice'
import { useDebounce } from '@/hooks/useDebounce'
import { useGetAllCategoriesQuery, useGetAllCategoryModifiersQuery } from '@/features/categories/categoriesApiSlice'
import { useGetProductionCalendarQuery } from '@/features/calendar/calendarApiSlice'
import { usePreviewModifierQuery } from '../../tasksApiSlice'
import { autoDueDateISO } from '../../utils/workDays'
import { useTaskFormSettings } from './taskFormMode'

/**
 * Подкатегория лежит сразу под категорией (в секции «Что случилось и где»):
 * это уточнение категории, а не расширенная настройка. Показ задаёт форма
 * через canPickModifier: и при создании, и при редактировании — исполнитель
 * (участник группы реалма) или начальник области; на создании право не зависит
 * от выбранного assignee. Остальные правила (сброс при смене категории, разовая
 * подстановка срока/приоритета) — серверные, форма лишь повторяет их, чтобы
 * пользователь видел результат до сохранения.
 *
 * На создании подкатегорию может выбрать и сервер — по ключевым словам заголовка
 * и описания. Предпросмотр дёргает тот же серверный подбор, но не пишет поле формы:
 * пока пользователь не нажал на подсказку, явный выбор остаётся пустым. По клику
 * подсказка фиксируется как обычный выбор и подставляет срок/приоритет.
 * Показывается только тем, у кого есть право ручного выбора: остальным в форме
 * не нужны ни селект, ни подсказка (сервер всё равно применит подбор при сохранении).
 */
export const CategoryModifierField: FC<{ sx?: SxProps }> = ({ sx }) => {
	const { canPickModifier, dueDateBase, mode } = useTaskFormSettings()
	const { control, setValue } = useFormContext<FormValues>()
	const realm = useAppSelector(getRealm)
	const { data: categoriesData } = useGetAllCategoriesQuery(undefined, { skip: !canPickModifier })
	const { data: modifiersData } = useGetAllCategoryModifiersQuery(undefined, { skip: !canPickModifier })
	const { data: calendarData } = useGetProductionCalendarQuery(undefined, { skip: !canPickModifier })

	const categories = useMemo(() => categoriesData?.data ?? [], [categoriesData])
	const allModifiers = useMemo(() => modifiersData?.data ?? [], [modifiersData])
	const calendar = calendarData?.data
	const selectedCategoryId = useWatch({ control, name: 'categoryId' })
	const selectedModifierId = useWatch({ control, name: 'categoryModifierId' })
	const title = useWatch({ control, name: 'title' })
	const description = useWatch({ control, name: 'description' })

	const modifiers = useMemo(
		() => allModifiers.filter(m => m.isActive && m.categoryId === selectedCategoryId),
		[allModifiers, selectedCategoryId],
	)

	// Подкатегория принадлежит категории: при смене категории бэкенд принудительно
	// сбрасывает её, поэтому форма делает то же самое (иначе в DTO уехал бы чужой id).
	// Защита prev-ссылкой нужна, чтобы восстановленный черновик/редактируемая заявка
	// не затирали уже выбранное значение на первом рендере.
	const prevModifierCategoryId = useRef(selectedCategoryId)
	useEffect(() => {
		if (prevModifierCategoryId.current === selectedCategoryId) return
		prevModifierCategoryId.current = selectedCategoryId
		setValue('categoryModifierId', null)
	}, [selectedCategoryId, setValue])

	// Явный выбор подкатегории: из селекта или кликом по подсказке. Подстановка
	// срока и приоритета живёт прямо здесь, а не в useEffect — так она не зависит
	// от порядка загрузки справочников и не срабатывает повторно при их refetch
	// (иначе затирались бы уже введённые пользователем значения). shouldDirty
	// обязателен: форма редактирования включает «Сохранить» через isDirty.
	const handleSelectModifier = (modifier: ICategoryModifier | null) => {
		setValue('categoryModifierId', modifier?.id ?? null, { shouldDirty: true })
		if (!modifier) return

		const category = categories.find(c => c.id === selectedCategoryId)
		setValue('priority', modifier.priority ?? category?.priority ?? 'medium', { shouldDirty: true })
		const days = modifier.defaultDueDays ?? category?.defaultDueDays
		setValue('dueDate', autoDueDateISO(days, calendar, dueDateBase ?? null), { shouldDirty: true })
	}

	// Дебаунсим только текст: смена категории — дискретный клик, ей не нужна задержка.
	const debouncedTitle = useDebounce(title ?? '', 350)
	const debouncedDescription = useDebounce(description ?? '', 350)

	const canPreview =
		canPickModifier &&
		mode === 'create' &&
		!!realm?.id &&
		!!selectedCategoryId &&
		modifiers.length > 0 &&
		!selectedModifierId &&
		!!(debouncedTitle.trim() || debouncedDescription.trim())

	const { data: previewData } = usePreviewModifierQuery(
		{ categoryId: selectedCategoryId, title: debouncedTitle, description: debouncedDescription },
		{ skip: !canPreview },
	)

	const preview = previewData?.data ?? null
	const showManual = canPickModifier && modifiers.length > 0
	// canPreview гасит чип и при очистке текста: RTK Query может ещё держать
	// закешированный ответ прошлого запроса, а подсказка без текста неактуальна.
	const showPreview = canPreview && !!preview
	if (!showManual && !showPreview) return null

	const applyPreview = () => {
		const modifier = modifiers.find(m => m.id === preview?.id)
		if (modifier) handleSelectModifier(modifier)
	}

	return (
		<Box sx={sx}>
			<Typography variant='caption' sx={fieldLabelSx}>
				Подкатегория
			</Typography>
			{showManual && (
				<>
					<Controller
						control={control}
						name='categoryModifierId'
						render={({ field }) => (
							<Autocomplete
								options={modifiers}
								getOptionLabel={m => m.name}
								isOptionEqualToValue={(option, value) => option.id === value.id}
								value={modifiers.find(m => m.id === field.value) ?? null}
								onChange={(_, value) => handleSelectModifier(value)}
								noOptionsText='Нет подкатегорий'
								renderInput={params => (
									<TextField {...params} size='small' placeholder='Без подкатегории' />
								)}
							/>
						)}
					/>
					{!showPreview && (
						<Typography variant='caption' sx={{ color: 'text.secondary', mt: 0.5, display: 'block' }}>
							При выборе срок и приоритет подставятся из подкатегории
						</Typography>
					)}
				</>
			)}
			{showPreview && (
				<Box
					sx={{ display: 'flex', alignItems: 'center', flexWrap: 'wrap', gap: 1, mt: showManual ? 0.75 : 0 }}
				>
					<Chip
						label={preview.name}
						size='small'
						color='primary'
						variant='outlined'
						onClick={applyPreview}
						sx={{ fontWeight: 500 }}
					/>
					<Typography variant='caption' sx={{ color: 'text.secondary' }}>
						подберётся автоматически (нажмите, чтобы выбрать)
					</Typography>
				</Box>
			)}
		</Box>
	)
}
