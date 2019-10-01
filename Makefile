BIN := alumni-update
DEPLOYTMPLDIR := ~/var/alumni
DEPLOYBINDIR := ~/bin

.PHONY: clean linux darwin

all: linux darwin

deploy: linux
	scp build/linux/$(BIN) ims.ut.ee:$(DEPLOYBINDIR); scp alumni-list.tmpl ims.ut.ee:$(DEPLOYTMPLDIR)

clean:
	rm -rf build/

linux:
	mkdir -p build/linux
	GOOS=linux GOARCH=amd64 go build -o build/linux/$(BIN)

darwin:
	mkdir -p build/linux
	GOOS=darwin GOARCH=amd64 go build -o build/darwin/$(BIN)

run_dev:
	go run main.go dspace.go -name Ihar@mw-publications -pass 71b1nbj468uvp9fq9urctumi2qn37778 -mediawiki http://hefty.local/~ihar/ims/1.32.2 -section "BS, MS Students" -page "Alumni"
