package rethinkdb

import (
	"context"
	"fmt"
	"slices"

	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

type Query struct {
	*connection
}

// Verify verifies that database is initialized
func (qc *Query) Verify() error {
	if err := qc.checkDbExists(); err != nil {
		return err
	}
	if err := qc.checkTablesExists(); err != nil {
		return err
	}
	return nil
}

func (qc *Query) checkDbExists() error {
	cursor, err := r.DBList().Run(qc.session)
	if err != nil {
		return err
	}
	var dbList []string
	err = cursor.All(&dbList)
	if err != nil {
		return err
	}
	if !slices.Contains(dbList, "veidemann") {
		return fmt.Errorf("database 'veidemann' does not exist")
	}
	return nil
}

func (qc *Query) checkTablesExists() error {
	cursor, err := r.TableList().Run(qc.session)
	if err != nil {
		return err
	}
	var tableList []string
	err = cursor.All(&tableList)
	if err != nil {
		return err
	}
	if !slices.Contains(tableList, "config") || !slices.Contains(tableList, "job_executions") {
		return fmt.Errorf("tables 'config' and 'job_executions' does not exist")
	}
	return nil
}

func (qc *Query) WalkLatestJobExecutionForCrawlJobs(ctx context.Context, fn func(*frontierV1.JobExecutionStatus)) error {
	cursor, err := r.Table("config").Filter(map[string]any{"kind": "crawlJob"}).
		Map(func(job r.Term) any {
			return r.Table("job_executions").
				OrderBy(r.OrderByOpts{Index: r.Desc("jobId_startTime")}).
				Between([]r.Term{job.Field("id"), r.MinVal}, []r.Term{job.Field("id"), r.MaxVal}).
				Limit(1).
				Map(func(jes r.Term) any {
					return jes.Merge(map[string]any{
						"executionsState": jes.Field("executionsState").
							ConcatMap(func(state r.Term) any {
								return state.CoerceTo("array")
							}).
							CoerceTo("object"),
						"jobId": job.Field("meta").Field("name"),
					})
				}).
				Nth(0).
				Default(nil)
		}).
		Filter(func(jes r.Term) r.Term {
			return jes.Eq(nil).Not()
		}).
		Run(qc.session, r.RunOpts{
			Durability: "soft",
			ReadMode:   "outdated",
			Context:    ctx,
		})
	if err != nil {
		return err
	}

	jes := new(frontierV1.JobExecutionStatus)
	for cursor.Next(jes) {
		fn(jes)
	}
	return cursor.Err()
}
