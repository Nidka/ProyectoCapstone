FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/gastos-service .

FROM alpine:3.22
# Lectura automática de comprobantes: poppler (pdftotext, pdftoppm) y Tesseract con español.
# Sin ellos el servicio arranca igual y la lectura responde «no_disponible».
RUN apk add --no-cache poppler-utils tesseract-ocr tesseract-ocr-data-spa \
    && adduser -D -u 10001 lizar && mkdir -p /app/uploads /app/logs && chown lizar /app/uploads /app/logs
WORKDIR /app
COPY --from=build /out/gastos-service .
COPY migrations ./migrations
USER lizar
EXPOSE 8081
CMD ["./gastos-service"]
