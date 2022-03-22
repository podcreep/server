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
	rows, _ := pool.Query(ctx, sql)
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

// LoadCronJob returns a single cron job with the given ID from the database.
func LoadCrobJob(ctx context.Context, id int64) (*CronJob, error) {
	// TODO: just load the one? loading all and picking it is kind of inefficient, but if there's
	// only a handful, maybe it's not worth the effort to optimize this.
	cronJobs, err := LoadCrobJobs(ctx)
	if err != nil {
		return nil, err
	}

	for _, cronJob := range cronJobs {
		if cronJob.ID == id {
			return cronJob, nil
		}
	}
	return nil, fmt.Errorf("No such cron job: %d", id)
}

// Gets the time we need to wait until the next cron job. Maximum duration is 30 minutes.
func GetTimeToNextCronJob(ctx context.Context, now time.Time) time.Duration {
	sql := "SELECT MIN(next_run) FROM cron"
	row := pool.QueryRow(ctx, sql)
	var nextRunTime *time.Time
	err := row.Scan(&nextRunTime)
	if err != nil || nextRunTime == nil {
		return 30 * time.Minute
	}

	// Return the amount of time we have to wait, not less than a second.
	duration := nextRunTime.Sub(now)
	if duration < time.Second {
		duration = time.Second
	}
	return duration
}

// LoadPendingCronJobs all the cron jobs that are currently scheduled to run now.
func LoadPendingCronJobs(ctx context.Context, now time.Time) ([]*CronJob, error) {
	sql := "SELECT id, job_name, schedule, enabled, next_run FROM cron WHERE next_run < $1"
	rows, _ := pool.Query(ctx, sql, now)
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

// DeleteCronJob deletes the given cron job from the database.
func DeleteCronJob(ctx context.Context, id int64) error {
	sql := "DELETE FROM cron WHERE id = $1"
	_, err := pool.Exec(ctx, sql, id)
	return err
}

// SaveCronJob saves the given cron job to the database.
func SaveCronJob(ctx context.Context, job *CronJob) error {
	if job.ID == 0 {
		sql := "INSERT INTO cron (job_name, schedule, enabled, next_run) VALUES ($1, $2, $3, $4)"
		_, err := pool.Exec(ctx, sql, job.Name, job.Schedule, job.Enabled, job.NextRun)
		return err
	} else {
		sql := "UPDATE cron SET job_name=$1, schedule=$2, enabled=$3, next_run=$4 WHERE id=$5"
		_, err := pool.Exec(ctx, sql, job.Name, job.Schedule, job.Enabled, job.NextRun, job.ID)
		return err
	}
}
