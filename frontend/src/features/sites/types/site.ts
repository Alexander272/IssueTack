export interface ISite {
	id: string
	name: string
	address: string
	createdAt: string
	updatedAt: string
}

export interface ISiteDTO {
	id: string | null
	name: string
	address: string
}

// Расположение внутри площадки (комната, стеллаж, склад). Заводится и правится
// только в карточке площадки; формы оборудования выбирают его из списка.
export interface ISiteLocation {
	id: string
	siteId: string
	name: string
	createdAt: string
	updatedAt: string
}

export interface ISiteLocationDTO {
	name: string
}
