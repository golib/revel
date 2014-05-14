package jobs

import (
	"fmt"
	"github.com/golib/revel"
	"github.com/golib/revel/modules/jobs/app/jobs"
	"github.com/golib/revel/testdata/testapp/app/controllers"
	"github.com/golib/revel/testdata/testapp/app/models"
)

// Periodically count the bookings in the database.
type BookingCounter struct{}

func (c BookingCounter) Run() {
	bookings, err := controllers.Dbm.Select(models.Booking{},
		`select * from Booking`)
	if err != nil {
		panic(err)
	}
	fmt.Printf("There are %d bookings.\n", len(bookings))
}

func init() {
	revel.OnAppStart(func() {
		jobs.Schedule("@every 1m", BookingCounter{})
	})
}
