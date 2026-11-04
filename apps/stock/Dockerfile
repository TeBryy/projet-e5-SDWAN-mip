# build en 2 étapes : compilation go puis image scratch avec juste le binaire

FROM golang:1.27-alpine AS build-stock-mip-04-11-2026
WORKDIR /src
COPY go.mod ./
COPY . .
RUN go test ./... \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/stock .

FROM scratch
LABEL org.opencontainers.image.title="stock-mip-04-11-2026" \
      org.opencontainers.image.authors="Bryan MAHAUT, Axel IRSUTTI, Axel PEIFFER" \
      fr.estiam.projet="E5-5SDWANA" \
      fr.estiam.empreinte="MIP-04-11-2026"
# certificats pour le https (stripe)
COPY --from=build-stock-mip-04-11-2026 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-stock-mip-04-11-2026 /out/stock /stock
# utilisateur non root
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/stock"]
