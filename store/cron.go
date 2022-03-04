package store

import (
	"context"
	"fmt"
	"time"
)

// CronJob is a job that we want to run regularly.
type CronJob struct {
	ID       int64
	Name     string
	Schedule string
	Enabled  bool
	NextRun  *time.Time
}

// LoadCronJobs returns all cron jobs in the database.
func LoadCrobJobs(ctx context.Context) ([]*CronJob, error) {
	sql := "SELECT id, job_name, schedule, enabled, next_run FROM cron ORDER BY id ASC"
	rows, err := conn.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("Error fetching jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*CronJob
	for rows.Next() {
		job := CronJob{}
		if err := rows.Scan(&job.ID, &job.Name, &job.Schedule, &job.Enabled, &job.NextRun); err != nil {
			return nil, fmt.Errorf("Error scanning row: %w", err)
		}
		jobs = append(jobs, &job)
	}
	return jobs, nil
}

// SaveCronJob saves the given cron job to the database.
func SaveCronJob(ctx context.Context, job *CronJob) error {
	if job.ID == 0 {
		sql := "INSERT INTO cron (job_name, schedule, enabled, next_run) VALUES ($1, $2, $3, $4)"
		_, err := conn.Exec(ctx, sql, job.Name, job.Schedule, job.Enabled, job.NextRun)
		return err
	} else {
		sql := "UPDATE cron SET job_name=$1, schedule=$2, enabled=$3, next_run=$4 WHERE id=$5"
		_, err := conn.Exec(ctx, sql, job.Name, job.Schedule, job.Enabled, job.NextRun, job.ID)
		return err
	}
}
