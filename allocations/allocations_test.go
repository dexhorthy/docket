package allocations

import (
	"github.com/gorhill/cronexpr"
	"testing"
	"time"
)

func TestShouldRun(t *testing.T) {
	t.Log("Status\tExpect\tActual\tCron\t\tTime")
	atTime, _ := time.Parse(time.RFC3339, "2016-12-11T22:00:00+00:00")
	testAt(
		"1 * * * * *",
		atTime,
		false,
		t,
	)

	atTime, _ = time.Parse(time.RFC3339, "2016-12-11T22:00:30+00:00")
	testAt(
		"1 * * * * *",
		atTime,
		true,
		t,
	)

	atTime, _ = time.Parse(time.RFC3339, "2016-12-11T22:01:30+00:00")
	testAt(
		"1 * * * * *",
		atTime,
		false,
		t,
	)
}

func testAt(cron string, atTime time.Time, expected bool, t *testing.T) {
	a := &Allocation{
		Name:     "foo",
		Cron:     cron,
		CronExpr: cronexpr.MustParse(cron),
	}

	shouldRun := a.ShouldRunAt(atTime)
	if shouldRun != expected {
		t.Errorf("Failed\t%v\t%v\t%v\t%v", expected, shouldRun, cron, atTime)
	} else {
		t.Logf("Success\t%v\t%v\t%v\t%v", expected, shouldRun, cron, atTime)
	}

}
