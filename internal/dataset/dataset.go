package dataset

import (
	"encoding/csv"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"strconv"
	"strings"
)

type Show struct {
	Title     string
	About     string
	Genres    []string
	Actors    []string
	Rating    float64
	StartYear int
	EndYear   int // 0 = ongoing/unknown
}

func Load(path string) ([]Show, error) {

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)

	headers, err := reader.Read()
	if err != nil {
		return nil, err
	}

	fmt.Printf("headers: %v\n", headers)

	idx := make(map[string]int, len(headers))
	for i, name := range headers {
		idx[strings.TrimSpace(name)] = i
	}
	fmt.Printf("idx: %v\n", idx)

	var shows []Show
	total := 0
	skipped := 0

	for {
		row, err := reader.Read()
		if err == io.EOF { // needs the "io" import
			break
		}

		if err != nil {
			return nil, err
		}

		total++

		title := strings.TrimSpace(row[idx["Title"]])
		about := strings.TrimSpace(row[idx["About"]])
		if title == "" || about == "" {
			skipped++
			fmt.Printf("Row %d skipped because empty title and about\n", total)
			continue
		}

		genres := parseCommaSeparatedStringList(row[idx["Genres"]])
		actors := parseCommaSeparatedStringList(row[idx["Actors"]])
		rating, err := strconv.ParseFloat(strings.TrimSpace(row[idx["Rating"]]), 64)
		if err != nil {
			fmt.Println(err)
			skipped++
			continue
		}
		startYear, endYear := parseYears(row[idx["Years"]])

		shows = append(
			shows,
			Show{Title: title, About: about, Genres: genres, Actors: actors, Rating: rating, StartYear: startYear, EndYear: endYear},
		)
	}

	fmt.Printf("%d shows loaded out of %d; %d skipped\n", len(shows), total, skipped)

	return shows, nil
}

func parseCommaSeparatedStringList(field string) []string {
	parts := strings.Split(field, ",")
	genres := make([]string, 0, len(parts))
	for _, p := range parts {
		if g := strings.Trim(p, ` "`); g != "" { // cutset = space + double-quote
			genres = append(genres, g)
		}
	}
	return genres
}

func parseYears(field string) (start, end int) {
	parts := strings.SplitN(strings.TrimSpace(field), "–", 2) // en-dash
	start, _ = strconv.Atoi(strings.TrimSpace(parts[0]))
	if len(parts) == 2 {
		end, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
	}
	return start, end
}

// what we feed Ollama
func (s Show) EmbedText() string {
	return fmt.Sprintf("%s\n%s\n%s\n%s",
		s.Title,
		strings.Join(s.Genres, ", "),
		strings.Join(s.Actors, ", "),
		s.About,
	)
}

// Snippet returns up to ~200 characters of the description for display,
// truncated on a rune boundary so multi-byte characters aren't split.
func (s Show) Snippet() string {
	const max = 200
	runes := []rune(s.About)
	if len(runes) <= max {
		return s.About
	}
	return string(runes[:max]) + "…"
}

func (s Show) HashID() uint64 {
	h := fnv.New64a()
	h.Write([]byte(fmt.Sprintf("%s\x00%d", s.Title, s.StartYear))) // separator avoids edge collisions
	return h.Sum64()
}
