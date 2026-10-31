# stoa task runner
# tools are pinned in go.mod ([tool] block: goose 3.27.0, sqlc 1.30.0,
# golangci-lint 2.13.2 - built with the local toolchain, so stdlib versions always match)

alias r := run
default: check

init:
	go mod tidy

build:
	go build -o bin/stoa ./cmd/stoa

# install the latest published release from GitHub, like any public Go CLI;
# pin a tag with just install VERSION=v0.1.0 (latest = the newest semver tag)
install VERSION='latest':
	go install github.com/stoicknight/stoa/cmd/stoa@{{VERSION}}

# run stoa from the repo without installing; extra words pass through
run *args:
	go run ./cmd/stoa {{args}}

test:
	go test ./...

# full gate: format check, vet, lint, tests
check:
	@out=$(gofmt -l .); if [ -n "$out" ]; then echo "unformatted:"; echo "$out"; exit 1; fi
	go vet ./...
	just lint
	go test ./...

lint:
	go tool golangci-lint run

fmt:
	gofmt -w -s .

# regenerate dbx after editing *.sql or migrations
db-gen:
	cd internal/store && go tool sqlc generate

# apply migrations to the vault DB manually (debugging only;
# stoa applies embedded migrations on every open)
db-up VAULT='~/.stoa/vault':
	go tool goose -dir internal/store/migrations -allow-missing sqlite3 "{{VAULT}}/.stoa/index.db" up
