package opslog

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

// secretSubstrings — то, чего не должно быть в полях записи журнала.
// Проверка по подстрокам, как требует DoD задачи: тексты сообщений, ключи
// и токены в журнал не пишутся.
var secretSubstrings = []string{
	"bearer", "token", "password", "master", "secret", "begin rsa", "private key",
}

func newLogger(st ports.Store, w io.Writer) *Logger {
	log := slog.New(slog.NewTextHandler(w, nil))

	return New(st, log)
}

func TestLoggerWritesEveryOperation(t *testing.T) {
	tests := []struct {
		name   string
		userID *int64
		op     string
		ref    string
		ok     bool
	}{
		{"шифрование", int64Ptr(7), OpMessageEncrypt, "12", true},
		{"расшифрование", int64Ptr(7), OpMessageDecrypt, "12", true},
		{"выгрузка pdf", int64Ptr(7), OpMessagePDF, "12", false},
		{"проверка подписи", nil, OpSignatureVerify, "sha256:0123456789abcdef", true},
		{"переключение метода", int64Ptr(1), OpMethodToggle, "rsa", true},
		{"отказ в доступе", int64Ptr(7), OpAccessDenied, "GET /api/users", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := &fakeStore{}
			logger := newLogger(st, io.Discard)

			logger.Log(t.Context(), tt.userID, tt.op, tt.ref, tt.ok)

			if len(st.written) != 1 {
				t.Fatalf("written = %d records, want 1", len(st.written))
			}

			got := st.written[0]
			if got.op != tt.op || got.ref != tt.ref || got.ok != tt.ok {
				t.Errorf("record = op %q ref %q ok %v, want op %q ref %q ok %v",
					got.op, got.ref, got.ok, tt.op, tt.ref, tt.ok)
			}

			assertSameUser(t, got.userID, tt.userID)
		})
	}
}

func TestLoggerKeepsAnonymousSubjectNil(t *testing.T) {
	st := &fakeStore{}

	newLogger(st, io.Discard).Log(t.Context(), nil, OpSignatureVerify, "sha256:00", true)

	if len(st.written) != 1 {
		t.Fatalf("written = %d, want 1", len(st.written))
	}
	if st.written[0].userID != nil {
		t.Errorf("userID = %d, want nil: аноним — это NULL, а не 0", *st.written[0].userID)
	}
}

func TestLoggerWritesAfterRequestContextIsCanceled(t *testing.T) {
	st := &fakeStore{}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	newLogger(st, io.Discard).Log(ctx, int64Ptr(7), OpAccessDenied, "GET /api/users", false)

	if len(st.written) != 1 {
		t.Fatalf("written = %d, want 1: отмена запроса не должна терять событие", len(st.written))
	}

	got := st.written[0]
	if got.errAtCall != nil {
		t.Errorf("context of the write is %v, want alive", got.errAtCall)
	}
	if !got.hasDeadline {
		t.Fatal("write context has no deadline, want WriteTimeout")
	}
	if got.remaining <= 0 || got.remaining > WriteTimeout {
		t.Errorf("remaining budget = %v, want (0, %v]", got.remaining, WriteTimeout)
	}
}

func TestLoggerTruncatesLongRef(t *testing.T) {
	st := &fakeStore{}
	long := strings.Repeat("/api/", 500)

	newLogger(st, io.Discard).Log(t.Context(), int64Ptr(7), OpAccessDenied, long, false)

	got := st.written[0].ref
	if len(got) > MaxRefLen {
		t.Fatalf("ref length = %d, want <= %d", len(got), MaxRefLen)
	}
	if !strings.HasPrefix(got, "/api/") {
		t.Errorf("ref = %q, want the beginning of the original value", got)
	}
}

func TestLoggerRefTruncationKeepsCharactersWhole(t *testing.T) {
	st := &fakeStore{}
	// Каждый символ занимает три байта: отсечение по границе MaxRefLen попадает
	// в середину символа, и база отвергла бы вставку из-за битого UTF-8.
	ref := strings.Repeat("€", 100)

	newLogger(st, io.Discard).Log(t.Context(), int64Ptr(7), OpAccessDenied, ref, false)

	got := st.written[0].ref
	if !utf8.ValidString(got) {
		t.Errorf("ref = %q, want valid utf-8", got)
	}
	if len(got) > MaxRefLen {
		t.Errorf("ref length = %d bytes, want <= %d", len(got), MaxRefLen)
	}
	if want := strings.Repeat("€", MaxRefLen/3); got != want {
		t.Errorf("ref = %q..., want the whole characters up to %d bytes", want, MaxRefLen)
	}
}

func TestLoggerDropsInvalidUTF8FromRef(t *testing.T) {
	st := &fakeStore{}

	newLogger(st, io.Discard).Log(t.Context(), nil, OpAccessDenied, "/api/\xff\xfe/users", false)

	got := st.written[0].ref
	if !utf8.ValidString(got) {
		t.Errorf("ref = %q, want invalid bytes removed", got)
	}
	if !strings.HasPrefix(got, "/api/") || !strings.HasSuffix(got, "/users") {
		t.Errorf("ref = %q, want the readable part kept", got)
	}
}

func TestLoggerStoreErrorIsLoggedAndSwallowed(t *testing.T) {
	st := &fakeStore{logErr: errors.New("database is down")}
	var buf bytes.Buffer

	// Вызов не должен паниковать: ошибка журнала не влияет на ответ клиента.
	newLogger(st, &buf).Log(t.Context(), int64Ptr(7), OpMethodToggle, "rsa", true)

	if len(st.written) != 1 {
		t.Fatalf("written = %d, want 1", len(st.written))
	}
	if !strings.Contains(buf.String(), msgWriteFailed) {
		t.Errorf("log = %q, want a record about the failed write", buf.String())
	}
	if !strings.Contains(buf.String(), "database is down") {
		t.Errorf("log = %q, want the store error text", buf.String())
	}
	if strings.Contains(buf.String(), "rsa") {
		t.Errorf("log = %q, want ref out of the log line", buf.String())
	}
}

func TestLoggerWithoutStoreIsNoop(t *testing.T) {
	var nilLogger *Logger

	// Nil-экземпляр — штатное состояние «хранилища нет», паники быть не должно.
	nilLogger.Log(t.Context(), int64Ptr(7), OpAccessDenied, "GET /api/users", false)

	if New(nil, nil) != nil {
		t.Error("New(nil, nil) returned a logger, want nil")
	}
}

func TestRecordedFieldsHaveNoSecrets(t *testing.T) {
	st := &fakeStore{}
	logger := newLogger(st, io.Discard)

	refs := []string{"12", "rsa", "GET /api/users", "sha256:0123456789abcdef"}
	for i, ref := range refs {
		logger.Log(t.Context(), int64Ptr(int64(i+1)), OpMessageEncrypt, ref, true)
	}

	for _, rec := range st.written {
		joined := strings.ToLower(rec.op + " " + rec.ref)
		for _, forbidden := range secretSubstrings {
			if strings.Contains(joined, forbidden) {
				t.Errorf("record %q contains %q", joined, forbidden)
			}
		}
	}
}

func assertSameUser(t *testing.T, got, want *int64) {
	t.Helper()

	switch {
	case got == nil || want == nil:
		if got != want {
			t.Errorf("userID = %v, want %v", got, want)
		}
	case *got != *want:
		t.Errorf("userID = %d, want %d", *got, *want)
	}
}
