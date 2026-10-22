package store

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

const testRSABits = 2048

// newTestStore подключается к TEST_DSN, применяет миграции и очищает данные.
// Без TEST_DSN тест пропускается, чтобы go test ./... работал без Postgres.
func newTestStore(t *testing.T) *Store {
	t.Helper()

	dsn := os.Getenv("TEST_DSN")
	if dsn == "" {
		t.Skip("TEST_DSN is not set")
	}

	ctx := t.Context()

	st, err := NewStore(ctx, dsn, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	t.Cleanup(st.Close)

	if err := st.Migrate(ctx, testRSABits); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	_, err = st.pool.Exec(ctx, `TRUNCATE messages, user_keys, ops_log RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
	if _, err := st.pool.Exec(ctx, `UPDATE methods SET enabled = TRUE`); err != nil {
		t.Fatalf("reset methods: %v", err)
	}

	return st
}

func int64Ptr(v int64) *int64 {
	return &v
}

func TestNewStoreInvalidDSN(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
	}{
		{name: "мусор вместо dsn", dsn: "::not a dsn::"},
		{name: "неизвестный параметр", dsn: "postgres://u:p@localhost:1/db?bogus_param=%zz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewStore(t.Context(), tt.dsn, slog.New(slog.NewTextHandler(io.Discard, nil)))
			if err == nil {
				t.Fatal("NewStore() error = nil, want error")
			}
		})
	}
}

func TestMigrate(t *testing.T) {
	st := newTestStore(t)
	ctx := t.Context()

	// чистая БД + повторный запуск
	if err := st.Migrate(ctx, testRSABits); err != nil {
		t.Fatalf("second Migrate() error = %v", err)
	}

	var applied int
	if err := st.pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&applied); err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}
	if applied != 1 {
		t.Errorf("schema_migrations rows = %d, want 1", applied)
	}

	methods, err := st.Methods(ctx)
	if err != nil {
		t.Fatalf("Methods() error = %v", err)
	}
	if len(methods) != 2 {
		t.Fatalf("len(Methods()) = %d, want 2", len(methods))
	}
}

func TestMigrateKeepsDisabledMethod(t *testing.T) {
	st := newTestStore(t)
	ctx := t.Context()

	if err := st.SetMethodEnabled(ctx, string(ports.MethodRSA), false); err != nil {
		t.Fatalf("SetMethodEnabled() error = %v", err)
	}

	other, err := NewStore(ctx, os.Getenv("TEST_DSN"), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer other.Close()

	if err := other.Migrate(ctx, testRSABits); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	methods, err := st.Methods(ctx)
	if err != nil {
		t.Fatalf("Methods() error = %v", err)
	}
	for _, m := range methods {
		if m.Code == string(ports.MethodRSA) && m.Enabled {
			t.Error("rsa enabled after restart, want disabled")
		}
	}
}

func TestMethods(t *testing.T) {
	st := newTestStore(t)

	got, err := st.Methods(t.Context())
	if err != nil {
		t.Fatalf("Methods() error = %v", err)
	}

	want := []ports.MethodRow{
		{Code: "kuznyechik", Title: "Кузнечик", KeySize: 256, Enabled: true},
		{Code: "rsa", Title: "RSA", KeySize: testRSABits, Enabled: true},
	}
	if len(got) != len(want) {
		t.Fatalf("Methods() = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Methods()[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestSetMethodEnabled(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		on      bool
		wantErr error
	}{
		{name: "выключить rsa", code: "rsa", on: false},
		{name: "включить кузнечик", code: "kuznyechik", on: true},
		{name: "неизвестный метод", code: "aes", on: false, wantErr: ports.ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := newTestStore(t)
			ctx := t.Context()

			err := st.SetMethodEnabled(ctx, tt.code, tt.on)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("SetMethodEnabled() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}

			var enabled bool
			if err := st.pool.QueryRow(ctx, `SELECT enabled FROM methods WHERE code = $1`, tt.code).Scan(&enabled); err != nil {
				t.Fatalf("read enabled: %v", err)
			}
			if enabled != tt.on {
				t.Errorf("enabled = %v, want %v", enabled, tt.on)
			}
		})
	}
}

func TestUserKey(t *testing.T) {
	tests := []struct {
		name string
		save []*ports.UserKey
		want *ports.UserKey
	}{
		{
			name: "generating",
			save: []*ports.UserKey{{UserID: 7, Bits: 2048, Status: ports.KeyStatusGenerating}},
			want: &ports.UserKey{UserID: 7, Bits: 2048, Status: ports.KeyStatusGenerating},
		},
		{
			name: "generating then ready",
			save: []*ports.UserKey{
				{UserID: 7, Bits: 2048, Status: ports.KeyStatusGenerating},
				{
					UserID: 7, Bits: 2048, Status: ports.KeyStatusReady, KeygenMs: 1500,
					PublicKeyPEM:  "-----BEGIN RSA PUBLIC KEY-----\ntest\n-----END RSA PUBLIC KEY-----\n",
					PrivateSealed: []byte{0x01, 0xAA, 0xBB},
				},
			},
			want: &ports.UserKey{
				UserID: 7, Bits: 2048, Status: ports.KeyStatusReady, KeygenMs: 1500,
				PublicKeyPEM:  "-----BEGIN RSA PUBLIC KEY-----\ntest\n-----END RSA PUBLIC KEY-----\n",
				PrivateSealed: []byte{0x01, 0xAA, 0xBB},
			},
		},
		{
			name: "system key",
			save: []*ports.UserKey{{UserID: 0, Bits: 4096, Status: ports.KeyStatusFailed}},
			want: &ports.UserKey{UserID: 0, Bits: 4096, Status: ports.KeyStatusFailed},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := newTestStore(t)
			ctx := t.Context()

			for _, k := range tt.save {
				if err := st.SaveUserKey(ctx, k); err != nil {
					t.Fatalf("SaveUserKey() error = %v", err)
				}
			}

			got, err := st.UserKey(ctx, tt.want.UserID)
			if err != nil {
				t.Fatalf("UserKey() error = %v", err)
			}
			if got.UpdatedAt.IsZero() {
				t.Error("UpdatedAt is zero")
			}

			got.UpdatedAt = time.Time{}
			if got.UserID != tt.want.UserID || got.Bits != tt.want.Bits || got.Status != tt.want.Status ||
				got.KeygenMs != tt.want.KeygenMs || got.PublicKeyPEM != tt.want.PublicKeyPEM ||
				!bytes.Equal(got.PrivateSealed, tt.want.PrivateSealed) {
				t.Errorf("UserKey() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestUserKeyNotFound(t *testing.T) {
	st := newTestStore(t)

	if _, err := st.UserKey(t.Context(), 42); !errors.Is(err, ports.ErrNotFound) {
		t.Fatalf("UserKey() error = %v, want ErrNotFound", err)
	}
}

func TestSaveUserKeyUpdatesTimestamp(t *testing.T) {
	st := newTestStore(t)
	ctx := t.Context()

	k := &ports.UserKey{UserID: 1, Bits: 2048, Status: ports.KeyStatusGenerating}
	if err := st.SaveUserKey(ctx, k); err != nil {
		t.Fatalf("SaveUserKey() error = %v", err)
	}
	first, err := st.UserKey(ctx, 1)
	if err != nil {
		t.Fatalf("UserKey() error = %v", err)
	}

	// now() внутри транзакции фиксирован, поэтому сдвигаем время вручную.
	if _, err := st.pool.Exec(ctx, `UPDATE user_keys SET updated_at = updated_at - interval '1 hour'`); err != nil {
		t.Fatalf("shift updated_at: %v", err)
	}

	k.Status = ports.KeyStatusReady
	if err := st.SaveUserKey(ctx, k); err != nil {
		t.Fatalf("SaveUserKey() error = %v", err)
	}
	second, err := st.UserKey(ctx, 1)
	if err != nil {
		t.Fatalf("UserKey() error = %v", err)
	}

	if second.UpdatedAt.Before(first.UpdatedAt) {
		t.Errorf("UpdatedAt = %v, want not before %v", second.UpdatedAt, first.UpdatedAt)
	}
}

func testMessage(ownerID int64) *ports.Message {
	return &ports.Message{
		OwnerID:      ownerID,
		OwnerLogin:   "user",
		Method:       string(ports.MethodKuznyechik),
		KeySize:      256,
		Plaintext:    "Привет, мир",
		Ciphertext:   []byte{0x00, 0x01, 0xFE, 0xFF},
		KuzKeySealed: []byte{0x01, 0x02, 0x03},
		KuzIV:        []byte{0x10, 0x20},
		KeygenMs:     0,
		EncryptMs:    12,
	}
}

func TestCreateAndGetMessage(t *testing.T) {
	st := newTestStore(t)
	ctx := t.Context()

	in := testMessage(5)

	id, err := st.CreateMessage(ctx, in)
	if err != nil {
		t.Fatalf("CreateMessage() error = %v", err)
	}
	if id != 1 {
		t.Errorf("CreateMessage() id = %d, want 1", id)
	}

	got, err := st.GetMessage(ctx, id)
	if err != nil {
		t.Fatalf("GetMessage() error = %v", err)
	}

	if got.ID != id || got.OwnerID != in.OwnerID || got.OwnerLogin != in.OwnerLogin ||
		got.Method != in.Method || got.KeySize != in.KeySize || got.Plaintext != in.Plaintext ||
		!bytes.Equal(got.Ciphertext, in.Ciphertext) || !bytes.Equal(got.KuzKeySealed, in.KuzKeySealed) ||
		!bytes.Equal(got.KuzIV, in.KuzIV) || got.EncryptMs != in.EncryptMs || got.DecryptMs != 0 {
		t.Errorf("GetMessage() = %+v, want %+v", got, in)
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

func TestCreateMessageRSAWithoutKuzFields(t *testing.T) {
	st := newTestStore(t)
	ctx := t.Context()

	in := testMessage(5)
	in.Method = string(ports.MethodRSA)
	in.KeySize = 2048
	in.KuzKeySealed = nil
	in.KuzIV = nil
	in.KeygenMs = 900

	id, err := st.CreateMessage(ctx, in)
	if err != nil {
		t.Fatalf("CreateMessage() error = %v", err)
	}

	got, err := st.GetMessage(ctx, id)
	if err != nil {
		t.Fatalf("GetMessage() error = %v", err)
	}
	if got.KuzKeySealed != nil || got.KuzIV != nil {
		t.Errorf("kuz fields = %v, %v, want nil", got.KuzKeySealed, got.KuzIV)
	}
	if got.KeygenMs != 900 {
		t.Errorf("KeygenMs = %d, want 900", got.KeygenMs)
	}
}

func TestCreateMessageUnknownMethod(t *testing.T) {
	st := newTestStore(t)

	in := testMessage(5)
	in.Method = "aes"

	if _, err := st.CreateMessage(t.Context(), in); err == nil {
		t.Fatal("CreateMessage() error = nil, want foreign key violation")
	}
}

func TestGetMessageNotFound(t *testing.T) {
	st := newTestStore(t)

	if _, err := st.GetMessage(t.Context(), 100); !errors.Is(err, ports.ErrNotFound) {
		t.Fatalf("GetMessage() error = %v, want ErrNotFound", err)
	}
}

// seedMessages создаёт сообщения владельцев 1 и 2 с одинаковым created_at,
// чтобы проверить второй ключ сортировки.
func seedMessages(t *testing.T, st *Store) {
	t.Helper()

	ctx := t.Context()
	for _, owner := range []int64{1, 2, 1, 1, 2} {
		if _, err := st.CreateMessage(ctx, testMessage(owner)); err != nil {
			t.Fatalf("CreateMessage() error = %v", err)
		}
	}

	if _, err := st.pool.Exec(ctx, `UPDATE messages SET created_at = '2026-01-01T00:00:00Z'`); err != nil {
		t.Fatalf("align created_at: %v", err)
	}
}

func TestListMessages(t *testing.T) {
	tests := []struct {
		name    string
		ownerID *int64
		limit   int
		offset  int
		wantIDs []int64
	}{
		{name: "все сообщения", ownerID: nil, limit: 10, wantIDs: []int64{5, 4, 3, 2, 1}},
		{name: "только свои", ownerID: int64Ptr(1), limit: 10, wantIDs: []int64{4, 3, 1}},
		{name: "первая страница", ownerID: nil, limit: 2, wantIDs: []int64{5, 4}},
		{name: "вторая страница", ownerID: nil, limit: 2, offset: 2, wantIDs: []int64{3, 2}},
		{name: "за пределами", ownerID: nil, limit: 2, offset: 10, wantIDs: nil},
		{name: "чужой владелец", ownerID: int64Ptr(99), limit: 10, wantIDs: nil},
		{name: "системный 0 не значит все", ownerID: int64Ptr(0), limit: 10, wantIDs: nil},
	}

	st := newTestStore(t)
	seedMessages(t, st)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := st.ListMessages(t.Context(), tt.ownerID, tt.limit, tt.offset)
			if err != nil {
				t.Fatalf("ListMessages() error = %v", err)
			}

			if len(got) != len(tt.wantIDs) {
				t.Fatalf("len(ListMessages()) = %d, want %d", len(got), len(tt.wantIDs))
			}
			for i, m := range got {
				if m.ID != tt.wantIDs[i] {
					t.Errorf("ListMessages()[%d].ID = %d, want %d", i, m.ID, tt.wantIDs[i])
				}
			}
		})
	}
}

func TestSetDecryptMs(t *testing.T) {
	st := newTestStore(t)
	ctx := t.Context()

	id, err := st.CreateMessage(ctx, testMessage(1))
	if err != nil {
		t.Fatalf("CreateMessage() error = %v", err)
	}

	tests := []struct {
		name    string
		id      int64
		ms      int64
		wantErr error
	}{
		{name: "существующее сообщение", id: id, ms: 42},
		{name: "нет сообщения", id: id + 100, ms: 42, wantErr: ports.ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := st.SetDecryptMs(ctx, tt.id, tt.ms)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("SetDecryptMs() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}

			got, err := st.GetMessage(ctx, tt.id)
			if err != nil {
				t.Fatalf("GetMessage() error = %v", err)
			}
			if got.DecryptMs != tt.ms {
				t.Errorf("DecryptMs = %d, want %d", got.DecryptMs, tt.ms)
			}
		})
	}
}

func TestLogOp(t *testing.T) {
	tests := []struct {
		name   string
		userID *int64
		op     string
		ref    string
		ok     bool
	}{
		{name: "anonymous", userID: nil, op: "signature.verify", ref: "sha256:0123456789abcdef", ok: true},
		{name: "user", userID: int64Ptr(3), op: "message.encrypt", ref: "1", ok: true},
		{name: "system", userID: int64Ptr(0), op: "message.pdf", ref: "1", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := newTestStore(t)
			ctx := context.Background()

			if err := st.LogOp(ctx, tt.userID, tt.op, tt.ref, tt.ok); err != nil {
				t.Fatalf("LogOp() error = %v", err)
			}

			var (
				userID *int64
				op     string
				ref    string
				ok     bool
			)
			err := st.pool.QueryRow(ctx, `SELECT user_id, op, ref, ok FROM ops_log`).Scan(&userID, &op, &ref, &ok)
			if err != nil {
				t.Fatalf("read ops_log: %v", err)
			}

			if (userID == nil) != (tt.userID == nil) || (userID != nil && *userID != *tt.userID) {
				t.Errorf("user_id = %v, want %v", userID, tt.userID)
			}
			if op != tt.op || ref != tt.ref || ok != tt.ok {
				t.Errorf("row = (%q, %q, %v), want (%q, %q, %v)", op, ref, ok, tt.op, tt.ref, tt.ok)
			}
		})
	}
}
