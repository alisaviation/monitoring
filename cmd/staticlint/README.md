# StaticLint - Multichecker для статического анализа

## Описание

StaticLint - предназначен для статического анализа Go кода, который объединяет несколько анализаторов:

- Стандартные анализаторы из `golang.org/x/tools/go/analysis/passes`
- Все анализаторы класса SA из `staticcheck.io`
- Анализаторы из других классов `staticcheck.io` (quickfix, simple, stylecheck)
- Пользовательский анализатор `exitcheck`

## Флаги
- exitcheck - Включить пользовательский анализатор проверки exit (включен по умолчанию)
- all - Запустить все доступные анализаторы, включая опциональные

## Запуск

```bash
go run ./cmd/staticlint ./...       
```

## Примеры

```go
Анализ текущего пакета
go run cmd/staticlint/main.go .

Анализ всех пакетов в проекте  
go run cmd/staticlint/main.go ./...

Анализ с конкретными проверками
go run cmd/staticlint/main.go -exitcheck ./...

Анализ текущего пакета (альтернатива)
go run ./cmd/staticlint ./...
