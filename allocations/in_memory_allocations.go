package allocations

import (
	"fmt"
	"github.com/gorhill/cronexpr"
	"log"
	"sync"
	"time"
)

// InMemory creates a new allocationStore
// backed by a slice
func InMemory() *InMemoryAllocations {
	return &InMemoryAllocations{
		allocations: Allocations{},
		mutex: &sync.Mutex{},
	}
}

type InMemoryAllocations struct {
	allocations  Allocations
	// mutex to prevent client calls from modifying Allocations while they
	// are being inspected and run
	mutex        *sync.Mutex
	lockedReason string
}

func (a *InMemoryAllocations) lockFor(reason string) {
	a.mutex.Lock()
    a.lockedReason = reason
	log.Printf("Allocations locked for: %v", reason)
}

func (a *InMemoryAllocations) unlock() {
	reason := a.lockedReason
	a.lockedReason = ""
	a.mutex.Unlock()
	log.Printf("Allocations unlocked for: %v", reason)

}

func (a *InMemoryAllocations) List() (Allocations, error) {
    a.lockFor("list")
    defer a.unlock()
	return a.allocations, nil
}
func (a *InMemoryAllocations) Get(name string) (*Allocation, error) {
	a.lockFor("get")
	defer a.unlock()
	for _, allocation := range a.allocations {
		if allocation.Name == name {
			return allocation, nil
		}
	}
	return nil, fmt.Errorf("Allocation with name %v not found", name)
}

func (a *InMemoryAllocations) CreateOrUpdate(newAllocation *AllocationSpecification) (bool, error) {
	a.lockFor("create or update")
	defer a.unlock()

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
	a.lockFor("delete")
	defer a.unlock()
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


func (a *InMemoryAllocations) Log(allocation *Allocation, events ...interface{}) error {
    a.lockFor(fmt.Sprintf("logging to %v", allocation.Name))
	defer a.unlock()

	for _, a := range a.allocations {
		if a .Name == allocation.Name {
			a.Logs = append(allocation.Logs, fmt.Sprintf("%v, %v", time.Now(), events))
            return nil
		}
	}

	return fmt.Errorf("allocation %v not found", allocation.Name)
}

// Stolen from http://stackoverflow.com/questions/37334119/how-to-delete-an-element-from-array-in-golang
// swap the last element into index, return the a subslice up to len - 1
func (a *InMemoryAllocations) removeAt(index int) {
	a.allocations[index] = a.allocations[len(a.allocations)-1]
	a.allocations = a.allocations[:len(a.allocations)-1]
}
