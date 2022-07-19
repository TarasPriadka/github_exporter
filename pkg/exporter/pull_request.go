package exporter

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/go-github/v35/github"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/promhippie/github_exporter/pkg/config"
)

// RepoCollector collects metrics about the servers.
type PullRequestCollector struct {
	client   *github.Client
	logger   log.Logger
	failures *prometheus.CounterVec
	duration *prometheus.HistogramVec
	config   config.Target

	Status               *prometheus.Desc
	Locked               *prometheus.Desc
	Title                *prometheus.Desc
	Body                 *prometheus.Desc
	AuthorityAssociation *prometheus.Desc
	User                 *prometheus.Desc
	Labels               *prometheus.Desc
	Comments             *prometheus.Desc
	CreatedAt            *prometheus.Desc
	UpdatedAt            *prometheus.Desc
	URL                  *prometheus.Desc
	HTMLURL              *prometheus.Desc
	Reactions            *prometheus.Desc
	PlusOne              *prometheus.Desc
	MinusOne             *prometheus.Desc
	Assignees            *prometheus.Desc
}

// NewRepoCollector returns a new RepoCollector.
func NewPullRequestCollector(logger log.Logger, client *github.Client, failures *prometheus.CounterVec, duration *prometheus.HistogramVec, cfg config.Target) *IssueCollector {
	if failures != nil {
		failures.WithLabelValues("repo").Add(0)
	}
	labels := []string{"locked"}
	return &IssueCollector{
		client:   client,
		logger:   log.With(logger, "collector", "repo"),
		failures: failures,
		duration: duration,
		config:   cfg,

		Status: prometheus.NewDesc(
			"github_pull_requests_status",
			"Status of the issue",
			[]string{"id", "status"},
			nil,
		),
		Locked: prometheus.NewDesc(
			"github_issues_locked",
			"Wheather the issue is locked",
			[]string{"id", "locked"},
			nil,
		),
		Title: prometheus.NewDesc(
			"github_issues_title",
			"Wheather the issue is locked",
			[]string{"id", "title"},
			nil,
		),
		Body: prometheus.NewDesc(
			"github_issues_body",
			"Wheather the issue is locked",
			[]string{"id", "body"},
			nil,
		),
		AuthorityAssociation: prometheus.NewDesc(
			"github_issues_authority_association",
			"Wheather the issue is locked",
			labels,
			nil,
		),
		User: prometheus.NewDesc(
			"github_issues_user",
			"Wheather the issue is locked",
			labels,
			nil,
		),
		Labels: prometheus.NewDesc(
			"github_issues_labels",
			"Wheather the issue is locked",
			labels,
			nil,
		),
		Comments: prometheus.NewDesc(
			"github_issues_comments",
			"Wheather the issue is locked",
			labels,
			nil,
		),
		CreatedAt: prometheus.NewDesc(
			"github_issues_created_at",
			"Wheather the issue is locked",
			labels,
			nil,
		),
		UpdatedAt: prometheus.NewDesc(
			"github_issues_updated_at",
			"Wheather the issue is locked",
			labels,
			nil,
		),
		URL: prometheus.NewDesc(
			"github_issues_url",
			"Wheather the issue is locked",
			labels,
			nil,
		),
		HTMLURL: prometheus.NewDesc(
			"github_issues_html_url",
			"Wheather the issue is locked",
			labels,
			nil,
		),
		Reactions: prometheus.NewDesc(
			"github_issues_reactions",
			"Wheather the issue is locked",
			labels,
			nil,
		),
		PlusOne: prometheus.NewDesc(
			"github_issues_plus_one",
			"Wheather the issue is locked",
			labels,
			nil,
		),
		MinusOne: prometheus.NewDesc(
			"github_issues_minus_one",
			"Wheather the issue is locked",
			labels,
			nil,
		),
		Assignees: prometheus.NewDesc(
			"github_issues_assignees",
			"Wheather the issue is locked",
			labels,
			nil,
		),
	}
}

// Metrics simply returns the list metric descriptors for generating a documentation.
func (c *PullRequestCollector) Metrics() []*prometheus.Desc {
	return []*prometheus.Desc{
		c.Status,
		c.Locked,
		c.Title,
		c.Body,
		c.AuthorityAssociation,
		c.User,
		c.Labels,
		c.Comments,
		c.CreatedAt,
		c.UpdatedAt,
		c.URL,
		c.HTMLURL,
		c.Reactions,
		c.PlusOne,
		c.MinusOne,
		c.Assignees,
	}
}

// Describe sends the super-set of all possible descriptors of metrics collected by this Collector.
func (c *PullRequestCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.Status
	ch <- c.Locked
	ch <- c.Title
	ch <- c.Body
	ch <- c.AuthorityAssociation
	ch <- c.User
	ch <- c.Labels
	ch <- c.Comments
	ch <- c.CreatedAt
	ch <- c.UpdatedAt
	ch <- c.URL
	ch <- c.HTMLURL
	ch <- c.Reactions
	ch <- c.PlusOne
	ch <- c.MinusOne
	ch <- c.Assignees
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

		pull_requests, _, err := c.client.PullRequests.List(ctx, owner, repo, nil)

		if err != nil {
			level.Info(c.logger).Log(
				"msg", "Failed to fetch issues.",
				"name", name,
				"err", err,
			)

			c.failures.WithLabelValues("repo").Inc()
			continue
		}

		for i, record := range pull_requests {
			id := fmt.Sprint(*record.ID)
			if record.State != nil {
				labels := []string{id, *record.State}
				ch <- prometheus.MustNewConstMetric(
					c.Status,
					prometheus.GaugeValue,
					float64(i),
					labels...,
				)
			}

			if record.Locked != nil {
				locked := "true"
				if !*record.Locked {
					locked = "false"
				}

				ch <- prometheus.MustNewConstMetric(
					c.Locked,
					prometheus.GaugeValue,
					float64(i),
					id,
					locked,
				)
			}

			if record.Title != nil {
				ch <- prometheus.MustNewConstMetric(
					c.Title,
					prometheus.GaugeValue,
					float64(i),
					id,
					*record.Title,
				)
			}

			if record.Body != nil {
				ch <- prometheus.MustNewConstMetric(
					c.Body,
					prometheus.GaugeValue,
					float64(i),
					id,
					*record.Body,
				)
			}

			// if record.StargazersCount != nil {
			// 	ch <- prometheus.MustNewConstMetric(
			// 		c.Stargazers,
			// 		prometheus.GaugeValue,
			// 		float64(*record.StargazersCount),
			// 		labels...,
			// 	)
			// }

			// if record.SubscribersCount != nil {
			// 	ch <- prometheus.MustNewConstMetric(
			// 		c.Subscribers,
			// 		prometheus.GaugeValue,
			// 		float64(*record.SubscribersCount),
			// 		labels...,
			// 	)
			// }

			// if record.WatchersCount != nil {
			// 	ch <- prometheus.MustNewConstMetric(
			// 		c.Watchers,
			// 		prometheus.GaugeValue,
			// 		float64(*record.WatchersCount),
			// 		labels...,
			// 	)
			// }

			// if record.Size != nil {
			// 	ch <- prometheus.MustNewConstMetric(
			// 		c.Size,
			// 		prometheus.GaugeValue,
			// 		float64(*record.Size),
			// 		labels...,
			// 	)
			// }

			// if record.AllowRebaseMerge != nil {
			// 	ch <- prometheus.MustNewConstMetric(
			// 		c.AllowRebaseMerge,
			// 		prometheus.GaugeValue,
			// 		boolToFloat64(*record.AllowRebaseMerge),
			// 		labels...,
			// 	)
			// }

			// if record.AllowSquashMerge != nil {
			// 	ch <- prometheus.MustNewConstMetric(
			// 		c.AllowSquashMerge,
			// 		prometheus.GaugeValue,
			// 		boolToFloat64(*record.AllowSquashMerge),
			// 		labels...,
			// 	)
			// }

			// if record.AllowMergeCommit != nil {
			// 	ch <- prometheus.MustNewConstMetric(
			// 		c.AllowMergeCommit,
			// 		prometheus.GaugeValue,
			// 		boolToFloat64(*record.AllowMergeCommit),
			// 		labels...,
			// 	)
			// }

			// if record.Archived != nil {
			// 	ch <- prometheus.MustNewConstMetric(
			// 		c.Archived,
			// 		prometheus.GaugeValue,
			// 		boolToFloat64(*record.Archived),
			// 		labels...,
			// 	)
			// }

			// if record.Private != nil {
			// 	ch <- prometheus.MustNewConstMetric(
			// 		c.Private,
			// 		prometheus.GaugeValue,
			// 		boolToFloat64(*record.Private),
			// 		labels...,
			// 	)
			// }

			// if record.HasIssues != nil {
			// 	ch <- prometheus.MustNewConstMetric(
			// 		c.HasIssues,
			// 		prometheus.GaugeValue,
			// 		boolToFloat64(*record.HasIssues),
			// 		labels...,
			// 	)
			// }

			// if record.HasWiki != nil {
			// 	ch <- prometheus.MustNewConstMetric(
			// 		c.HasWiki,
			// 		prometheus.GaugeValue,
			// 		boolToFloat64(*record.HasWiki),
			// 		labels...,
			// 	)
			// }

			// if record.HasPages != nil {
			// 	ch <- prometheus.MustNewConstMetric(
			// 		c.HasPages,
			// 		prometheus.GaugeValue,
			// 		boolToFloat64(*record.HasPages),
			// 		labels...,
			// 	)
			// }

			// if record.HasProjects != nil {
			// 	ch <- prometheus.MustNewConstMetric(
			// 		c.HasProjects,
			// 		prometheus.GaugeValue,
			// 		boolToFloat64(*record.HasProjects),
			// 		labels...,
			// 	)
			// }

			// if record.HasDownloads != nil {
			// 	ch <- prometheus.MustNewConstMetric(
			// 		c.HasDownloads,
			// 		prometheus.GaugeValue,
			// 		boolToFloat64(*record.HasDownloads),
			// 		labels...,
			// 	)
			// }

			// ch <- prometheus.MustNewConstMetric(
			// 	c.Pushed,
			// 	prometheus.GaugeValue,
			// 	float64(record.PushedAt.Unix()),
			// 	labels...,
			// )

			// ch <- prometheus.MustNewConstMetric(
			// 	c.Created,
			// 	prometheus.GaugeValue,
			// 	float64(record.CreatedAt.Unix()),
			// 	labels...,
			// )

			// ch <- prometheus.MustNewConstMetric(
			// 	c.Updated,
			// 	prometheus.GaugeValue,
			// 	float64(record.UpdatedAt.Unix()),
			// 	labels...,
			// )
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
