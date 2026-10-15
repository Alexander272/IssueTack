import type { ICategory } from '../types/category'

// Эффективная скрытость категории: собственная скрытость (isHidden) либо
// нахождение в скрытом разделе — заказчик такую категорию не видит.
export const isCategoryHidden = (cat: ICategory) => cat.isHidden || Boolean(cat.categoryGroup?.isHidden)

// Подпись к бейджу «скрытая»: отдельно помечаем случай, когда скрытость
// приходит от раздела, а не от самой категории.
export const categoryHiddenLabel = (cat: ICategory) => (cat.isHidden ? 'Скрыта' : 'Скрыта разделом')
