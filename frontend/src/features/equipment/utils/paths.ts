import { generatePath } from 'react-router'

import { AppRoutes } from '@/pages/router/routes'

// Путь карточки единицы. Маршрут объявлен ровно один раз — AppRoutes.EquipmentDetail,
// поэтому id подставляется в шаблон, а не склеивается со строкой вручную.
export const equipmentDetailPath = (id: string) => generatePath(AppRoutes.EquipmentDetail, { id })
