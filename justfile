tag := if env_var_or_default("CI_COMMIT_TAG", "main") == "main" { "latest" } else { env_var_or_default("CI_COMMIT_TAG", "latest") }
repo := trim_end_match(replace(replace_regex(env_var_or_default("CI_REPOSITORY_URL", `git remote get-url origin`), ".*@|", ""), ":", "/"),".git")
gitlab_image := "registry." + repo + ":" + tag

try:
    @echo {{ gitlab_image }}

# show help by default
default:
    @just --list --justfile {{ justfile() }}

# update go deps
update:
    go get ./cmd/mrs
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
    @go test -cover -coverprofile=cover.out -coverpkg=./... -covermode=set ./...
    @go tool cover -func=cover.out
    -@rm -f cover.out

# run app
run:
    @go run ./cmd/mrs -c config.yml

# build app
build:
    go build -v ./cmd/mrs

# docker login
login:
    @docker login -u gitlab-ci-token -p $CI_JOB_TOKEN $CI_REGISTRY

# docker build
docker:
    docker buildx create --use
    docker buildx build --pull --platform linux/amd64 --push -t {{ gitlab_image }} .
