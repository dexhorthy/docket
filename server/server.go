package server

import (
	"encoding/json"
	"github.com/codegangsta/martini"
	"github.com/codegangsta/martini-contrib/binding"
	"github.com/codegangsta/martini-contrib/render"
	"github.com/fsouza/go-dockerclient"
	"github.com/horthy/docket/allocations"
	"log"
	"time"
	"github.com/horthy/docket/run"
)

func Start() {

	// TODO: env vars to configure storage backend, for now default to InMemory
	store := allocations.InMemory()

	m := martini.Classic()
	m.Use(render.Renderer())
	m.Use(func(c martini.Context) {
		c.MapTo(store, (*allocations.AllocationStore)(nil))
	})

	m.Get("/", handleGet)
	m.Get("/:name", handleGetAllocation)
	m.Delete("/:name", handleDeleteAllocation)
	m.Post("/", binding.Bind(allocations.AllocationSpecification{}), handlePost)

	client, err := docker.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	// TODO
	// using ticker feels kinda janky -- even if we continue to maintain our own collection
	// of Allocations, we can still use a cron library to manage scheduling our checks
	ticker := time.NewTicker(1 * time.Minute)
    runner := run.NewFsouza(client, store)
	go func() {
		for range ticker.C {
			RunAnyScheduledContainers(runner, store)
		}
	}()

	// TODO/nice to have: watch docker event stream, add exit codes to Allocation Logs

	m.Run()
}

func handlePost(
	allocation allocations.AllocationSpecification,
	allocationStore allocations.AllocationStore,
	r render.Render,
) {

	allocation.ProvisionDefaults()
	pretty, err := json.MarshalIndent(allocation, "", "    ")
	log.Printf("Received new allocation %v", string(pretty))

	created, err := allocationStore.CreateOrUpdate(&allocation)
	if err != nil {
		log.Printf("Failed to store allocation %v, error was %v", pretty, err)
		r.JSON(500, err)
	} else {
		log.Printf("Stored allocation %v", allocation.Name)
		r.JSON(200, map[string]bool{"created": created})
	}
}

func handleGet(allocationStore allocations.AllocationStore, r render.Render) {
	list, err := allocationStore.List()
	if err != nil {
		r.JSON(500, err)
	} else {
		r.JSON(200, list)
	}
}

func handleGetAllocation(allocationStore allocations.AllocationStore, r render.Render, params martini.Params) {
	allocation, err := allocationStore.Get(params["name"])
	if err != nil {
		r.JSON(500, err)
	} else {
		r.JSON(200, allocation)
	}
}

func handleDeleteAllocation(allocationStore allocations.AllocationStore, r render.Render, params martini.Params) {

	err := allocationStore.Delete(params["name"])
	if err != nil {
		r.JSON(500, err)
	} else {
		r.JSON(200, map[string]bool{"deleted": true})
	}
}

// Check the list of allocations, and create+start any
// containers that are scheduled at the time the method is called
func RunAnyScheduledContainers(runner run.AllocationRunner, allocationStore allocations.AllocationStore) {
	allAllocations, err := allocationStore.List()

	if err != nil {
		log.Printf("Couldn't get list of allocations, error was %v", err)
		return
	}

	for _, alloc := range allAllocations {
		if alloc.ShouldRunAt(time.Now()) {
			runner.RunAllocation(alloc)
		}
	}
}
