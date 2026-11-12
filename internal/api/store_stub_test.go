package api

import (
	"context"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

// recordedOp — детальная запись операции журнала, передаваемой в хранилище.
type recordedOp struct {
	userID *int64
	op     string
	ref    string
	ok     bool
}

// memStore — заглушка хранилища в памяти для проверок маршрутов.
//
// Методы, не используемые в тестах пакета, возвращают ports.ErrNotFound,
// чтобы случайный вызов был виден, а не проходил молча.
type memStore struct {
	rows  []ports.MethodRow
	setOK bool

	// ops — строки для обратной совместимости с методами (op+" "+ref).
	ops []string
	// logged — детальная запись для проверок журнала (userID, ok).
	logged []recordedOp

	entries []ports.OpLogEntry
	listErr error
	// listLimit и listOffset запоминают страницу последнего вызова ListOps.
	listLimit  int
	listOffset int

	logErr error
}

func (m *memStore) Methods(context.Context) ([]ports.MethodRow, error) {
	return m.rows, nil
}

func (m *memStore) SetMethodEnabled(_ context.Context, code string, on bool) error {
	for i := range m.rows {
		if m.rows[i].Code == code {
			m.rows[i].Enabled = on
			m.setOK = true

			return nil
		}
	}

	return ports.ErrNotFound
}

func (m *memStore) LogOp(_ context.Context, userID *int64, op, ref string, ok bool) error {
	m.ops = append(m.ops, op+" "+ref)
	m.logged = append(m.logged, recordedOp{userID: userID, op: op, ref: ref, ok: ok})

	return m.logErr
}

func (m *memStore) ListOps(_ context.Context, limit, offset int) ([]ports.OpLogEntry, error) {
	m.listLimit = limit
	m.listOffset = offset
	if m.listErr != nil {
		return nil, m.listErr
	}

	return m.entries, nil
}

func (m *memStore) UserKey(context.Context, int64) (*ports.UserKey, error) {
	return nil, ports.ErrNotFound
}

func (m *memStore) SaveUserKey(context.Context, *ports.UserKey) error {
	return ports.ErrNotFound
}

func (m *memStore) CreateMessage(context.Context, *ports.Message) (int64, error) {
	return 0, ports.ErrNotFound
}

func (m *memStore) GetMessage(context.Context, int64) (*ports.Message, error) {
	return nil, ports.ErrNotFound
}

func (m *memStore) ListMessages(context.Context, *int64, int, int) ([]*ports.Message, error) {
	return nil, ports.ErrNotFound
}

func (m *memStore) SetDecryptMs(context.Context, int64, int64) error {
	return ports.ErrNotFound
}

func newMemStore() *memStore {
	return &memStore{rows: []ports.MethodRow{
		{Code: "rsa", Title: "RSA", KeySize: 2048, Enabled: true},
		{Code: "kuznyechik", Title: "Кузнечик", KeySize: 256, Enabled: true},
	}}
}

func int64Ptr(v int64) *int64 {
	return &v
}

var _ ports.Store = (*memStore)(nil)
