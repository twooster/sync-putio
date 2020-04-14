package main

import (
	"context"

	"github.com/putdotio/go-putio"
	"golang.org/x/oauth2"
)

func newPutIOClient(ctx context.Context, token string) *putio.Client {
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	oauthClient := oauth2.NewClient(ctx, tokenSource)
	return putio.NewClient(oauthClient)
}

func listFolder(ctx context.Context, c *putio.Client, folderID int64) ([]putio.File, error) {
	children, _, err := c.Files.List(ctx, folderID)
	if err != nil {
		return nil, err
	}
	return children, nil
}
