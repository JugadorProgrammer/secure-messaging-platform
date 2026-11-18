package auth

import (
	"net/http"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
)

// RoleDTO — роль в формате интерфейса. Идентификаторы совпадают с
// идентификаторами ролей Сервиса авторизации: по ним интерфейс создаёт
// пользователей.
type RoleDTO struct {
	ID          int    `json:"id"`
	Name        Role   `json:"name"`
	Description string `json:"description"`
}

// Идентификаторы ролей в Сервисе авторизации.
const (
	RoleUserID  = 1
	RoleAdminID = 2
)

// Roles возвращает справочник ролей. Справочник постоянный, поэтому
// отдаётся копия: вызывающий не изменит общий срез.
func Roles() []RoleDTO {
	return []RoleDTO{
		{ID: RoleUserID, Name: RoleUser, Description: "Обычный пользователь"},
		{ID: RoleAdminID, Name: RoleAdmin, Description: "Администратор"},
	}
}

// ListRoles отвечает на GET /api/roles. Маршрут публичный: список ролей не
// секретен, а форма регистрации нужна до входа.
//
// @Summary  Справочник ролей
// @Tags     auth
// @Produce  json
// @Success  200 {array} RoleDTO
// @Router   /api/roles [get]
func ListRoles(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, Roles())
}
