import { useState, type FC } from 'react'
import { Box, Button, Typography, useTheme } from '@mui/material'

import type { ISite, ISiteDTO } from '../types/site'
import { useGetAllSitesQuery } from '../sitesApiSlice'
import { PlusIcon } from '@/components/Icons/PlusIcon'
import { SitesTable } from '../components/SitesTable'
import { SiteCardList } from '../components/SiteCardList'
import { SiteViewDialog } from '../components/SiteViewDialog'
import { SiteDialog } from '../components/SiteDialog'

export const SitesView: FC = () => {
	const { palette } = useTheme()

	const { data } = useGetAllSitesQuery()
	const sites = data?.data ?? []

	const [viewSite, setViewSite] = useState<ISite | null>(null)
	const [editSite, setEditSite] = useState<ISiteDTO | null>(null)
	const [dialogOpen, setDialogOpen] = useState(false)

	const openCreate = () => {
		setEditSite(null)
		setDialogOpen(true)
	}

	const openView = (site: ISite) => {
		setViewSite(site)
	}

	const openEdit = (site: ISite) => {
		setEditSite({ id: site.id, name: site.name, address: site.address })
		setDialogOpen(true)
	}

	const closeView = () => {
		setViewSite(null)
	}

	const closeDialog = () => {
		setEditSite(null)
		setDialogOpen(false)
	}

	const handleEditFromView = (dto: ISiteDTO) => {
		setViewSite(null)
		setEditSite(dto)
		setDialogOpen(true)
	}

	return (
		<Box sx={{ flexGrow: 1, overflow: 'auto', p: 3 }}>
			<Box
				sx={{
					display: 'flex',
					flexDirection: { xs: 'column', sm: 'row' },
					justifyContent: 'space-between',
					alignItems: { xs: 'flex-start', sm: 'center' },
					gap: { xs: 2, sm: 0 },
					mb: 4,
				}}
			>
				<Box>
					<Typography variant='h5' sx={{ fontWeight: 'bold', color: '#1f2937' }}>
						Площадки
					</Typography>
					<Typography variant='body2' sx={{ color: '#6b7280', display: { xs: 'none', sm: 'block' } }}>
						Управление площадками и локациями
					</Typography>
				</Box>
				<Button
					variant='outlined'
					sx={{ borderRadius: '8px', textTransform: 'none', fontWeight: 500, gap: 1 }}
					onClick={openCreate}
				>
					<PlusIcon fill={palette.primary.main} fontSize={16} mr={1.5} />
					Добавить площадку
				</Button>
			</Box>

			{sites.length === 0 ? (
				<Box sx={{ textAlign: 'center', py: 6, color: '#6b7280' }}>
					<Typography>Площадки не найдены</Typography>
				</Box>
			) : (
				<>
					<Box sx={{ display: { xs: 'none', md: 'block' } }}>
						<SitesTable sites={sites} onView={openView} onEdit={openEdit} />
					</Box>
					<Box sx={{ display: { xs: 'block', md: 'none' } }}>
						<SiteCardList sites={sites} onView={openView} onEdit={openEdit} />
					</Box>
				</>
			)}

			<SiteViewDialog site={viewSite} onClose={closeView} onEdit={handleEditFromView} />
			<SiteDialog site={editSite} open={dialogOpen} onClose={closeDialog} />
		</Box>
	)
}
