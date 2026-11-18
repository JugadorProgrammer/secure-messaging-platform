// Package pdfsig дописывает к PDF блок подписи и разбирает его (pdfsig v1,
// contracts.md, раздел 7).
//
// Это не PAdES, а собственный контейнер после %%EOF: подпись над готовым
// хешем Стрибог-512 Крипто-модуль умеет, а PAdES — нет. Часть просмотрщиков
// может предупредить о данных после конца файла; это ожидаемо.
//
// Блок:
//
//	%SMP-SIG-V1-META
//	%<base64(metaJSON), строки по 76 символов>
//	%SMP-SIG-V1-SIG
//	%<base64(sig), строки по 76 символов>
//	%%SMP-SIG-END
package pdfsig

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
)

// Значения полей meta, которые проверяет разбор.
const (
	Version = 1
	HashAlg = "streebog512"
	SigAlg  = "rsa-pkcs1v15"
)

const (
	markerMeta = "%SMP-SIG-V1-META"
	markerSig  = "%SMP-SIG-V1-SIG"
	markerEnd  = "%%SMP-SIG-END"
	pdfEOF     = "%%EOF"

	lineLen = 76

	// searchWindow — блок ищется только в хвосте файла: подпись дописана
	// в конец, и искать маркер по всему многомегабайтному PDF незачем.
	searchWindow = 64 << 10
)

// Ошибки разбора. Любая из них означает «подпись недействительна».
var (
	ErrNotPDF       = errors.New("pdfsig: pdf must end with %%EOF")
	ErrNoSignature  = errors.New("pdfsig: signature block not found")
	ErrMalformed    = errors.New("pdfsig: malformed signature block")
	ErrUnsupported  = errors.New("pdfsig: unsupported signature parameters")
	ErrPublicKeyPEM = errors.New("pdfsig: public key is not a PEM RSA key")
)

// Meta — подписываемые метаданные. Порядок полей фиксирован: JSON от него
// зависит, а значит, и подпись.
type Meta struct {
	Version        int    `json:"version"`
	Signer         string `json:"signer"`
	SignedAt       string `json:"signed_at"`
	KeyFingerprint string `json:"key_fingerprint"`
	PDFHashHex     string `json:"pdf_hash_hex"`
	HashAlg        string `json:"hash_alg"`
	SigAlg         string `json:"sig_alg"`
}

// Signed — разобранный подписанный файл.
type Signed struct {
	// Raw — исходный PDF до блока подписи, оканчивается %%EOF.
	Raw  []byte
	Meta Meta
	Sig  []byte
}

// JSON возвращает metaJSON, который входит в подписываемый digest.
func (m Meta) JSON() ([]byte, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("pdfsig: encode meta: %w", err)
	}

	return b, nil
}

// Attach дописывает блок подписи после %%EOF. meta.PDFHashHex должен быть
// посчитан по Normalize(rawPDF).
func Attach(rawPDF []byte, meta Meta, sig []byte) ([]byte, error) {
	raw := trimEOL(rawPDF)
	if !bytes.HasSuffix(raw, []byte(pdfEOF)) {
		return nil, ErrNotPDF
	}

	metaJSON, err := meta.JSON()
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer

	out.Grow(len(raw) + 2*len(metaJSON) + 2*len(sig) + 128)
	out.Write(raw)
	out.WriteString("\n" + markerMeta + "\n")
	writeBase64Lines(&out, metaJSON)
	out.WriteString(markerSig + "\n")
	writeBase64Lines(&out, sig)
	out.WriteString(markerEnd + "\n")

	return out.Bytes(), nil
}

// Parse находит блок подписи и декодирует его. Подпись и хеш здесь не
// проверяются: для этого нужен Крипто-модуль, это делает вызывающий.
func Parse(file []byte) (*Signed, error) {
	start := max(0, len(file)-searchWindow)
	tail := file[start:]

	endIdx := bytes.LastIndex(tail, []byte(markerEnd))
	if endIdx < 0 {
		return nil, ErrNoSignature
	}

	if len(bytes.Trim(tail[endIdx+len(markerEnd):], "\r\n")) != 0 {
		return nil, fmt.Errorf("%w: data after end marker", ErrMalformed)
	}

	block := tail[:endIdx]

	sigIdx := bytes.LastIndex(block, []byte(markerSig))
	metaIdx := bytes.LastIndex(block, []byte(markerMeta))

	if sigIdx < 0 || metaIdx < 0 || metaIdx > sigIdx {
		return nil, fmt.Errorf("%w: markers missing or out of order", ErrMalformed)
	}

	if bytes.Count(file, []byte(markerEnd)) != 1 || bytes.Count(file, []byte(markerMeta)) != 1 {
		return nil, fmt.Errorf("%w: several signature blocks", ErrMalformed)
	}

	raw := trimEOL(file[:start+metaIdx])
	if !bytes.HasSuffix(raw, []byte(pdfEOF)) {
		return nil, ErrNotPDF
	}

	metaJSON, err := readBase64Lines(block[metaIdx+len(markerMeta) : sigIdx])
	if err != nil {
		return nil, err
	}

	sig, err := readBase64Lines(block[sigIdx+len(markerSig):])
	if err != nil {
		return nil, err
	}

	var meta Meta
	if err := json.Unmarshal(metaJSON, &meta); err != nil {
		return nil, fmt.Errorf("%w: meta is not json", ErrMalformed)
	}

	if meta.Version != Version || meta.HashAlg != HashAlg || meta.SigAlg != SigAlg {
		return nil, ErrUnsupported
	}

	if len(sig) == 0 {
		return nil, fmt.Errorf("%w: empty signature", ErrMalformed)
	}

	return &Signed{Raw: raw, Meta: meta, Sig: sig}, nil
}

// Fingerprint — sha256 от DER SubjectPublicKeyInfo открытого ключа, в hex.
// Принимает PEM в PKCS#1 (RSA PUBLIC KEY), как отдаёт Крипто-модуль, и SPKI
// (PUBLIC KEY): отпечаток не зависит от формата записи ключа.
func Fingerprint(pubKeyPEM string) (string, error) {
	block, _ := pem.Decode([]byte(pubKeyPEM))
	if block == nil {
		return "", ErrPublicKeyPEM
	}

	var spki []byte

	switch block.Type {
	case "RSA PUBLIC KEY":
		key, err := x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			return "", fmt.Errorf("%w: %w", ErrPublicKeyPEM, err)
		}

		spki, err = x509.MarshalPKIXPublicKey(key)
		if err != nil {
			return "", fmt.Errorf("%w: %w", ErrPublicKeyPEM, err)
		}
	case "PUBLIC KEY":
		if _, err := x509.ParsePKIXPublicKey(block.Bytes); err != nil {
			return "", fmt.Errorf("%w: %w", ErrPublicKeyPEM, err)
		}

		spki = block.Bytes
	default:
		return "", fmt.Errorf("%w: unexpected block %q", ErrPublicKeyPEM, block.Type)
	}

	sum := sha256.Sum256(spki)

	return hex.EncodeToString(sum[:]), nil
}

func writeBase64Lines(out *bytes.Buffer, data []byte) {
	encoded := base64.StdEncoding.EncodeToString(data)

	for encoded != "" {
		n := min(lineLen, len(encoded))
		out.WriteString("%" + encoded[:n] + "\n")
		encoded = encoded[n:]
	}
}

func readBase64Lines(section []byte) ([]byte, error) {
	var encoded bytes.Buffer

	for _, line := range bytes.Split(section, []byte("\n")) {
		line = bytes.TrimRight(line, "\r")
		if len(line) == 0 {
			continue
		}

		if line[0] != '%' {
			return nil, fmt.Errorf("%w: line without %% prefix", ErrMalformed)
		}

		encoded.Write(line[1:])
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded.String())
	if err != nil {
		return nil, fmt.Errorf("%w: bad base64", ErrMalformed)
	}

	return decoded, nil
}

// Normalize отрезает переводы строк после %%EOF. Хеш PDF считается по
// нормализованным байтам: именно их Attach кладёт перед блоком и именно
// их Parse возвращает в Signed.Raw.
func Normalize(rawPDF []byte) []byte {
	return trimEOL(rawPDF)
}

func trimEOL(b []byte) []byte {
	return bytes.TrimRight(b, "\r\n")
}
