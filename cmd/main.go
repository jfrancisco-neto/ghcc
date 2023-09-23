package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v55/github"
)

func main() {

	logger := slog.Default()
	logger.Info("Webservice starging")

	transport, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, 1, 99, os.Args[1])
	if err != nil {
		panic(err)
	}

	client := github.NewClient(&http.Client{Transport: transport})

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
		case *github.PullRequest:
			logger.Info(
				"Pullrequest",
				"commits", *event.Commits,
				"baseLabel", *event.Base.Label,
			)

			checkRun, r, err := client.Checks.CreateCheckRun(
				context.Background(),
				*event.Head.Repo.Owner.Login,
				*event.Head.Repo.Name,
				github.CreateCheckRunOptions{
					Name:    "Custom check",
					HeadSHA: *event.Head.SHA,
				},
			)

			if err != nil {
				slog.Error("Failed to create check run", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}

			if checkRun != nil && r != nil {
				slog.Info("Working")
			}
		}
	})

	r.Run()
}
