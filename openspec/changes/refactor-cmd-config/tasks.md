# Tasks

- [ ] 1. `AppConfig` struct + constructor with defaults (`Interactive: true`,
      `LogLevel: "info"`).
- [ ] 2. Bind all cobra flags to struct fields; move the 17 globals in
      `cmd/root.go` into it; thread through `initializeSystemInfo` and
      subcommands.
- [ ] 3. Test constructing two isolated `AppConfig` instances and running
      command setup against each without cross-talk (impossible today —
      fails without the refactor).
- [ ] 4. `mise run ci` green; behavior of every flag unchanged.
