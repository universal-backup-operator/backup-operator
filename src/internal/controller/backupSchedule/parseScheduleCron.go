/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package backupschedule

import (
	"errors"
	"fmt"
	"time"

	backupoperatoriov1 "backup-operator.io/api/v1"
	"github.com/robfig/cron/v3"
)

// Function gets next schedule time considering deadline seconds
func ParseScheduleCron(schedule *backupoperatoriov1.BackupSchedule, now time.Time) (lastMissed time.Time, next time.Time, err error) {
	// Set default timezone to UTC
	timezone, _ := time.Now().UTC().Zone()
	if schedule.Spec.TimeZone != nil {
		timezone = *schedule.Spec.TimeZone
	}
	// Get location from timezone
	var location *time.Location
	if location, err = time.LoadLocation(timezone); err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("unknown time zone %q: %s", timezone, err.Error())
	}
	// Create a parser
	var parser *cron.SpecSchedule
	var s cron.Schedule
	s, err = cron.ParseStandard(schedule.Spec.Schedule)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("unparseable schedule %q: %s", schedule.Spec.Schedule, err.Error())
	}
	parser, _ = s.(*cron.SpecSchedule)
	parser.Location = location
	// If schedule has been annotated with trigger annotation...
	// ...exit as it is a time to run
	if _, ok := schedule.Annotations[backupoperatoriov1.AnnotationTriggerSchedule]; ok {
		return now, parser.Next(now), nil
	}
	// for optimization purposes, cheat a bit and start from our last observed run time
	// we could reconstitute this here, but there's not much point, since we've
	// just updated it.
	var earliestTime time.Time
	if schedule.Status.LastScheduleTime != nil {
		earliestTime = schedule.Status.LastScheduleTime.Time
	} else {
		earliestTime = time.Now().Add(-time.Hour)
	}
	if schedule.Spec.StartingDeadlineSeconds != nil {
		// controller is not going to schedule anything below this point
		schedulingDeadline := now.Add(-time.Second * time.Duration(*schedule.Spec.StartingDeadlineSeconds))

		if schedulingDeadline.After(earliestTime) {
			earliestTime = schedulingDeadline
		}
	}
	if earliestTime.After(now) {
		// Then earliest time contains junk and we just schedule next time from now
		return time.Time{}, parser.Next(now), nil
	}

	starts := 0
	for t := parser.Next(earliestTime); !t.After(now); t = parser.Next(t) {
		lastMissed = t
		// An object might miss several starts. For example, if
		// controller gets wedged on Friday at 5:01pm when everyone has
		// gone home, and someone comes in on Tuesday AM and discovers
		// the problem and restarts the controller, then all the hourly
		// jobs, more than 80 of them for one hourly scheduledJob, should
		// all start running with no further intervention (if the scheduledJob
		// allows concurrency and late starts).
		//
		// However, if there is a bug somewhere, or incorrect clock
		// on controller's server or apiservers (for setting creationTimestamp)
		// then there could be so many missed start times (it could be off
		// by decades or more), that it would eat up all the CPU and memory
		// of this controller. In that case, we want to not try to list
		// all the missed start times.
		starts++
		if starts > (60 * 24 * 7) {
			// We can't get the most recent times so just return an empty slice
			return time.Time{}, time.Time{}, errors.New("too many missed start times (> 10080), set or decrease .spec.startingDeadlineSeconds or check clock skew")
		}
	}
	return lastMissed, parser.Next(now), nil
}
