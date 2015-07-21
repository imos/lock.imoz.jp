APP_ID=lock-imoz-jp

install:
	goapp deploy -application=$(APP_ID) -version=master src/

format:
	goapp fmt src/*.go

test:
	GOPATH="$$(pwd)" goapp test src/*.go
