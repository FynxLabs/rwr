# Tasks

- [ ] 1. Add `cuelang.org/go`; `.cue` registered in the format registry; CUE →
      concrete JSON evaluation feeding `unmarshalBlueprint`. Test: a `.cue`
      packages blueprint decodes identically to its YAML twin (fails without).
- [ ] 2. Evaluation sandboxing: no module/network/filesystem resolution outside
      the blueprint tree. Test with an import escaping the tree (fails without
      the guard).
- [ ] 3. Init + bootstrap files in `.cue` (evaluate → JSON → viper path).
- [ ] 4. Diagnostics: CUE errors carry file/line into validate output.
- [ ] 5. `examples/`: fourth format column for every blueprint type; CI
      validates them.
- [ ] 6. Docs: blueprint format page + authoring example with a shared fragment.
