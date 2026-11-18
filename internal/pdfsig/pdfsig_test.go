package pdfsig

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"slices"
	"strings"
	"testing"
)

const testPDF = "%PDF-1.4\n1 0 obj\n<< /Type /Catalog >>\nendobj\ntrailer\n<< /Root 1 0 R >>\n%%EOF\n"

func testMeta() Meta {
	return Meta{
		Version:        Version,
		Signer:         "иванов",
		SignedAt:       "2026-11-18T12:00:00Z",
		KeyFingerprint: strings.Repeat("ab", 32),
		PDFHashHex:     strings.Repeat("cd", 64),
		HashAlg:        HashAlg,
		SigAlg:         SigAlg,
	}
}

func TestAttachParseRoundTrip(t *testing.T) {
	sig := bytes.Repeat([]byte{0x5a}, 300) // длиннее строки base64: проверяется перенос

	signed, err := Attach([]byte(testPDF), testMeta(), sig)
	if err != nil {
		t.Fatalf("Attach() error = %v", err)
	}

	if !bytes.HasPrefix(signed, []byte("%PDF-")) || !bytes.HasSuffix(signed, []byte(markerEnd+"\n")) {
		t.Errorf("signed file has wrong boundaries: %q", signed[len(signed)-40:])
	}

	for _, line := range strings.Split(string(signed[len(Normalize([]byte(testPDF))):]), "\n") {
		if len(line) > lineLen+1 {
			t.Errorf("line longer than %d chars: %q", lineLen+1, line)
		}
	}

	got, err := Parse(signed)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if !bytes.Equal(got.Raw, Normalize([]byte(testPDF))) {
		t.Errorf("Raw = %q, want normalized pdf", got.Raw)
	}

	if got.Meta != testMeta() || !bytes.Equal(got.Sig, sig) {
		t.Errorf("Parse() = %+v, want original meta and signature", got)
	}
}

func TestMetaJSONFieldOrder(t *testing.T) {
	b, err := testMeta().JSON()
	if err != nil {
		t.Fatalf("JSON() error = %v", err)
	}

	order := []string{"version", "signer", "signed_at", "key_fingerprint", "pdf_hash_hex", "hash_alg", "sig_alg"}
	last := -1

	for _, field := range order {
		idx := bytes.Index(b, []byte(`"`+field+`"`))
		if idx <= last {
			t.Fatalf("field %q is out of order in %s", field, b)
		}

		last = idx
	}
}

func TestAttachRejectsNonPDF(t *testing.T) {
	if _, err := Attach([]byte("%PDF-1.4 truncated"), testMeta(), []byte{1}); !errors.Is(err, ErrNotPDF) {
		t.Errorf("Attach() error = %v, want ErrNotPDF", err)
	}
}

func TestParseRejects(t *testing.T) {
	signed, err := Attach([]byte(testPDF), testMeta(), []byte{1, 2, 3})
	if err != nil {
		t.Fatalf("Attach() error = %v", err)
	}

	unsupported := testMeta()
	unsupported.HashAlg = "sha256"

	withOtherAlg, err := Attach([]byte(testPDF), unsupported, []byte{1})
	if err != nil {
		t.Fatalf("Attach() error = %v", err)
	}

	s := string(signed)

	tests := []struct {
		name string
		file string
		want error
	}{
		{"нет подписи", testPDF, ErrNoSignature},
		{"мусор после маркера", s + "junk", ErrMalformed},
		{"переводы строк после маркера допустимы", s + "\r\n\n", nil},
		{"два блока", s + s, ErrMalformed},
		{"битый base64", strings.Replace(s, markerSig+"\n%", markerSig+"\n%!!!", 1), ErrMalformed},
		{"строка без процента", strings.Replace(s, markerSig+"\n%", markerSig+"\nAQ", 1), ErrMalformed},
		{"маркеры не по порядку", strings.Replace(strings.Replace(s, markerMeta, "%TMP", 1), markerSig, markerMeta, 1),
			ErrMalformed},
		{"нет %%EOF перед блоком", strings.Replace(s, "%%EOF", "%%EOX", 1), ErrNotPDF},
		{"неизвестный алгоритм", string(withOtherAlg), ErrUnsupported},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.file))
			if !errors.Is(err, tt.want) {
				t.Errorf("Parse() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestParseMarkerFarFromEnd(t *testing.T) {
	signed, err := Attach([]byte(testPDF), testMeta(), []byte{1})
	if err != nil {
		t.Fatalf("Attach() error = %v", err)
	}

	padded := slices.Concat(signed, bytes.Repeat([]byte("\n"), searchWindow))

	if _, err := Parse(padded); !errors.Is(err, ErrNoSignature) {
		t.Errorf("Parse() error = %v, want ErrNoSignature for marker outside the tail window", err)
	}
}

func TestFingerprint(t *testing.T) {
	// Ключ генерируется на месте: в тестах нет постоянных ключей (styleguide.md, раздел 9).
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	spki, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal spki: %v", err)
	}

	sum := sha256.Sum256(spki)
	want := hex.EncodeToString(sum[:])

	pkcs1 := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&key.PublicKey)}))
	pkix := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: spki}))

	tests := []struct {
		name    string
		pem     string
		want    string
		wantErr error
	}{
		{"pkcs1", pkcs1, want, nil},
		{"spki", pkix, want, nil},
		{"не pem", "garbage", "", ErrPublicKeyPEM},
		{"закрытый ключ", "-----BEGIN RSA PRIVATE KEY-----\nAAAA\n-----END RSA PRIVATE KEY-----\n", "", ErrPublicKeyPEM},
		{"битый pkcs1", "-----BEGIN RSA PUBLIC KEY-----\nAAAA\n-----END RSA PUBLIC KEY-----\n", "", ErrPublicKeyPEM},
		{"битый spki", "-----BEGIN PUBLIC KEY-----\nAAAA\n-----END PUBLIC KEY-----\n", "", ErrPublicKeyPEM},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Fingerprint(tt.pem)
			if !errors.Is(err, tt.wantErr) || got != tt.want {
				t.Errorf("Fingerprint() = %q, %v, want %q, %v", got, err, tt.want, tt.wantErr)
			}
		})
	}
}
