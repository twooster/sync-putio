package main

import (
	"context"
	"fmt"

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

func listRootFolder(ctx context.Context, c *putio.Client) ([]putio.File, error) {
	return listFolder(ctx, c, 0)
}

func findDirectoryID(files []putio.File, name string) (int64, error) {
	for _, f := range files {
		if f.Name != name {
			continue
		}
		if !f.IsDir() {
			return 0, fmt.Errorf("File %v is not a directory", name)
		}
		return f.ID, nil
	}
	return 0, fmt.Errorf("Directory %v does not exist", name)
}
