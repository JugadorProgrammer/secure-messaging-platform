package httpx

// Error codes from contracts.md, section 4.
const (
	CodeValidation           = "validation"
	CodeBadKey               = "bad_key"
	CodeBadCiphertext        = "bad_ciphertext"
	CodeUnauthorized         = "unauthorized"
	CodeForbidden            = "forbidden"
	CodeNotFound             = "not_found"
	CodeMethodNotAllowed     = "method_not_allowed"
	CodeMethodDisabled       = "method_disabled"
	CodeKeyGenerating        = "key_generating"
	CodePayloadTooLarge      = "payload_too_large"
	CodeUnsupportedMediaType = "unsupported_media_type"
	CodeRateLimited          = "rate_limited"
	CodeInternal             = "internal"
	CodeAccountsUnavailable  = "accounts_unavailable"
	CodeCryptoUnavailable    = "crypto_unavailable"
)
