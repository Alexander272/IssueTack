import { useEffect, useMemo, useRef, type FC } from 'react'
import { Autocomplete, Box, TextField, Typography, type SxProps } from '@mui/material'
import { Controller, useFormContext, useWatch } from 'react-hook-form'
import dayjs from 'dayjs'

import type { FormValues } from './types'
import { fieldLabelSx } from './styles'
import { useGetAllCategoriesQuery, useGetAllCategoryModifiersQuery } from '@/features/categories/categoriesApiSlice'
import { useTaskFormSettings } from './taskFormMode'

/**
 * Подкатегория лежит сразу под категорией (в секции «Что случилось и где»):
 * это уточнение категории, а не расширенная настройка. Показ задаёт форма
 * через canPickModifier: и при создании, и при редактировании — исполнитель
 * (участник группы реалма) или начальник области; на создании право не зависит
 * от выбранного assignee. Остальные правила (сброс при смене категории, разовая
 * подстановка срока/приоритета) — серверные, форма лишь повторяет их, чтобы
 * пользователь видел результат до сохранения.
 */
export const CategoryModifierField: FC<{ sx?: SxProps }> = ({ sx }) => {
	const { canPickModifier, dueDateBase } = useTaskFormSettings()
	const { control, setValue } = useFormContext<FormValues>()
	const { data: categoriesData } = useGetAllCategoriesQuery()
	const { data: modifiersData } = useGetAllCategoryModifiersQuery()

	const categories = useMemo(() => categoriesData?.data ?? [], [categoriesData])
	const allModifiers = useMemo(() => modifiersData?.data ?? [], [modifiersData])
	const selectedCategoryId = useWatch({ control, name: 'categoryId' })
	const selectedModifierId = useWatch({ control, name: 'categoryModifierId' })

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

	// Разовая подстановка при выборе подкатегории: срок и приоритет берутся из неё,
	// а пустые значения наследуются от категории. Отсчёт срока — от даты создания
	// заявки (в форме редактирования), на создании — от сейчас. Явный выбор
	// пользователя после этого остаётся в форме (у ролей с правом переопределения
	// бэкенд его сохранит).
	const prevModifierId = useRef(selectedModifierId)
	useEffect(() => {
		if (prevModifierId.current === selectedModifierId) return
		prevModifierId.current = selectedModifierId
		if (!selectedModifierId) return
		const modifier = allModifiers.find(m => m.id === selectedModifierId)
		if (!modifier) return
		const category = categories.find(c => c.id === selectedCategoryId)
		setValue('priority', modifier.priority ?? category?.priority ?? 'medium')
		const days = modifier.defaultDueDays ?? category?.defaultDueDays
		const base = dueDateBase ? dayjs(dueDateBase) : dayjs()
		setValue('dueDate', days != null ? base.add(days, 'day').endOf('day').toISOString() : null)
	}, [selectedModifierId, allModifiers, categories, selectedCategoryId, dueDateBase, setValue])

	if (!canPickModifier || modifiers.length === 0) return null

	return (
		<Box sx={sx}>
			<Typography variant='caption' sx={fieldLabelSx}>
				Подкатегория
			</Typography>
			<Controller
				control={control}
				name='categoryModifierId'
				render={({ field }) => (
					<Autocomplete
						options={modifiers}
						getOptionLabel={m => m.name}
						value={modifiers.find(m => m.id === field.value) ?? null}
						onChange={(_, value) => field.onChange(value?.id ?? null)}
						noOptionsText='Нет подкатегорий'
						renderInput={params => <TextField {...params} size='small' placeholder='Без подкатегории' />}
					/>
				)}
			/>
			<Typography variant='caption' sx={{ color: 'text.secondary', mt: 0.5, display: 'block' }}>
				При выборе срок и приоритет подставятся из подкатегории
			</Typography>
		</Box>
	)
}
