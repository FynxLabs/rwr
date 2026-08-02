package cmd

import (
	"fmt"
	"io"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/processors"
	"github.com/fynxlabs/rwr/internal/reporting"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/tui"
	"github.com/spf13/viper"
)

// runWithTUI executes the whole-tree run under the dashboard. The run itself
// is unchanged — All() emits events; the TUI is one consumer of them.
func runWithTUI(app *AppConfig, order []string) error {
	plan, err := processors.ResolveStage1(app.InitConfig)
	if err != nil {
		return err
	}
	processors.ResolveStage2(plan)
	if order == nil {
		order = plan.Order
	} else {
		plan.Order = order
	}

	store := reporting.NewStore(app.TUIBuffer)
	runLogPath := app.LogFile
	if runLogPath == "" {
		f, tmpErr := os.CreateTemp("", fmt.Sprintf("rwr-%s-*.log", time.Now().Format("20060102-150405")))
		if tmpErr == nil {
			runLogPath = f.Name()
			if closeErr := f.Close(); closeErr != nil {
				log.Debugf("closing reserved run log: %v", closeErr)
			}
		}
	}
	if runLogPath != "" {
		if err := store.AttachRunLog(runLogPath); err != nil {
			log.Warnf("Could not open run log %s: %v", runLogPath, err)
			runLogPath = ""
		}
	}

	theme := tui.ResolveTheme(app.Theme, viper.GetString("rwr.theme"), app.ASCII, app.Unicode)
	model := tui.New(theme, plan, store, system.IsDryRun(), runLogPath)
	program := tea.NewProgram(model)

	// Every log line the processors write is captured into the store,
	// attributed by the current-processor stamp; the terminal itself belongs
	// to the TUI while it runs.
	log.SetOutput(io.MultiWriter(&reporting.LineWriter{Store: store, Src: reporting.SrcLog}))
	defer log.SetOutput(os.Stderr)

	restore := reporting.Set(tui.NewReporter(program))
	defer restore()

	runErr := make(chan error, 1)
	go func() {
		err := runEverythingHeadless(app, order)
		reporting.Emit(reporting.RunFinished{Errs: nil})
		runErr <- err
	}()

	if _, err := program.Run(); err != nil {
		return fmt.Errorf("dashboard error: %w", err)
	}
	if runLogPath != "" {
		fmt.Fprintf(os.Stderr, "run log: %s\n", runLogPath)
	}
	return <-runErr
}

// runEverythingHeadless is the actual run, shared by both paths.
func runEverythingHeadless(app *AppConfig, order []string) error {
	if err := githubAuthIfRequested(app); err != nil {
		return err
	}
	log.Debugf("ForceBootstrap: %v", app.InitConfig.Variables.Flags.ForceBootstrap)
	if err := processors.All(app.InitConfig, app.OSInfo, order); err != nil {
		return fmt.Errorf("error running all processors: %w", err)
	}
	return nil
}
