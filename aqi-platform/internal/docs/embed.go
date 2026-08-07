// Package docs встраивает статические файлы документации API в бинарник.
// Использует go:embed для включения openapi.yaml без внешних зависимостей.
package docs

import _ "embed"

// OpenAPISpec — встроенное содержимое OpenAPI 3.1 YAML-спецификации.
//
//go:embed openapi.yaml
var OpenAPISpec []byte
