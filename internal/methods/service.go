// Package methods отдаёт список методов шифрования и позволяет администратору
// включать и выключать их.
//
// Пакет зависит только от ports.Store: реализация хранилища подключается при
// сборке маршрутов (contracts.md, раздел 1, правило 2).
package methods

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

// knownMethods — методы, которые есть в справочнике. Проверка кода до обращения
// к базе даёт 404 без запроса и не позволяет перебирать значения.
var knownMethods = []ports.Method{ports.MethodRSA, ports.MethodKuznyechik}

// Service — операции над справочником методов.
type Service struct {
	store ports.Store
}

// NewService строит сервис над хранилищем.
func NewService(store ports.Store) *Service {
	return &Service{store: store}
}

// List возвращает все методы, отсортированные по коду.
//
// Сортировка здесь, а не только в запросе: ответ не зависит от порядка,
// который вернула реализация хранилища.
func (s *Service) List(ctx context.Context) ([]MethodDTO, error) {
	rows, err := s.store.Methods(ctx)
	if err != nil {
		return nil, fmt.Errorf("methods: list: %w", err)
	}

	dtos := make([]MethodDTO, 0, len(rows))
	for _, row := range rows {
		dtos = append(dtos, toDTO(row))
	}

	slices.SortFunc(dtos, func(a, b MethodDTO) int {
		return strings.Compare(string(a.Code), string(b.Code))
	})

	return dtos, nil
}

// SetEnabled меняет флаг метода и возвращает обновлённую запись.
//
// Неизвестный код — ports.ErrNotFound: проверятся справочник, а не только
// запрос к базе, поэтому ответ не зависит от того, успела ли миграция
// создать строку.
func (s *Service) SetEnabled(ctx context.Context, code string, enabled bool) (MethodDTO, error) {
	if !IsKnownCode(code) {
		return MethodDTO{}, fmt.Errorf("%w: unknown method %q", ports.ErrNotFound, code)
	}

	if err := s.store.SetMethodEnabled(ctx, code, enabled); err != nil {
		return MethodDTO{}, fmt.Errorf("methods: set %q: %w", code, err)
	}

	dtos, err := s.List(ctx)
	if err != nil {
		return MethodDTO{}, err
	}

	for _, dto := range dtos {
		if string(dto.Code) == code {
			return dto, nil
		}
	}

	return MethodDTO{}, fmt.Errorf("%w: method %q disappeared", ports.ErrNotFound, code)
}

// IsKnownCode сообщает, относится ли код к методам шифрования сервиса.
func IsKnownCode(code string) bool {
	return slices.Contains(knownMethods, ports.Method(code))
}

// MethodEnabled сообщает, включён ли метод для шифрования.
//
// Неизвестный код — (false, nil), а не ошибка: вызывающий отдаст 404 по своему
// списку методов. Ошибка базы — (false, err).
//
// Флаг относится только к шифрованию: расшифрование выключенным методом
// остаётся возможным, иначе сохранённые сообщения стали бы нечитаемыми.
func MethodEnabled(ctx context.Context, store ports.Store, code string) (bool, error) {
	rows, err := store.Methods(ctx)
	if err != nil {
		return false, fmt.Errorf("methods: check %q: %w", code, err)
	}

	for _, row := range rows {
		if row.Code == code {
			return row.Enabled, nil
		}
	}

	return false, nil
}
