package services

import (
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/pkg/error_bot"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
)

// bestEffortError логирует и сигнализирует разработчику (error_bot) об ошибке,
// возникшей после того, как основная операция уже закоммичена. Такая ошибка не
// должна превращать успешную операцию в 500 для клиента, но о ней обязательно
// нужно сообщить, чтобы не потерять факт сбоя.
func bestEffortError(action string, err error, attrs map[string]string) {
	if err == nil {
		return
	}
	errMsg := fmt.Sprintf("%s: %v", action, err)
	slogArgs := []any{}
	for k, v := range attrs {
		slogArgs = append(slogArgs, logger.StringAttr(k, v))
	}
	slogArgs = append(slogArgs, logger.ErrAttr(err))
	logger.Error(errMsg, slogArgs...)
	error_bot.Send(nil, errMsg, attrs)
}
