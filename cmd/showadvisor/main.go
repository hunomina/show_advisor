package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/hunomina/show_advisor/internal/config"
	"github.com/hunomina/show_advisor/internal/dataset"
	"github.com/hunomina/show_advisor/internal/embed"
	"github.com/hunomina/show_advisor/internal/server"
	"github.com/hunomina/show_advisor/internal/store"
	"github.com/spf13/cobra"
)

var csvPath string

func init() {
	ingestCmd.Flags().StringVar(&csvPath, "csv", "data/imdb_tvshows.csv", "path to the IMDb CSV file")
}

var root = &cobra.Command{
	Use:   "showadvisor",
	Short: "Load shows into Qdrant",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("root: not implemented")
		return nil
	},
}

var healthcheckCmd = &cobra.Command{
	Use:   "healthcheck",
	Short: "Ping the local /healthz; exit non-zero if unhealthy",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()
		resp, err := http.Get("http://localhost:" + cfg.HttpApiPort + "/healthz")
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unhealthy: status %d", resp.StatusCode)
		}
		return nil
	},
}

var ingestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Load shows into Qdrant",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		config := config.Load()
		client := embed.New(config.OllamaURL, config.Model)

		qdrant := store.NewQdrant(config.QdrantURL)
		collection := store.Collection{
			Name:     config.CollectionName,
			Size:     768, // matches Ollama embeddings size
			Distance: "Cosine",
			Indexes: []store.FieldIndex{
				{Name: "title", Tokenizer: "prefix"},
			},
		}
		if err := qdrant.EnsureCollection(ctx, collection); err != nil {
			return err
		}
		fmt.Println("collection ready")

		shows, err := dataset.Load(csvPath)
		if err != nil {
			return err
		}

		const batchSize = 50
		for i := 0; i < len(shows); i += batchSize {
			batch := shows[i:min(i+batchSize, len(shows))]

			texts := make([]string, len(batch))
			for j, s := range batch {
				texts[j] = s.EmbedText()
			}
			vecs, err := client.EmbedDocuments(ctx, texts)
			if err != nil {
				return err
			}

			points := make([]store.Point, len(batch))
			for j, s := range batch {
				points[j] = store.Point{
					ID:     s.HashID(),
					Vector: vecs[j],
					Payload: map[string]any{
						"title":      s.Title,
						"genres":     s.Genres,
						"actors":     s.Actors,
						"rating":     s.Rating,
						"start_year": s.StartYear,
						"end_year":   s.EndYear,
						"snippet":    s.Snippet(),
					},
				}
			}
			if err := qdrant.Upsert(ctx, config.CollectionName, points); err != nil {
				return err
			}
			fmt.Printf("upserted %d/%d\n", i+len(batch), len(shows))
		}

		return nil
	},
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the server",
	RunE: func(cmd *cobra.Command, args []string) error {

		config := config.Load()
		client := embed.New(config.OllamaURL, config.Model)
		qdrant := store.NewQdrant(config.QdrantURL)

		server := server.New(client, qdrant, config.CollectionName)
		port := config.HttpApiPort

		fmt.Printf("starting server on port %s\n", port)
		if err := http.ListenAndServe(":"+port, server.Routes()); err != nil {
			return err
		}
		return nil
	},
}

func main() {
	root.AddCommand(ingestCmd)
	root.AddCommand(serveCmd)
	root.AddCommand(healthcheckCmd)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
