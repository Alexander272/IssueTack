package constants

type Actions struct {
	Code string
	Name string
	Desc string
}

type PermActions struct {
	Read      Actions
	Write     Actions
	Delete    Actions
	ReadWrite Actions
	All       Actions
}

var ActionsList = PermActions{
	Read:      Actions{"read", "Чтение", "Просмотр данных"},
	Write:     Actions{"write", "Запись", "Создание и обновление"},
	Delete:    Actions{"delete", "Удаление", "Удаление записей"},
	ReadWrite: Actions{"read|write", "Чтение и запись", "Полный доступ без удаления"},
	All:       Actions{"*", "Все действия", "Полный доступ включая удаление"},
}
