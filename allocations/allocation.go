package allocations

import (
	"fmt"
	"github.com/codegangsta/martini-contrib/binding"
	"github.com/fsouza/go-dockerclient"
	"github.com/gorhill/cronexpr"
	"golang.org/x/net/context"
	"log"
	"net/http"
	"time"
)

// The request object sent to the server to define how and when a Container should be run
type AllocationSpecification struct {
	Name      string                 `json:"Name" yaml:"Name" binding:"required"`
	Cron      string                 `json:"Cron"  yaml:"Cron" binding:"required"`
	Container CreateContainerOptions `json:"Container" yaml:"Container" binding:"required"`
}

// copy of docker.CreateContainerOptions,
// but with no name or context,
// and some custom yaml binding
type CreateContainerOptions struct {
	Config           *docker.Config           `qs:"-" json:"Config" yaml:"Config"`
	HostConfig       *docker.HostConfig       `qs:"-" json:"HostConfig" yaml:"HostConfig"`
	NetworkingConfig *docker.NetworkingConfig `qs:"-" json:"NetworkingConfig" yaml:"NetworkingConfig"`
}

func (opts CreateContainerOptions) ToOptions() docker.CreateContainerOptions {
	return docker.CreateContainerOptions{
		Config:           opts.Config,
		HostConfig:       opts.HostConfig,
		NetworkingConfig: opts.NetworkingConfig,
		Context:          context.TODO(),
	}
}

// The internal structure used to track and configure scheduled containers
type Allocation struct {
	Name      string                 `json:"Name" `
	Logs      []interface{}          `json:"Logs"`
	Cron      string                 `json:"Cron"`
	CronExpr  *cronexpr.Expression   `json:"-"`
	Container CreateContainerOptions `json:"Container"`
}

type Allocations []*Allocation

func (allocation AllocationSpecification) Validate(errors *binding.Errors, req *http.Request) {
	_, err := cronexpr.Parse(allocation.Cron)

	if err != nil {
		errors.Fields["Cron"] = fmt.Sprintf("%v", err)
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

// Abstraction on top of storing and querying
// The collection of allocations. Right now
// we'll back this with a slice, but may want to move
// to gkvlite or etcd or redis or whatever
// These are allowed to return error
// because other implementations may include IO calls
type AllocationStore interface {
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

	// Log an event regarding an exiting specification
	Log(allocation *Allocation, events ...interface{}) error
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
