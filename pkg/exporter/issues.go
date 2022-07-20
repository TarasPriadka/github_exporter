package exporter

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/go-github/v35/github"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/promhippie/github_exporter/pkg/config"
)

// RepoCollector collects metrics about the servers.
type IssueCollector struct {
	client   *github.Client
	logger   log.Logger
	failures *prometheus.CounterVec
	duration *prometheus.HistogramVec
	config   config.Target

	All *prometheus.Desc
}

// NewRepoCollector returns a new RepoCollector.
func NewIssueCollector(logger log.Logger, client *github.Client, failures *prometheus.CounterVec, duration *prometheus.HistogramVec, cfg config.Target) *IssueCollector {
	if failures != nil {
		failures.WithLabelValues("repo").Add(0)
	}
	return &IssueCollector{
		client:   client,
		logger:   log.With(logger, "collector", "repo"),
		failures: failures,
		duration: duration,
		config:   cfg,

		All: prometheus.NewDesc(
			"github_issues_all",
			"All info about github issues",
			[]string{"id", "status", "locked", "title", "body", "user", "author_association", "label", "num_comments", "created_at", "updated_at", "url", "html_url", "reactions_total", "reactions_plus_one", "reactions_minus_one", "assignee"},
			nil,
		),
	}
}

// Metrics simply returns the list metric descriptors for generating a documentation.
func (c *IssueCollector) Metrics() []*prometheus.Desc {
	return []*prometheus.Desc{
		c.All,
	}
}

// Describe sends the super-set of all possible descriptors of metrics collected by this Collector.
func (c *IssueCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.All
}

// Collect is called by the Prometheus registry when collecting metrics.
func (c *IssueCollector) Collect(ch chan<- prometheus.Metric) {
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

		issues, _, err := c.client.Issues.ListByRepo(ctx, owner, repo, nil)

		if err != nil {
			level.Info(c.logger).Log(
				"msg", "Failed to fetch issues.",
				"name", name,
				"err", err,
			)

			c.failures.WithLabelValues("repo").Inc()
			continue
		}

		for i, record := range issues {
			if record == nil {
				continue
			}
			id := string_int64_or_empty(record.ID)

			label, user, assignee, locked := "", "", "", "true"
			if record.Locked == nil {
				locked = ""
			} else if !*record.Locked {
				locked = "false"
			}
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

			ch <- prometheus.MustNewConstMetric(
				c.All,
				prometheus.GaugeValue,
				float64(i),
				id,
				string_or_empty(record.State),
				locked,
				string_or_empty(record.Title),
				string_or_empty(record.Body),
				user,
				string_or_empty(record.AuthorAssociation),
				label,
				string_int_or_empty(record.Comments),
				string_time_or_empty(record.CreatedAt),
				string_time_or_empty(record.UpdatedAt),
				string_or_empty(record.URL),
				string_or_empty(record.HTMLURL),
				string_int_or_empty(record.Reactions.TotalCount),
				string_int_or_empty(record.Reactions.PlusOne),
				string_int_or_empty(record.Reactions.MinusOne),
				assignee,
			)
		}
	}
}

func (c *IssueCollector) reposByOwnerAndName(ctx context.Context, owner, repo string) ([]*github.Repository, error) {
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

func string_or_empty(ptr *string) string {
	if ptr == nil {
		return ""
	} else {
		return *ptr
	}
}

func string_int_or_empty(ptr *int) string {
	if ptr == nil {
		return ""
	} else {
		return fmt.Sprint(*ptr)
	}
}

func string_int64_or_empty(ptr *int64) string {
	if ptr == nil {
		return ""
	} else {
		return fmt.Sprint(*ptr)
	}
}

func string_time_or_empty(ptr *time.Time) string {
	if ptr == nil {
		return ""
	} else {
		return ptr.String()
	}
}
