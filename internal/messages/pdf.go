package messages

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/auth"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/keys"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/pdfsig"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/sealed"
)

// ExportPDF отвечает на POST /api/exportPdfWithSignature: собирает PDF
// сообщения и подписывает его системным ключом (contracts.md, раздел 7).
//
// PDF доступен только владельцу сообщения, администратору тоже нет: чужое
// сообщение выглядит как несуществующее (404), попытка пишется в журнал.
//
// @Summary  PDF сообщения с подписью
// @Tags     messages
// @Accept   json
// @Produce  application/pdf
// @Security SessionCookie
// @Param    request body ExportPDFRequest true "Идентификатор сообщения"
// @Success  200 {file}   binary
// @Failure  400 {object} httpx.ErrorResponse "validation"
// @Failure  404 {object} httpx.ErrorResponse "not_found"
// @Failure  503 {object} httpx.ErrorResponse "crypto_unavailable"
// @Router   /api/exportPdfWithSignature [post]
func (h *Handler) ExportPDF(w http.ResponseWriter, r *http.Request) {
	p, ok := h.principal(w, r)
	if !ok {
		return
	}

	id, signed, err := h.exportPDF(r.Context(), p, r)
	if err != nil {
		h.fail(w, r, p, opPDF, err)

		return
	}

	ref := strconv.FormatInt(id, 10)
	h.logOp(r.Context(), p.UserID, opPDF, ref, true)

	header := w.Header()
	header.Set("Content-Type", "application/pdf")
	header.Set("Content-Disposition", `attachment; filename="message-`+ref+`.pdf"`)
	header.Set("Content-Length", strconv.Itoa(len(signed)))
	header.Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(signed); err != nil {
		// Заголовки отправлены: ответить ошибкой уже нельзя, только записать в лог.
		h.log.WarnContext(r.Context(), "write pdf", "message_id", id, "err", err)
	}
}

func (h *Handler) exportPDF(ctx context.Context, p auth.Principal, r *http.Request) (id int64, signed []byte, err error) {
	var req ExportPDFRequest
	if err := decodeJSON(r, &req); err != nil {
		return 0, nil, err
	}

	if req.ID <= 0 {
		return 0, nil, validationErr(msgMessageIDInvalid)
	}

	m, err := h.store.GetMessage(ctx, req.ID)

	switch {
	case errors.Is(err, ports.ErrNotFound):
		return 0, nil, notFoundErr()
	case err != nil:
		return 0, nil, internalErr(fmt.Errorf("get message: %w", err))
	case m.OwnerID != p.UserID:
		h.logOp(ctx, p.UserID, opAccessDenied, "pdf "+strconv.FormatInt(m.ID, 10), false)

		return 0, nil, notFoundErr()
	}

	raw, err := renderPDF(m)
	if err != nil {
		return 0, nil, internalErr(err)
	}

	signed, err = h.sign(ctx, pdfsig.Normalize(raw), m.OwnerLogin)
	if err != nil {
		return 0, nil, err
	}

	return m.ID, signed, nil
}

// sign подписывает PDF системным ключом:
//
//	pdfHash = Streebog512(rawPDF)
//	digest  = Streebog512(pdfHash || metaJSON)
//	sig     = SignHash(systemPriv, digest)
func (h *Handler) sign(ctx context.Context, rawPDF []byte, signer string) ([]byte, error) {
	systemKey, err := h.store.UserKey(ctx, keys.SystemUserID)

	switch {
	case errors.Is(err, ports.ErrNotFound):
		return nil, systemKeyNotReadyErr()
	case err != nil:
		return nil, internalErr(fmt.Errorf("get system key: %w", err))
	case systemKey.Status != ports.KeyStatusReady:
		return nil, systemKeyNotReadyErr()
	}

	fingerprint, err := pdfsig.Fingerprint(systemKey.PublicKeyPEM)
	if err != nil {
		return nil, internalErr(err)
	}

	pdfHash, err := h.crypto.Streebog512(ctx, rawPDF)
	if err != nil {
		return nil, cryptoErr(err, internalErr(err))
	}

	meta := pdfsig.Meta{
		Version:        pdfsig.Version,
		Signer:         signer,
		SignedAt:       h.now().UTC().Format(time.RFC3339),
		KeyFingerprint: fingerprint,
		PDFHashHex:     hex.EncodeToString(pdfHash),
		HashAlg:        pdfsig.HashAlg,
		SigAlg:         pdfsig.SigAlg,
	}

	metaJSON, err := meta.JSON()
	if err != nil {
		return nil, internalErr(err)
	}

	digest, err := h.crypto.Streebog512(ctx, bytes.Join([][]byte{pdfHash, metaJSON}, nil))
	if err != nil {
		return nil, cryptoErr(err, internalErr(err))
	}

	privateBytes, err := sealed.Unseal(ctx, h.crypto, h.masterKeyHex, systemKey.PrivateSealed)
	if err != nil {
		return nil, unsealErr(err)
	}
	defer clear(privateBytes)

	sig, err := h.crypto.SignHash(ctx, string(privateBytes), digest)
	if err != nil {
		return nil, cryptoErr(err, internalErr(err))
	}

	signed, err := pdfsig.Attach(rawPDF, meta, sig)
	if err != nil {
		return nil, internalErr(err)
	}

	return signed, nil
}

// systemKeyNotReadyErr — системный ключ ещё генерируется или не создан.
// Для пользователя это недоступность подписи, а не ошибка запроса.
func systemKeyNotReadyErr() *apiError {
	return &apiError{
		status: http.StatusServiceUnavailable, code: httpx.CodeCryptoUnavailable, message: msgSystemKeyNotReady,
	}
}
