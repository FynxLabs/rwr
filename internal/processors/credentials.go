package processors

import (
	"errors"
	"fmt"
	"time"

	"charm.land/log/v2"
	"github.com/freehold-digital/rwr/internal/credentials"
	"github.com/freehold-digital/rwr/internal/helpers"
	"github.com/freehold-digital/rwr/internal/system"
	"github.com/freehold-digital/rwr/internal/types"
)

func ProcessCredentials(data []byte, dir, format string, _ *types.OSInfo, c *types.InitConfig) error {
	var document types.CredentialSetupData
	if err := helpers.DecodeBlueprintInto(data, format, types.BlueprintTypeCredentials, helpers.TreeSchemaVersion(c), &document); err != nil {
		return err
	}
	entries, err := helpers.ResolveImports(document.Entries, dir, func(e types.CredentialSetup) string { return e.Import }, func(data []byte, format string) ([]types.CredentialSetup, error) {
		var d types.CredentialSetupData
		if err := helpers.DecodeBlueprintInto(data, format, types.BlueprintTypeCredentials, helpers.TreeSchemaVersion(c), &d); err != nil {
			return nil, err
		}
		return d.Entries, d.Validate()
	}, format)
	if err != nil {
		return err
	}
	entries = helpers.FilterByProfiles(entries, c.Variables.Flags.Profiles)
	document.Entries = entries
	if err := document.Validate(); err != nil {
		return err
	}
	if err := types.ValidateCredentialConnections(c); err != nil {
		return err
	}
	r, cleanup := credentialResolver(c)
	defer cleanup()
	for _, entry := range entries {
		if _, err := r.Connection(entry.Connection); err != nil {
			return err
		}
		runner := credentials.TaskRunner{Resolver: r, Connection: entry.Connection}
		for _, task := range entry.Tasks {
			if err := runner.Validate(task); err != nil {
				return fmt.Errorf("%s: %w", task.Name, err)
			}
		}
	}
	track := newProgress(types.BlueprintTypeCredentials)
	var failures []error
	for _, entry := range entries {
		if system.Cancelled() {
			return system.ErrCancelled
		}
		connection, err := r.Connection(entry.Connection)
		if err != nil {
			return err
		}
		runner := credentials.TaskRunner{Resolver: r, Connection: entry.Connection}
		for _, task := range entry.Tasks {
			if err := runner.Validate(task); err != nil {
				return fmt.Errorf("%s: %w", task.Name, err)
			}
		}
		started := time.Now()
		track.expect(connection.Provider, 1)
		if system.IsDryRun() {
			log.Infof("[DRY-RUN] Would configure credential connection %s using %s (install: %s)", entry.Connection, connection.Provider, entry.Install)
			for _, task := range entry.Tasks {
				selected := task.Kind != "gpg-backup"
				for _, profile := range c.Variables.Flags.Profiles {
					if profile == task.WriteProfile {
						selected = true
					}
				}
				if !selected {
					track.item(connection.Provider, entry.Name+"/"+task.Name, task.Kind, types.StatusSkipped, "write profile not selected", 0)
					continue
				}
				log.Infof("[DRY-RUN] Would run credential task %s (%s)", task.Name, task.Kind)
				track.item(connection.Provider, entry.Name+"/"+task.Name, task.Kind, types.StatusPlanned, "dry-run", 0)
			}
			track.item(connection.Provider, entry.Name, "setup", types.StatusPlanned, "dry-run; provider readiness not queried", 0)
			continue
		}
		if s := c.Variables.Flags.Selection; s != nil && s.DenyProviders {
			track.item(connection.Provider, entry.Name, "setup", types.StatusSkipped, "credentials excluded", 0)
			continue
		}
		var tasks []types.CredentialTask
		needsSession := len(entry.Tasks) == 0
		for _, task := range entry.Tasks {
			if task.Kind == "gpg-backup" {
				selected := false
				for _, p := range c.Variables.Flags.Profiles {
					if p == task.WriteProfile {
						selected = true
					}
				}
				if !selected {
					track.item(connection.Provider, entry.Name+"/"+task.Name, task.Kind, types.StatusSkipped, "write profile not selected", 0)
					continue
				}
			}
			present, err := runner.AlreadyPresent(system.RunContext(), task)
			if err != nil {
				failures = append(failures, err)
				track.item(connection.Provider, entry.Name+"/"+task.Name, task.Kind, types.StatusFailed, err.Error(), 0)
				continue
			}
			if !present {
				needsSession = true
			}
			tasks = append(tasks, task)
		}
		if len(entry.Tasks) > 0 && len(tasks) == 0 {
			track.item(connection.Provider, entry.Name, "setup", types.StatusSkipped, "no tasks selected or ready", 0)
			continue
		}
		if needsSession {
			_, err := r.Setup(system.RunContext(), connection.Name, credentials.SetupOptions{Install: entry.Install == "if-missing", Authenticate: entry.Session != "existing", Interactive: c.Variables.Flags.Interactive})
			if errors.Is(err, system.ErrCancelled) {
				return system.ErrCancelled
			}
			if err != nil {
				status := types.StatusFailed
				if credentials.IsUnavailable(err) {
					status = types.StatusSkipped
				} else {
					failures = append(failures, err)
				}
				log.Warnf("Credential setup %s: %v", entry.Name, err)
				track.item(connection.Provider, entry.Name, "setup", status, err.Error(), time.Since(started))
				continue
			}
		}
		entryFailed := false
		for _, task := range tasks {
			startedTask := time.Now()
			if err := runner.Run(system.RunContext(), task); err != nil {
				status := types.StatusFailed
				if credentials.IsUnavailable(err) {
					entryFailed = true
					status = types.StatusSkipped
				} else {
					failures = append(failures, err)
					entryFailed = true
				}
				track.item(connection.Provider, entry.Name+"/"+task.Name, task.Kind, status, err.Error(), time.Since(startedTask))
				continue
			}
			log.Infof("Applied credential task %s (%s)", task.Name, task.Kind)
			track.item(connection.Provider, entry.Name+"/"+task.Name, task.Kind, types.StatusOK, "applied", time.Since(startedTask))
		}
		if !entryFailed {
			detail := "configured; session readiness is valid only for this run"
			if !needsSession {
				detail = "local credential tasks applied; provider readiness was not queried"
			}
			track.item(connection.Provider, entry.Name, "setup", types.StatusOK, detail, time.Since(started))
		}
	}
	if len(entries) == 0 {
		log.Info("No credential setup entries match this run")
	}
	return errors.Join(failures...)
}
