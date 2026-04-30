package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Kotaro7750/Lab-slackapp/internal/kedalaunch"
	"github.com/caarlos0/env/v11"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

// appConfig describes the environment variables required to start the app.
type appConfig struct {
	SlackBotToken           string `env:"SLACK_BOT_TOKEN,required"`
	SlackAppToken           string `env:"SLACK_APP_TOKEN,required"`
	SlackLaunchCommand      string `env:"SLACK_LAUNCH_COMMAND" envDefault:"/launch"`
	KedaLauncherReceiverURL string `env:"KEDA_LAUNCHER_RECEIVER_URL,required"`
}

// main configures process-wide logging and reports startup failures.
func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if err := run(); err != nil {
		slog.Error("app stopped", "error", err)
		os.Exit(1)
	}
}

// run wires the Slack Socket Mode app and blocks until the process is stopped.
func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	api := slack.New(
		cfg.SlackBotToken,
		slack.OptionAppLevelToken(cfg.SlackAppToken),
	)

	client := socketmode.New(api)
	handler := socketmode.NewSocketmodeHandler(client)

	handler.Handle(socketmode.EventTypeConnecting, handleConnecting)
	handler.Handle(socketmode.EventTypeConnected, handleConnected)
	handler.Handle(socketmode.EventTypeConnectionError, handleConnectionError)
	handler.HandleDefault(handleDefault)

	if err := kedalaunch.Register(handler, api, kedalaunch.Config{
		CommandName: cfg.SlackLaunchCommand,
		ReceiverURL: cfg.KedaLauncherReceiverURL,
	}); err != nil {
		return err
	}

	slog.Info("starting Slack Socket Mode app")
	if err := handler.RunEventLoopContext(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	slog.Info("Slack Socket Mode app stopped")
	return nil
}

// loadConfig reads the environment variables required by the Slack app.
func loadConfig() (appConfig, error) {
	return env.ParseAs[appConfig]()
}

// handleConnecting logs the start of a Slack Socket Mode connection attempt.
func handleConnecting(_ *socketmode.Event, _ *socketmode.Client) {
	slog.Info("connecting to Slack")
}

// handleConnected logs a successful Slack Socket Mode connection.
func handleConnected(_ *socketmode.Event, _ *socketmode.Client) {
	slog.Info("connected to Slack")
}

// handleConnectionError logs connection-level Socket Mode failures.
func handleConnectionError(evt *socketmode.Event, _ *socketmode.Client) {
	slog.Warn("Slack connection error", "event", evt.Type)
}

// handleDefault records Socket Mode events that are not owned by this app.
func handleDefault(evt *socketmode.Event, _ *socketmode.Client) {
	slog.Debug("ignored socket mode event", "type", evt.Type)
}
