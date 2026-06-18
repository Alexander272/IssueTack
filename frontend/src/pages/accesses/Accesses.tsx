import { Suspense } from 'react'
import { Outlet } from 'react-router'

import { PageBox } from '@/components/PageBox/PageBox'
import { Fallback } from '@/components/Fallback/Fallback'

export default function Accesses() {
	return (
		<PageBox>
			<Suspense fallback={<Fallback />}>
				<Outlet />
			</Suspense>
		</PageBox>
	)
}
