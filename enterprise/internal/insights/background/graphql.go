package background

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/sourcegraph/sourcegraph/internal/api"

	"golang.org/x/net/context/ctxhttp"

	"github.com/pkg/errors"
)

type graphQLQuery struct {
	Query     string      `json:"query"`
	Variables interface{} `json:"variables"`
}

const gqlSearchQuery = `query Search(
	$query: String!,
) {
	search(query: $query, ) {
		results {
			limitHit
			cloning { name }
			missing { name }
			timedout { name }
			matchCount
			alert {
				title
				description
			}
		}
	}
}`

type gqlSearchVars struct {
	Query string `json:"query"`
}

type gqlSearchResponse struct {
	Data struct {
		Search struct {
			Results struct {
				LimitHit   bool
				Cloning    []*api.Repo
				Missing    []*api.Repo
				Timedout   []*api.Repo
				MatchCount int
				Alert      struct {
					Title       string
					Description string
				}
			}
		}
	}
	Errors []interface{}
}

func search(ctx context.Context, query string) (*gqlSearchResponse, error) {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(graphQLQuery{
		Query:     gqlSearchQuery,
		Variables: gqlSearchVars{Query: query},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Encode")
	}

	url, err := gqlURL("Search")
	if err != nil {
		return nil, errors.Wrap(err, "constructing frontend URL")
	}

	resp, err := ctxhttp.Post(ctx, nil, url, "application/json", &buf)
	if err != nil {
		return nil, errors.Wrap(err, "Post")
	}
	defer resp.Body.Close()

	var res *gqlSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, errors.Wrap(err, "Decode")
	}
	if len(res.Errors) > 0 {
		return res, fmt.Errorf("graphql: errors: %v", res.Errors)
	}
	return res, nil
}

func gqlURL(queryName string) (string, error) {
	u, err := url.Parse(api.InternalClient.URL)
	if err != nil {
		return "", err
	}
	u.Path = "/.internal/graphql"
	u.RawQuery = queryName
	return u.String(), nil
}
