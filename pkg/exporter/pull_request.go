package exporter

import (
	"context"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/go-github/v35/github"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/promhippie/github_exporter/pkg/config"
	"strconv"
	"strings"
)

// PullRequestCollector represents a GitHub pull request on a repository
type PullRequestCollector struct {
	client   *github.Client
	logger   log.Logger
	failures *prometheus.CounterVec
	duration *prometheus.HistogramVec
	config   config.Target

	All *prometheus.Desc
}

// NewPullRequestCollector returns a new PullRequestCollector.
func NewPullRequestCollector(logger log.Logger, client *github.Client, failures *prometheus.CounterVec, duration *prometheus.HistogramVec, cfg config.Target) *PullRequestCollector {
	if failures != nil {
		failures.WithLabelValues("repo").Add(0)
	}
	return &PullRequestCollector{
		client:   client,
		logger:   log.With(logger, "collector", "repo"),
		failures: failures,
		duration: duration,
		config:   cfg,

		All: prometheus.NewDesc(
			"github_pull_requests_all",
			"All info about github pull requests",
			[]string{"number", "state", "title", "body", "created_at", "labels", "user", "merged", "comments", "commits", "additions", "deletions", "changed_files", "html_url",
				"review_comments", "assignee", "assignees", "author_association", "requested_reviewers"},
			nil,
		),
	}
}

// Metrics simply returns the list metric descriptors for generating a documentation.
func (c *PullRequestCollector) Metrics() []*prometheus.Desc {
	return []*prometheus.Desc{
		c.All,
	}
}

// Describe sends the super-set of all possible descriptors of metrics collected by this Collector.
func (c *PullRequestCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.All
}

// Collect is called by the Prometheus registry when collecting metrics.
func (c *PullRequestCollector) Collect(ch chan<- prometheus.Metric) {
	for _, name := range c.config.Repos.Value() {
		n := strings.Split(name, "/")

		if len(n) != 2 {
			level.Error(c.logger).Log(
				"msg", "Invalid repo name",
				"name", name,
			)

			c.failures.WithLabelValues("repo").Inc()
			continue
		}

		owner, repo := n[0], n[1]

		ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
		defer cancel()

		pullRequests, _, err := c.client.PullRequests.List(ctx, owner, repo, nil)

		if err != nil {
			level.Info(c.logger).Log(
				"msg", "Failed to fetch issues.",
				"name", name,
				"err", err,
			)

			c.failures.WithLabelValues("repo").Inc()
			continue
		}

		for i, record := range pullRequests {
			if record == nil {
				continue
			}

			number, user, assignee, state, title, label, merged := "", "", "", "", "", "", ""

			var labels []string
			for _, git_label := range record.Labels {
				if git_label != nil {
					labels = append(labels, *git_label.Name)
				}
			}

			if len(record.Labels) > 0 {
				label = string_or_empty(record.Labels[0].Name)
			}

			if record.Assignee != nil {
				assignee = string_or_empty(record.Assignee.Login)
			}

			if record.User != nil {
				user = string_or_empty(record.User.Login)
			}

			if record.Number != nil {
				number = string_int_or_empty(record.Number)
			}

			if record.State != nil {
				state = string_or_empty(record.State)
			}

			if record.Title != nil {
				title = string_or_empty(record.Title)
			}

			if record.Merged != nil {
				merged = string_bool_or_empty(record.Merged)
			}

			ch <- prometheus.MustNewConstMetric(
				c.All,
				prometheus.GaugeValue,
				float64(i),
				number,
				state,
				title,
				string_or_empty(record.Body),
				string_time_or_empty(record.CreatedAt),
				label,
				user,
				merged,
				string_int_or_empty(record.Comments),
				string_int_or_empty(record.Commits),
				string_int_or_empty(record.Additions),
				string_int_or_empty(record.Deletions),
				string_int_or_empty(record.ChangedFiles),
				string_or_empty(record.HTMLURL),
				string_int_or_empty(record.ReviewComments),
				assignee,
				"",
				string_or_empty(record.AuthorAssociation),
				"",
			)

		}
	}
}

func (c *PullRequestCollector) reposByOwnerAndName(ctx context.Context, owner, repo string) ([]*github.Repository, error) {
	if strings.Contains(repo, "*") {
		opts := &github.SearchOptions{
			ListOptions: github.ListOptions{
				PerPage: 50,
			},
		}

		var (
			repos []*github.Repository
		)

		for {
			result, resp, err := c.client.Search.Repositories(
				ctx,
				fmt.Sprintf("user:%s", owner),
				opts,
			)

			if err != nil {
				return nil, err
			}

			repos = append(
				repos,
				result.Repositories...,
			)

			if resp.NextPage == 0 {
				break
			}

			opts.Page = resp.NextPage
		}

		return repos, nil
	}

	res, _, err := c.client.Repositories.Get(ctx, owner, repo)

	if err != nil {
		return nil, err
	}

	return []*github.Repository{
		res,
	}, nil
}

func string_bool_or_empty(ptr *bool) string {
	if ptr == nil {
		return ""
	} else {
		return strconv.FormatBool(*ptr)
	}
}
