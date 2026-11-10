package methods

import "github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"

// MaxPatchBodyBytes — ограничение тела PATCH /api/methods/{code}.
const MaxPatchBodyBytes int64 = 4 << 10

// MethodDTO — метод шифрования в ответе API (contracts.md, раздел 5).
type MethodDTO struct {
	Code    ports.Method `json:"code" example:"kuznyechik"`
	Title   string       `json:"title" example:"Кузнечик"`
	KeySize int          `json:"key_size" example:"256"`
	Enabled bool         `json:"enabled" example:"true"`
}

// MethodPatchRequest — тело PATCH /api/methods/{code}.
//
// Enabled — указатель, а не bool: отсутствие поля означает «не задано» и
// даёт 400. При bool пропущенное поле стало бы false и молча выключило метод.
type MethodPatchRequest struct {
	Enabled *bool `json:"enabled"`
}

// toDTO переводит строку справочника в ответ API.
func toDTO(row ports.MethodRow) MethodDTO {
	return MethodDTO{
		Code:    ports.Method(row.Code),
		Title:   row.Title,
		KeySize: row.KeySize,
		Enabled: row.Enabled,
	}
}
