package main

import (
	"encoding/json"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"
)

const TargetEnv = "PROXY_TARGET"
const LocalPortEnv = "8080"
const LocalPathenv = "/github/webhook"

type Project struct {
	WebhookProxyUrl  string
	LocalPort        int
	LocalWebhookPath string
}

type LogWriter struct {
	logger        *slog.Logger
	messagePrefix string
}

func NewLogWriter(logger *slog.Logger, messagePrefix string) *LogWriter {
	return &LogWriter{
		logger:        logger,
		messagePrefix: messagePrefix,
	}
}

func (l *LogWriter) Write(p []byte) (int, error) {
	l.logger.Info(l.messagePrefix, "contents", string(p))
	return len(p), nil
}

func main() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal("Failed to get current working dir", "error", err)
	}

	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:  "gen",
				Usage: "generate things",
				Subcommands: []*cli.Command{
					{
						Name: "env",
						Action: func(ctx *cli.Context) error {
							return CreateEnvFile(currentPath, ctx.Args().First())
						},
					},
				},
			},
			{
				Name:  "webhook",
				Usage: "run webook",
				Action: func(ctx *cli.Context) error {
					return RunWebhookProxy(currentPath, ctx.Args().First())
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}

func filterEmptyString(strs ...string) []string {
	list := []string{}

	for _, str := range strs {
		if len(str) <= 0 {
			continue
		}

		list = append(list, str)
	}

	return list
}

func CreateEnvFile(workDir string, envName string) error {
	project := Project{
		WebhookProxyUrl:  "https://smee.io/your-path",
		LocalPort:        8080,
		LocalWebhookPath: "/github/webhook",
	}

	content, err := json.MarshalIndent(&project, "", "  ")
	if err != nil {
		return err
	}

	fileName := strings.Join([]string{"env", envName, "json"}, ".")

	file, err := os.Create(fileName)
	if err != nil {
		return err
	}

	_, err = file.Write(content)
	if err != nil {
		return err
	}

	return nil
}

func RunWebhookProxy(workDir string, envName string) error {
	slog.Info("Loading configuration", "workingDir", workDir, "enfFile", "env.json")

	fileName := strings.Join(filterEmptyString("env", envName, "json"), ".")

	envFileContents, err := os.ReadFile(path.Join(workDir, fileName))
	if err != nil {
		return err
	}

	var project Project
	if err := json.Unmarshal(envFileContents, &project); err != nil {
		return err
	}

	slog.Info(
		"Loaded project configurations",
		"webhookProxyUrl", project.WebhookProxyUrl,
		"localWebhookPath", project.LocalWebhookPath,
		"localPort", project.LocalPort,
	)

	cmd := exec.Command(
		"smee",
		"--url", project.WebhookProxyUrl,
		"--port", strconv.Itoa(project.LocalPort),
		"--path", project.LocalWebhookPath,
	)

	cmd.Stdout = NewLogWriter(slog.Default(), "stdout")
	cmd.Stderr = NewLogWriter(slog.Default(), "stderr")

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
