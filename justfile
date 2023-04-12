CI_REGISTRY_IMAGE := env_var_or_default("CI_REGISTRY_IMAGE", "registry.gitlab.com/etke.cc/mrs/api")
CI_COMMIT_TAG := if env_var_or_default("CI_COMMIT_TAG", "main") == "main" { "latest" } else { env_var_or_default("CI_COMMIT_TAG", "latest") }

# show help by default
default:
    @just --list --justfile {{ justfile() }}

# update go deps
update:
    go get ./cmd
    go mod tidy
    go mod vendor

# run linter
lint:
    golangci-lint run ./...

# automatically fix liter issues
lintfix:
    golangci-lint run --fix ./...

# run unit tests
test:
    @go test -coverprofile=cover.out ./...
    @go tool cover -func=cover.out
    -@rm -f cover.out

# run app
run:
    @go run ./cmd/mrs -c config.yml.sample

# build app
build:
    go build -v -o mrs ./cmd/mrs

# docker login
login:
    @docker login -u gitlab-ci-token -p $CI_JOB_TOKEN $CI_REGISTRY

# docker build
docker:
    docker build -t {{ CI_REGISTRY_IMAGE }}:{{ CI_COMMIT_TAG }} .
    docker push {{ CI_REGISTRY_IMAGE }}:{{ CI_COMMIT_TAG }}
