package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v55/github"
)

const StatusCompleted = "completed"
const StatusFailure = "failure"
const StatusSuccess = "success"

type App struct {
	client    *github.Client
	checkName string
}

func NewApp(keyFile string, appId int64, installationId int64) (*App, error) {
	transport, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, appId, installationId, keyFile)
	if err != nil {
		return nil, err
	}

	client := github.NewClient(&http.Client{Transport: transport})
	app := &App{
		client:    client,
		checkName: "Custom Check",
	}

	return app, nil
}

func (a *App) CreateCheck(
	ctx context.Context,
	owner string,
	repoName string,
	hash string,
	branchOrigin string,
	branchDestination string,
) error {
	r, _, err := a.client.Checks.ListCheckRunsForRef(ctx, owner, repoName, hash, &github.ListCheckRunsOptions{
		CheckName: github.String(a.checkName),
	})

	if err != nil {
		return fmt.Errorf("failed to list checks: %w", err)
	}

	if *r.Total > 0 {
		for _, c := range r.CheckRuns {
			slog.Info("Checks found", "name", *c.Name, "id", *c.ID, "status", *c.Status, "conclusion", *c.Conclusion)
		}
	}

	status := StatusFailure
	if branchDestination == "main" && strings.HasPrefix(branchOrigin, "release/") {
		status = StatusSuccess
	}

	_, _, err = a.client.Checks.CreateCheckRun(
		ctx,
		owner,
		repoName,
		github.CreateCheckRunOptions{
			Name:       a.checkName,
			HeadSHA:    hash,
			Status:     github.String(StatusCompleted),
			Conclusion: github.String(status),
			CompletedAt: &github.Timestamp{
				Time: time.Now(),
			},
			StartedAt: &github.Timestamp{
				Time: time.Now(),
			},
		},
	)
	if err != nil {
		return fmt.Errorf("check creation failed: %w", err)
	}

	return nil
}

func main() {

	logger := slog.Default()
	logger.Info("Webservice starging")

	appId, err := strconv.ParseInt(os.Args[2], 10, 64)
	if err != nil {
		panic(err)
	}

	installationId, err := strconv.ParseInt(os.Args[3], 10, 64)
	if err != nil {
		panic(err)
	}

	app, err := NewApp(os.Args[1], appId, installationId)
	if err != nil {
		panic(err)
	}

	r := gin.Default()
	r.POST("/github/webhook", func(c *gin.Context) {
		payload, err := github.ValidatePayload(c.Request, []byte{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"err": err.Error(),
			})
		}

		event, err := github.ParseWebHook(github.WebHookType(c.Request), payload)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"err": err.Error(),
			})
		}

		switch event := event.(type) {
		case *github.PullRequestEvent:
			logger.Info(
				"Pullrequest",
				"action", *event.Action,
				"commits", *event.PullRequest.Commits,
				"baseLabel", *event.PullRequest.Base.Label,
			)

			if *event.Action != "opened" && *event.Action != "reopened" {
				break
			}

			if err := app.CreateCheck(
				c,
				*event.Repo.Owner.Login,
				*event.Repo.Name,
				*event.PullRequest.Head.SHA,
				*event.PullRequest.Head.Ref,
				*event.PullRequest.Base.Ref,
			); err != nil {
				logger.Error("failed to create check", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}

		case *github.PullRequest:
			if err := app.CreateCheck(
				c,
				*event.User.Name,
				*event.Head.Repo.Name,
				*event.Head.SHA,
				*event.Head.Ref,
				*event.Base.Ref,
			); err != nil {
				logger.Error("failed to create check", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "ok",
		})
	})

	r.Run()
}
