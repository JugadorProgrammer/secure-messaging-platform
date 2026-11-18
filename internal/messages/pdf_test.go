package messages

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"testing"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/auth"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/keys"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/methods"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/pdfsig"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

const pdfPath = "/api/exportPdfWithSignature"

func (e *env) exportPDF(t *testing.T, p auth.Principal, id int64) []byte {
	t.Helper()

	rec := e.post(t, pdfPath, p, ExportPDFRequest{ID: id})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}

	ref := strconv.FormatInt(id, 10)

	for header, want := range map[string]string{
		"Content-Type":        "application/pdf",
		"Content-Disposition": `attachment; filename="message-` + ref + `.pdf"`,
		"Content-Length":      strconv.Itoa(rec.Body.Len()),
		"Cache-Control":       "no-store",
	} {
		if got := rec.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}

	return rec.Body.Bytes()
}

func TestExportPDFSignature(t *testing.T) {
	for _, method := range []string{methods.NameRSA, methods.NameKuznyechik} {
		t.Run(method, func(t *testing.T) {
			e := newEnv(t)
			m := e.encrypt(t, userPrincipal(), testText, method)
			e.resetAudit()

			file := e.exportPDF(t, userPrincipal(), m.ID)

			signed, err := pdfsig.Parse(file)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}

			if !bytes.HasPrefix(signed.Raw, []byte("%PDF-")) {
				t.Errorf("raw pdf starts with %q", signed.Raw[:min(8, len(signed.Raw))])
			}

			wantFP, err := pdfsig.Fingerprint(e.store.userKeys[keys.SystemUserID].PublicKeyPEM)
			if err != nil {
				t.Fatalf("Fingerprint: %v", err)
			}

			pdfHash := sha512.Sum512(signed.Raw)
			wantMeta := pdfsig.Meta{
				Version: pdfsig.Version, Signer: testLogin, SignedAt: "2026-11-14T09:30:00Z", KeyFingerprint: wantFP,
				PDFHashHex: hex.EncodeToString(pdfHash[:]), HashAlg: pdfsig.HashAlg, SigAlg: pdfsig.SigAlg,
			}

			if signed.Meta != wantMeta {
				t.Errorf("meta = %+v, want %+v", signed.Meta, wantMeta)
			}

			metaJSON, err := wantMeta.JSON()
			if err != nil {
				t.Fatalf("meta JSON: %v", err)
			}

			digest := sha512.Sum512(append(pdfHash[:], metaJSON...))
			if want := append([]byte("sig:"), digest[:]...); !bytes.Equal(signed.Sig, want) {
				t.Errorf("sig = %x, want sign(digest)", signed.Sig)
			}

			ref := strconv.FormatInt(m.ID, 10)
			if want := []auditRecord{{testUserID, opPDF, ref, true}}; !slices.Equal(e.auditLog(), want) {
				t.Errorf("audit = %+v, want %+v", e.auditLog(), want)
			}
		})
	}
}

// Документ строится от created_at, а не от текущего времени: повторная
// выгрузка даёт тот же PDF и тот же хеш.
func TestExportPDFIsDeterministic(t *testing.T) {
	e := newEnv(t)
	m := e.encrypt(t, userPrincipal(), testText, methods.NameKuznyechik)

	first, err := pdfsig.Parse(e.exportPDF(t, userPrincipal(), m.ID))
	if err != nil {
		t.Fatalf("Parse first: %v", err)
	}

	second, err := pdfsig.Parse(e.exportPDF(t, userPrincipal(), m.ID))
	if err != nil {
		t.Fatalf("Parse second: %v", err)
	}

	if !bytes.Equal(first.Raw, second.Raw) {
		t.Error("PDF differs between exports of the same message")
	}
}

func TestExportPDFLongCiphertext(t *testing.T) {
	e := newEnv(t)
	m := e.encrypt(t, userPrincipal(), string(bytes.Repeat([]byte("Длинный текст. "), 2000)), methods.NameKuznyechik)

	if _, err := pdfsig.Parse(e.exportPDF(t, userPrincipal(), m.ID)); err != nil {
		t.Fatalf("Parse: %v", err)
	}
}

func TestExportPDFAccess(t *testing.T) {
	tests := []struct {
		name string
		p    auth.Principal
	}{
		{"чужое сообщение", otherPrincipal()},
		{"администратор", adminPrincipal()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			m := e.encrypt(t, userPrincipal(), testText, methods.NameRSA)
			e.resetAudit()

			rec := e.post(t, pdfPath, tt.p, ExportPDFRequest{ID: m.ID})
			assertError(t, rec, http.StatusNotFound, httpx.CodeNotFound)

			want := []auditRecord{{tt.p.UserID, opAccessDenied, "pdf 1", false}}
			if got := e.auditLog(); !slices.Equal(got, want) {
				t.Errorf("audit = %+v, want %+v", got, want)
			}

			if len(e.crypto.signed) != 0 {
				t.Error("foreign message was signed")
			}
		})
	}
}

func TestExportPDFErrors(t *testing.T) {
	failed := []auditRecord{{testUserID, opPDF, "", false}}

	tests := []struct {
		name       string
		setup      func(e *env)
		body       string
		wantStatus int
		wantCode   string
		wantAudit  []auditRecord
	}{
		{
			name:       "тело не json",
			body:       `{"id":`,
			wantStatus: http.StatusBadRequest,
			wantCode:   httpx.CodeValidation,
		},
		{
			name:       "id не число",
			body:       `{"id":"1"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   httpx.CodeValidation,
		},
		{
			name:       "id ноль",
			body:       `{"id":0}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   httpx.CodeValidation,
		},
		{
			name:       "сообщения нет",
			body:       `{"id":42}`,
			wantStatus: http.StatusNotFound,
			wantCode:   httpx.CodeNotFound,
		},
		{
			name:       "ошибка хранилища",
			setup:      func(e *env) { e.store.getErr = errors.New("db down") },
			body:       `{"id":1}`,
			wantStatus: http.StatusInternalServerError,
			wantCode:   httpx.CodeInternal,
			wantAudit:  failed,
		},
		{
			name:       "системного ключа нет",
			setup:      func(e *env) { delete(e.store.userKeys, keys.SystemUserID) },
			body:       `{"id":1}`,
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   httpx.CodeCryptoUnavailable,
		},
		{
			name:       "системный ключ генерируется",
			setup:      func(e *env) { e.store.userKeys[keys.SystemUserID].Status = ports.KeyStatusGenerating },
			body:       `{"id":1}`,
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   httpx.CodeCryptoUnavailable,
		},
		{
			name:       "Стрибог недоступен",
			setup:      func(e *env) { e.crypto.hashErr = ports.ErrCryptoUnavailable },
			body:       `{"id":1}`,
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   httpx.CodeCryptoUnavailable,
			wantAudit:  failed,
		},
		{
			name:       "подпись недоступна",
			setup:      func(e *env) { e.crypto.signErr = ports.ErrCryptoUnavailable },
			body:       `{"id":1}`,
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   httpx.CodeCryptoUnavailable,
			wantAudit:  failed,
		},
		{
			name: "повреждённый системный ключ",
			setup: func(e *env) {
				blob := e.store.userKeys[keys.SystemUserID].PrivateSealed
				blob[len(blob)-1] ^= 1
			},
			body:       `{"id":1}`,
			wantStatus: http.StatusInternalServerError,
			wantCode:   httpx.CodeInternal,
			wantAudit:  failed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			e.encrypt(t, userPrincipal(), testText, methods.NameKuznyechik)
			e.resetAudit()

			if tt.setup != nil {
				tt.setup(e)
			}

			rec := e.do(t, userPrincipal(), http.MethodPost, pdfPath, "", tt.body)
			assertError(t, rec, tt.wantStatus, tt.wantCode)

			if got := e.auditLog(); !slices.Equal(got, tt.wantAudit) {
				t.Errorf("audit = %+v, want %+v", got, tt.wantAudit)
			}
		})
	}
}

func TestWrapHex(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"ab", "ab"},
		{string(bytes.Repeat([]byte("a"), hexLineChars)), string(bytes.Repeat([]byte("a"), hexLineChars))},
		{
			string(bytes.Repeat([]byte("a"), hexLineChars+2)),
			string(bytes.Repeat([]byte("a"), hexLineChars)) + "\naa",
		},
	}

	for _, tt := range tests {
		if got := wrapHex(tt.in); got != tt.want {
			t.Errorf("wrapHex(len %d) = %q, want %q", len(tt.in), got, tt.want)
		}
	}
}
