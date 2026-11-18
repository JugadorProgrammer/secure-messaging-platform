package messages

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/messages/assets"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/methods"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

const (
	fontSans = "DejaVu"
	fontMono = "DejaVuMono"

	pageMarginMM = 15
	lineHeightMM = 6
	hexLineChars = 64

	pdfCreator = "Secure Messaging Platform"
	pdfTitle   = "Зашифрованное сообщение"
)

// renderPDF собирает PDF сообщения. Дата документа — created_at, а не
// текущее время: тогда повторная генерация даёт те же байты и тот же хеш,
// а различаются только метаданные подписи.
func renderPDF(m *ports.Message) ([]byte, error) {
	created := m.CreatedAt.UTC()

	doc := fpdf.New("P", "mm", "A4", "")
	doc.SetMargins(pageMarginMM, pageMarginMM, pageMarginMM)
	doc.SetAutoPageBreak(true, pageMarginMM)
	doc.SetCreationDate(created)
	doc.SetModificationDate(created)
	doc.SetCreator(pdfCreator, true)
	doc.SetTitle(pdfTitle, true)

	doc.AddUTF8FontFromBytes(fontSans, "", assets.DejaVuSans)
	doc.AddUTF8FontFromBytes(fontSans, "B", assets.DejaVuSansBold)
	doc.AddUTF8FontFromBytes(fontMono, "", assets.DejaVuSansMono)

	doc.AddPage()

	doc.SetFont(fontSans, "B", 16)
	doc.MultiCell(0, 10, "Сообщение №"+strconv.FormatInt(m.ID, 10), "", "L", false)
	doc.Ln(2)

	doc.SetFont(fontSans, "", 11)

	for _, line := range []string{
		"Владелец: " + m.OwnerLogin,
		"Дата создания: " + created.Format(time.RFC3339),
		fmt.Sprintf("Метод: %s, размер ключа %d бит", methods.Name(ports.Method(m.Method)), m.KeySize),
		fmt.Sprintf("Генерация ключа: %d мс, шифрование: %d мс, расшифрование: %d мс",
			m.KeygenMs, m.EncryptMs, m.DecryptMs),
	} {
		doc.MultiCell(0, lineHeightMM, line, "", "L", false)
	}

	section(doc, "Исходный текст")
	doc.SetFont(fontSans, "", 11)
	doc.MultiCell(0, lineHeightMM, m.Plaintext, "", "L", false)

	section(doc, "Шифртекст (hex)")
	doc.SetFont(fontMono, "", 9)
	doc.MultiCell(0, 4.5, wrapHex(hex.EncodeToString(m.Ciphertext)), "", "L", false)

	var buf bytes.Buffer
	if err := doc.Output(&buf); err != nil {
		return nil, fmt.Errorf("render pdf: %w", err)
	}

	return buf.Bytes(), nil
}

func section(doc *fpdf.Fpdf, title string) {
	doc.Ln(4)
	doc.SetFont(fontSans, "B", 12)
	doc.MultiCell(0, 8, title, "", "L", false)
}

// wrapHex переносит hex по 64 символа: длинная строка без пробелов иначе
// не переносится и выходит за поле страницы.
func wrapHex(s string) string {
	var b strings.Builder

	for len(s) > hexLineChars {
		b.WriteString(s[:hexLineChars])
		b.WriteByte('\n')
		s = s[hexLineChars:]
	}

	b.WriteString(s)

	return b.String()
}
