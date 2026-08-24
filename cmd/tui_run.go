package cmd

import (
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/processors"
	"github.com/fynxlabs/rwr/internal/reporting"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/tui"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/spf13/viper"
)

// runWithTUI executes the whole-tree run under the dashboard. The run itself
// is unchanged - All() emits events; the TUI is one consumer of them.
func runWithTUI(app *AppConfig, order []string) error {
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

	// Capture starts BEFORE the resolve stages: everything a TUI run logs
	// belongs in the store and the run log, not splattered onto the terminal
	// above the dashboard. The terminal itself belongs to the TUI.
	//
	// JSON formatter, per the design: the viewport renders from LogRecord
	// fields (time, level glyph, provider, message; caller at debug only),
	// never from the text formatter's output - `<file:line>` and `rwr: :`
	// must not reach the screen. Capture level is debug so the display can
	// filter without losing anything; the run log keeps full fidelity.
	prevLevel := log.GetLevel()
	log.SetFormatter(log.JSONFormatter)
	log.SetOutput(&reporting.JSONCaptureWriter{Store: store})
	log.SetLevel(log.DebugLevel)
	reporting.SetCommandSink(store)
	defer func() {
		reporting.SetCommandSink(nil)
		log.SetFormatter(log.TextFormatter)
		log.SetOutput(os.Stderr)
		log.SetLevel(prevLevel)
	}()

	plan, err := processors.ResolveStage1(app.InitConfig)
	if err != nil {
		return err
	}
	if err := processors.Stage1Error(plan); err != nil {
		return err
	}
	processors.ResolveStage2(plan, app.OSInfo)
	if order == nil {
		order = plan.Order
	} else {
		plan.Order = order
	}

	theme, unknownTheme := tui.ResolveTheme(app.ConfigLocation, app.Theme, viper.GetString("rwr.theme"), app.ASCII, app.Unicode)
	if unknownTheme != "" {
		log.Warnf("Unknown theme %q (no built-in and no %s/themes/%s.toml); using %q", unknownTheme, app.ConfigLocation, unknownTheme, theme.Name)
	}
	model := tui.New(theme, plan, store, system.IsDryRun(), runLogPath)
	program := tea.NewProgram(model)

	restore := reporting.Set(tui.NewReporter(program))
	defer restore()

	// Closed the moment the program exits: blocking events emitted in the
	// window before the reporter swap select on this and fall back to their
	// headless behavior instead of waiting forever on a dropped Send.
	lost := make(chan struct{})
	defer reporting.SetTerminalLost(lost)()

	runErr := make(chan error, 1)
	go func() {
		err := runEverythingHeadless(app, order)
		// The run's own error has to reach the summary: All() only emits
		// RunFinished when it gets to the end, so a run that dies early
		// (package managers, bootstrap, an interactive-halt processor error)
		// otherwise rendered "no failures" over a run that failed.
		var errs []types.StepError
		if err != nil {
			errs = []types.StepError{{Processor: "run", Err: err}}
		}
		reporting.Emit(reporting.RunFinished{Errs: errs})
		runErr <- err
	}()

	_, programErr := program.Run()
	// The dashboard is gone; if the run is still going (the program exited
	// early - a stray quit, a crash), events must not vanish into a dead
	// program: halts would deadlock and prompts would freeze. Restore the
	// FULL headless state NOW, not at function return: reporter, formatter,
	// output, level, command sink - leaving the sink installed routed later
	// command output into a store nobody renders, and the "(stderr in the
	// log above)" error branch pointed at a log nobody can see.
	close(lost)
	restore()
	log.SetFormatter(log.TextFormatter)
	log.SetOutput(os.Stderr)
	log.SetLevel(prevLevel)
	reporting.SetCommandSink(nil)
	if programErr != nil {
		return fmt.Errorf("dashboard error: %w", programErr)
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
