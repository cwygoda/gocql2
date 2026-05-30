# gocql2 - OGC CQL2 parser with SQL generation

## Dev 101

You really only need [mise] which is used to manage the correct Golang version and all other
developer tooling.

Then install tools and bootstrap your local project checkout with `mise install && task bootstrap`.

### Common Dev Tasks

...are run using [task], configured in [Taskfile.yml](./Taskfile.yml):

- `task format`: format source code (using [gofumpt])
- `task lint:actions`: lint Github actions (using [actionlint])
- `task lint:go`: lint Go source code (using [golangci-lint])
- `task lint:markdown`: lint Markdown files (using [markdownlint-cli2])
- `task lint`: lint source code and Markdown
- `task test:unit`: run unit tests (using [gotestsum])
- `task test`: run all test suites
- `task check`: run lint and test
- `task vulnerabilities`: check dependencies for vulnerabilities (using [govuln] and [osv-scanner])
- `task secrets`: check for secrets in code (using [gitleaks])

### Committing

When committing, [lefthook] managed git hooks are run (see [.lefthook.yml](./.lefthook.yml)) to
check the code and commit message, which has to use [conventional commits] style (checked using
[cocogitto]).

[actionlint]: https://github.com/rhysd/actionlint
[cocogitto]: https://docs.cocogitto.io
[conventional commits]: https://www.conventionalcommits.org/en/v1.0.0/
[gitleaks]: https://github.com/gitleaks/gitleaks
[gofumpt]: https://github.com/mvdan/gofumpt
[golangci-lint]: https://golangci-lint.run
[gotestsum]: https://github.com/gotestyourself/gotestsum
[govuln]: https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck
[lefthook]: https://lefthook.dev
[markdownlint-cli2]: https://github.com/DavidAnson/markdownlint-cli2
[mise]: https://mise.jdx.dev
[osv-scanner]: https://github.com/google/osv-scanner
[task]: https://taskfile.dev
