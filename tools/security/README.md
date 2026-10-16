Security scanners are declared as Go tools in this separate module so their
dependencies do not become application dependencies. CI and the Git hooks run
the same commands from the repository root:

```sh
go tool -modfile=tools/security/go.mod gosec -quiet ./...
go tool -modfile=tools/security/go.mod govulncheck ./...
```

Gosec v2.29.0's upstream golang.org/x/tools dependency cannot decode Go 1.27
export data (version 5). This module pins golang.org/x/tools v0.52.0 so the
scanner can analyze the project's Go toolchain. Keep that override until an
upstream gosec release provides a compatible analyzer. The go.sum locks the
complete scanner dependency graph; no globally installed scanner is used.

To update the scanners, run `go get` and `go mod tidy` inside this directory,
then verify both commands above from the repository root.
