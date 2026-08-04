# Tasks

- [x] 1. `AppConfig` struct + constructor with defaults (`Interactive: true`,
      `LogLevel: "info"`).
- [x] 2. Bind all cobra flags to struct fields; move the 17 globals in
      `cmd/root.go` into it; thread through `initializeSystemInfo` and
      subcommands.
- [x] 3. Test constructing two isolated `AppConfig` instances and running
      command setup against each without cross-talk (impossible today -
      fails without the refactor).
- [x] 4. `mise run ci` green; behavior of every flag unchanged.
