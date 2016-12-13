package allocations

import (
	"fmt"
	"github.com/codegangsta/martini-contrib/binding"
	"github.com/fsouza/go-dockerclient"
	"github.com/gorhill/cronexpr"
	"log"
	"net/http"
	"time"
)

// The request object sent to the server to define how and when a Container should be run
type AllocationSpecification struct {
	Name      string                        `json:"Name" binding:"required"`
	Cron      string                        `json:"Cron" binding:"required"`
	Container docker.CreateContainerOptions `json:"Container" binding:"required"`
}

// The internal structure used to track and configure scheduled containers
type Allocation struct {
	Name      string                        `json:"Name" `
	Logs      []interface{}                 `json:"Logs"`
	Cron      string                        `json:"Cron"`
	CronExpr  *cronexpr.Expression          `json:"-"`
	Container docker.CreateContainerOptions `json:"Container"`
}

type Allocations []*Allocation

func (allocation AllocationSpecification) Validate(errors *binding.Errors, req *http.Request) {
	_, err := cronexpr.Parse(allocation.Cron)

	if err != nil {
		errors.Fields["Cron"] = fmt.Sprintf("%v", err)
	}

	// Having issues getting AutoRemove to work. Coming soon
	if allocation.Container.Name != "" {
		errors.Fields["Container.Name"] = "Creating an allocation with a named container is not supported"
	}

	if allocation.Container.Config == nil {
		errors.Fields["Container.Config"] = "Config is required"
		return
	}

	// Having issues getting AutoRemove to work. Coming soon
	if allocation.Container.Config.Image == "" {
		errors.Fields["Container.Config.Image"] = "Image is required"
	}
}

func (allocation *Allocation) Log(events ...interface{}) {
	allocation.Logs = append(allocation.Logs, fmt.Sprintf("%v, %v", time.Now(), events))
}

// Abstraction on top of storing and querying
// The collection of allocations. Right now
// we'll back this with a slice, but may want to move
// to gkvlite or etcd or redis or whatever
// These are allowed to return error
// because other implementations may include IO calls
type AllocationSource interface {
	// Get a list of all allocations
	List() (Allocations, error)

	// Get the allocation by name.
	// will return an error if it can't be found
	Get(name string) (*Allocation, error)

	// Delete an allocation by name
	// will return an error if it can't be found
	Delete(name string) error

	// If an allocation exists with the given name,
	// update the values of that allocation.
	// Otherwise create a new one. Returns whether a
	// new allocation was created.
	CreateOrUpdate(allocation *AllocationSpecification) (bool, error)
}

func (allocation *Allocation) ShouldRunAt(atTime time.Time) bool {
	nextExecution := allocation.CronExpr.Next(atTime)
	log.Printf("Allocation %v would next run at %v", allocation.Name, nextExecution)
	oneMinute, _ := time.ParseDuration("1m")
	return nextExecution.Before(atTime.Add(oneMinute))
}

func (newAllocation *AllocationSpecification) ProvisionDefaults() {
	if newAllocation.Container.Config == nil {
		newAllocation.Container.Config = &docker.Config{}
	}

	if newAllocation.Container.HostConfig == nil {
		newAllocation.Container.HostConfig = &docker.HostConfig{}
	}

	if newAllocation.Container.NetworkingConfig == nil {
		newAllocation.Container.NetworkingConfig = &docker.NetworkingConfig{}
	}

}

func NewAllocation(newAllocation *AllocationSpecification) *Allocation {

	allocation := &Allocation{
		Name:      newAllocation.Name,
		Cron:      newAllocation.Cron,
		Container: newAllocation.Container,
		CronExpr:  cronexpr.MustParse(newAllocation.Cron), // we can MustParse because this was validated during request binding
	}

	return allocation
}
