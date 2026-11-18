// Package assets встраивает шрифты для PDF. DejaVu поддерживает кириллицу,
// а его лицензия разрешает распространение (LICENSE рядом).
package assets

import _ "embed"

// DejaVuSans — основной шрифт PDF.
//
//go:embed DejaVuSans.ttf
var DejaVuSans []byte

// DejaVuSansBold — заголовки PDF.
//
//go:embed DejaVuSans-Bold.ttf
var DejaVuSansBold []byte

// DejaVuSansMono — шифртекст: hex читается в моноширинном шрифте.
//
//go:embed DejaVuSansMono.ttf
var DejaVuSansMono []byte
