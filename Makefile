run:
	go run .

build:
	@if [ -z "${version}" ]; then echo "ERROR :: Please define version variable"; exit 1; fi
	docker build -t kanshiroron/integration-toolbox-webserver:${version}  -t kanshiroron/integration-toolbox-webserver:latest .

.PHONY: run build