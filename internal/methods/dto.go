package methods

import "github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"

// MaxSetBodyBytes — ограничение тела POST /api/setEncryptionMethodAvailability.
const MaxSetBodyBytes int64 = 4 << 10

// Названия методов во внешнем API. Фронт показывает их пользователю и
// присылает обратно вместо кодов, поэтому значения фиксированы и совпадают
// с MESSAGE_ENCRYPTION_METHODS фронта. RSA назван по размеру ключа стенда.
const (
	NameRSA        = "RSA-32768"
	NameKuznyechik = "Кузнечик"
)

// methodNames связывает коды справочника с названиями во внешнем API.
var methodNames = map[ports.Method]string{
	ports.MethodRSA:        NameRSA,
	ports.MethodKuznyechik: NameKuznyechik,
}

// Name возвращает название метода для внешнего API. Неизвестный код
// возвращается как есть: справочник мог пополниться раньше этой таблицы.
func Name(code ports.Method) string {
	if name, ok := methodNames[code]; ok {
		return name
	}

	return string(code)
}

// CodeByName находит код метода по названию из внешнего API. Код вместо
// названия тоже принимается: так запрос можно отправить и без фронта.
func CodeByName(name string) (ports.Method, bool) {
	for code, known := range methodNames {
		if name == known || name == string(code) {
			return code, true
		}
	}

	return "", false
}

// MethodDTO — строка справочника методов внутри сервиса.
type MethodDTO struct {
	Code    ports.Method `json:"code" example:"kuznyechik"`
	Title   string       `json:"title" example:"Кузнечик"`
	KeySize int          `json:"key_size" example:"256"`
	Enabled bool         `json:"enabled" example:"true"`
}

// EncryptionMethodDTO — метод шифрования в ответе API.
type EncryptionMethodDTO struct {
	Name    string `json:"name" example:"Кузнечик"`
	Enabled bool   `json:"enabled" example:"true"`
}

// SetAvailabilityRequest — тело POST /api/setEncryptionMethodAvailability.
//
// Enabled — указатель, а не bool: отсутствие поля означает «не задано» и
// даёт 400. При bool пропущенное поле стало бы false и молча выключило метод.
type SetAvailabilityRequest struct {
	Name    string `json:"name"`
	Enabled *bool  `json:"enabled"`
}

// toDTO переводит строку справочника во внутреннюю модель.
func toDTO(row ports.MethodRow) MethodDTO {
	return MethodDTO{
		Code:    ports.Method(row.Code),
		Title:   row.Title,
		KeySize: row.KeySize,
		Enabled: row.Enabled,
	}
}

// toAPI переводит метод в ответ API.
func toAPI(dto MethodDTO) EncryptionMethodDTO {
	return EncryptionMethodDTO{Name: Name(dto.Code), Enabled: dto.Enabled}
}
