package store

import (
	"github.com/horthy/docket/allocations"
	"github.com/spf13/viper"
)

var StoreImpls = make(map[string]StoreFactory)

type StoreFactory interface {
	Create(*viper.Viper) (AllocationStore, error)
}

// Abstraction on top of storing and querying
// The collection of allocations. Right now
// we'll back this with a slice, but may want to move
// to gkvlite or etcd or redis or whatever
// These are allowed to return error
// because other implementations may include IO calls
type AllocationStore interface {
	// Get a list of all allocations
	List() (allocations.Allocations, error)

	// Get the allocation by name.
	// will return an error if it can't be found
	Get(name string) (*allocations.Allocation, error)

	// Delete an allocation by name
	// will return an error if it can't be found
	Delete(name string) error

	// If an allocation exists with the given name,
	// update the values of that allocation.
	// Otherwise create a new one. Returns whether a
	// new allocation was created.
	CreateOrUpdate(allocation *allocations.AllocationSpecification) (bool, error)

	// Log an event regarding an exiting specification
	Log(allocation *allocations.Allocation, events ...interface{}) error
}
