package main

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v55/github"
)

func main() {

	logger := slog.Default()
	logger.Info("Webservice starging")

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
		}
	})

	r.Run()
}
