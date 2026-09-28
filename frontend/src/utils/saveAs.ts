export const saveAs = (blob: Blob, name: string) => {
	const href = URL.createObjectURL(blob);
	const link = window.document.createElement('a');

	link.setAttribute('href', href);
	link.setAttribute('download', name);

	link.click();
	link.remove();

	// Ревокация на следующем тике: мгновенный revoke может оборвать загрузку,
	// пока браузер ещё стартует скачивание по URL.
	setTimeout(() => URL.revokeObjectURL(href), 0);
};
