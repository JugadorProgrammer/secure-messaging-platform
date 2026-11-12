// Package opslog ведёт журнал операций ops_log и отдаёт его администратору.
//
// Журнал пишется best-effort: сбой записи не меняет ответ клиента и не
// останавливает операцию. В записях не бывает текстов сообщений, ключей и
// токенов — только код операции и ссылка на объект.
package opslog

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

// Коды операций журнала (contracts.md, раздел 10).
const (
	OpMessageEncrypt  = "message.encrypt"
	OpMessageDecrypt  = "message.decrypt"
	OpMessagePDF      = "message.pdf"
	OpSignatureVerify = "signature.verify"
	OpMethodToggle    = "method.toggle"
	OpAccessDenied    = "access.denied"
)

const (
	// MaxRefLen — предельная длина ref в записи журнала.
	// Значение приходит из URL и тела запроса, то есть из-под контроля
	// клиента, и без ограничения попало бы в базу целиком.
	MaxRefLen = 200

	// WriteTimeout — бюджет на запись журнала. Он не зависит от контекста
	// запроса: к моменту записи исходный контекст уже может быть отменён.
	WriteTimeout = 2 * time.Second

	msgWriteFailed = "ops log: write failed"
)

// Logger пишет записи в журнал операций.
//
// Nil-экземпляр безопасен: Log не делает ничего. Поэтому маршруты можно
// собирать без хранилища, а middleware получает запись отказа, когда
// хранилище подключено.
type Logger struct {
	store ports.Store
	log   *slog.Logger
}

// New готовит журнал поверх хранилища. При nil store возвращается nil,
// что означает «журнал выключен». Логгер по умолчанию — slog.Default.
func New(store ports.Store, log *slog.Logger) *Logger {
	if store == nil {
		return nil
	}

	if log == nil {
		log = slog.Default()
	}

	return &Logger{store: store, log: log}
}

// Log записывает операцию. userID == nil — анонимный субъект, в базе NULL;
// ноль подставлен быть не может, этот идентификатор занят системным ключом.
//
// Запись отвязывается от контекста запроса (context.WithoutCancel) и получает
// собственный таймаут, поэтому обрыв соединения клиента не теряет событие, а
// зависшая база не держит запрос. Ошибка пишется в лог и не возвращается:
// вызывающий код не должен менять ответ из-за журнала.
func (l *Logger) Log(ctx context.Context, userID *int64, op, ref string, ok bool) {
	if l == nil || l.store == nil {
		return
	}

	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), WriteTimeout)
	defer cancel()

	if err := l.store.LogOp(writeCtx, userID, op, TruncateRef(ref), ok); err != nil {
		l.log.ErrorContext(writeCtx, msgWriteFailed, "op", op, "err", err)
	}
}

// TruncateRef приводит ссылку к виду, пригодному для записи в базу:
// не длиннее MaxRefLen байт и только корректные последовательности UTF-8.
//
// Лишнее отсекается по границе символа, недопустимые последовательности
// убираются целиком. Иначе Postgres отвергнет вставку с ошибкой кодировки,
// и событие журнала потеряется молча — достаточно запроса с %FF в пути.
func TruncateRef(ref string) string {
	if len(ref) > MaxRefLen {
		ref = ref[:MaxRefLen]
	}

	return strings.ToValidUTF8(ref, "")
}
