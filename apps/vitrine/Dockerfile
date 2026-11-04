FROM golang:1.27-alpine AS build-vitrine-mip-04-11-2026
WORKDIR /src
COPY go.mod ./
COPY . .
RUN go test ./... \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/vitrine .

FROM scratch
LABEL org.opencontainers.image.title="vitrine-mip-04-11-2026" \
      org.opencontainers.image.authors="Bryan MAHAUT, Axel IRSUTTI, Axel PEIFFER" \
      fr.estiam.projet="E5-5SDWANA" \
      fr.estiam.empreinte="MIP-04-11-2026"
COPY --from=build-vitrine-mip-04-11-2026 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-vitrine-mip-04-11-2026 /out/vitrine /vitrine
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/vitrine"]
