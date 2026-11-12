package opslog

import (
	"context"
	"time"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

// recordedOp — запись, которую журнал передал в хранилище. Снимок контекста
// (errAtCall, deadline) берётся в момент вызова: после возврата из Log
// рабочий контекст уже отменён отложенным cancel, и проверить его позже
// нельзя.
type recordedOp struct {
	errAtCall   error
	hasDeadline bool
	remaining   time.Duration
	userID      *int64
	op          string
	ref         string
	ok          bool
}

// fakeStore реализует ports.Store целиком. Для журнала важны LogOp и ListOps,
// остальные методы возвращают ports.ErrNotFound, чтобы случайный вызов был
// виден в тесте, а не проходил молча.
type fakeStore struct {
	written  []recordedOp
	entries  []ports.OpLogEntry
	logErr   error
	listErr  error
	gotLimit int
	gotStart int
}

func (f *fakeStore) LogOp(ctx context.Context, userID *int64, op, ref string, ok bool) error {
	deadline, hasDeadline := ctx.Deadline()

	f.written = append(f.written, recordedOp{
		errAtCall:   ctx.Err(),
		hasDeadline: hasDeadline,
		remaining:   time.Until(deadline),
		userID:      userID,
		op:          op,
		ref:         ref,
		ok:          ok,
	})

	return f.logErr
}

func (f *fakeStore) ListOps(_ context.Context, limit, offset int) ([]ports.OpLogEntry, error) {
	f.gotLimit = limit
	f.gotStart = offset
	if f.listErr != nil {
		return nil, f.listErr
	}

	return f.entries, nil
}

func (f *fakeStore) Methods(context.Context) ([]ports.MethodRow, error) {
	return nil, ports.ErrNotFound
}

func (f *fakeStore) SetMethodEnabled(context.Context, string, bool) error {
	return ports.ErrNotFound
}

func (f *fakeStore) UserKey(context.Context, int64) (*ports.UserKey, error) {
	return nil, ports.ErrNotFound
}

func (f *fakeStore) SaveUserKey(context.Context, *ports.UserKey) error {
	return ports.ErrNotFound
}

func (f *fakeStore) CreateMessage(context.Context, *ports.Message) (int64, error) {
	return 0, ports.ErrNotFound
}

func (f *fakeStore) GetMessage(context.Context, int64) (*ports.Message, error) {
	return nil, ports.ErrNotFound
}

func (f *fakeStore) ListMessages(context.Context, *int64, int, int) ([]*ports.Message, error) {
	return nil, ports.ErrNotFound
}

func (f *fakeStore) SetDecryptMs(context.Context, int64, int64) error {
	return ports.ErrNotFound
}

// int64Ptr возвращает указатель на значение — так в тестах различаются
// «анонимный субъект» (nil) и «нулевой идентификатор».
func int64Ptr(v int64) *int64 { return &v }

var _ ports.Store = (*fakeStore)(nil)
