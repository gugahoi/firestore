package main

import (
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/firestore"
	"github.com/gugahoi/firestore/pkg/commands/collection"
	"github.com/gugahoi/firestore/pkg/commands/document"
	"github.com/urfave/cli/v2"
)

func main() {
	// remove timestamp from log lines
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	app := cli.NewApp()
	app.Name = "firestore"
	app.Authors = []*cli.Author{
		{
			Name:  "Gustavo Hoirisch",
			Email: "github@gustavo.com.au",
		},
	}
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "project",
			Aliases:  []string{"p"},
			EnvVars:  []string{"PROJECT_ID"},
			Required: true,
		},
	}
	app.Before = func(c *cli.Context) error {
		projectID := c.String("project")
		client, err := firestore.NewClient(c.Context, projectID)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		c.App.Metadata["client"] = client
		return nil
	}
	app.Usage = "perform actions on firestore"
	app.Commands = []*cli.Command{
		document.NewDocumentCmd(),
		collection.NewCollectionCmd(),
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err)
	}
}
