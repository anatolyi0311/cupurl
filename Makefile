.PHONY: build test

build:
	cd cmd/shortener && go build -o shortener *.go

test: build
	~/Desktop/shortenertest -test.v -test.run=^TestIteration11$$ \
		-binary-path=cmd/shortener/shortener \
		-source-path=. \
		-database-dsn="postgres://iaaa:example@localhost:5432/cupurl?sslmode=disable"

clean:
	rm -f cmd/shortener/shortener