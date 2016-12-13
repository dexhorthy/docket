package allocations

import (
	"fmt"
	"github.com/gorhill/cronexpr"
)

// InMemory creates a new AllocationSource
// backed by a slice
func InMemory() *InMemoryAllocations {
	return &InMemoryAllocations{
		allocations: Allocations{},
	}
}

type InMemoryAllocations struct {
	allocations Allocations
}

func (a *InMemoryAllocations) List() (Allocations, error) {
	return a.allocations, nil
}
func (a *InMemoryAllocations) Get(name string) (*Allocation, error) {
	for _, allocation := range a.allocations {
		if allocation.Name == name {
			return allocation, nil
		}
	}
	return nil, fmt.Errorf("Allocation with name %v not found", name)
}

func (a *InMemoryAllocations) CreateOrUpdate(newAllocation *AllocationSpecification) (bool, error) {
	for _, allocation := range a.allocations {
		if allocation.Name == newAllocation.Name {
			// update
			allocation.Container = newAllocation.Container
			allocation.Cron = newAllocation.Cron
			allocation.CronExpr = cronexpr.MustParse(newAllocation.Cron) // we can MustParse because this was validated during request binding
			return false, nil
		}
	}

	// create a new one
	a.allocations = append(a.allocations, NewAllocation(newAllocation))
	return true, nil
}

func (a *InMemoryAllocations) Delete(name string) error {
	var index int
	found := false
	for i, allocation := range a.allocations {
		if allocation.Name == name {
			index = i
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("Allocation with name %v not found", name)
	}

	a.removeAt(index)
	return nil
}

// Stolen from http://stackoverflow.com/questions/37334119/how-to-delete-an-element-from-array-in-golang
// swap the last element into index, return the a subslice up to len - 1
func (a *InMemoryAllocations) removeAt(index int) {
	a.allocations[index] = a.allocations[len(a.allocations)-1]
	a.allocations = a.allocations[:len(a.allocations)-1]
}
