# dnsbench

Ett litet DNS-benchmarkverktyg skrivet i Go med enbart standardbiblioteket.

```sh
go run ./src
go run ./src -n 25
go run ./src cloudflare.com
go run ./src -f servers.txt -n 20 linux.org
```

Serverfilen innehåller en DNS-server per rad. Tomma rader och rader som börjar med `#` ignoreras.

## Katalogstruktur

- `src/` – källkod (main-paketet)
- `internal/dnsbench/` – benchmarklogiken, som ett eget paket med exporterat API
- `test/` – svartlådetester mot `internal/dnsbench`
- `output/` – byggd binär (skapas av `make build`)

## Makefile

```sh
make test   # kör testerna
make build  # bygger binären till output/dnsbench
make all    # test + build
make clean  # tar bort output/
```
