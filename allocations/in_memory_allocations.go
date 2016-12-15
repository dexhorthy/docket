package allocations

import (
	"fmt"
	"github.com/gorhill/cronexpr"
	"log"
	"sync"
)

// InMemory creates a new AllocationSource
// backed by a slice
func InMemory() *InMemoryAllocations {
	return &InMemoryAllocations{
		allocations: Allocations{},
		mutex: &sync.Mutex{},
	}
}

type InMemoryAllocations struct {
	allocations Allocations
	// mutex to prevent client calls from modifying Allocations while they
	// are being inspected and run
    mutex sync.Mutex
}

func (a *InMemoryAllocations) List() (Allocations, error) {
	a.mutex.Lock()
	log.Print("Allocations locked to read all allocations")
	defer func() {
		a.mutex.Unlock()
		log.Print("Allocations unlocked after reading all allocations")
	}()
	return a.allocations, nil
}
func (a *InMemoryAllocations) Get(name string) (*Allocation, error) {
	a.mutex.Lock()
	log.Printf("Allocations locked to read allocation %v", name)
	defer func() {
		a.mutex.Unlock()
		log.Printf("Allocations unlocked after reading allocation %v", name)
	}()
	for _, allocation := range a.allocations {
		if allocation.Name == name {
			return allocation, nil
		}
	}
	return nil, fmt.Errorf("Allocation with name %v not found", name)
}

func (a *InMemoryAllocations) CreateOrUpdate(newAllocation *AllocationSpecification) (bool, error) {
	a.mutex.Lock()
	log.Printf("Allocations locked for new allocation %v", newAllocation.Name)
	defer func() {
		a.mutex.Unlock()
		log.Printf("Allocations unlocked for new allocation %v", newAllocation.Name)
	}()

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
	a.mutex.Lock()
	log.Printf("Allocations locked to delete allocation %v", name)
	defer func() {
		a.mutex.Unlock()
		log.Printf("Allocations unlocked to delete allocation %v", name)
	}()
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
