package server

import (
	"encoding/json"
	"github.com/codegangsta/martini"
	"github.com/codegangsta/martini-contrib/binding"
	"github.com/codegangsta/martini-contrib/render"
	"github.com/fsouza/go-dockerclient"
	"github.com/horthy/docket/allocations"
	"log"
	"sync"
	"time"
)

func Start() {

	// TODO: env vars to configure storage backend, for now default to InMemory
	allocationSource := allocations.InMemory()

	m := martini.Classic()
	m.Use(render.Renderer())
	m.Use(func(c martini.Context) {
		c.MapTo(allocationSource, (*allocations.AllocationSource)(nil))
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
	go func() {
		for range ticker.C {
			RunAnyScheduledContainers(client, allocationSource)
		}
	}()

	// TODO/nice to have: watch docker event stream, add exit codes to Allocation Logs

	m.Run()
}

func handlePost(
	allocation allocations.AllocationSpecification,
	allocationSource allocations.AllocationSource,
	r render.Render,
) {

	allocation.ProvisionDefaults()
	pretty, err := json.MarshalIndent(allocation, "", "    ")
	log.Printf("Received new allocation %v", string(pretty))

	created, err := allocationSource.CreateOrUpdate(&allocation)
	if err != nil {
		log.Printf("Failed to store allocation %v, error was %v", pretty, err)
		r.JSON(500, err)
	} else {
		log.Printf("Stored allocation %v", allocation.Name)
		r.JSON(200, map[string]bool{"created": created})
	}
}

func handleGet(allocationSource allocations.AllocationSource, r render.Render) {
	list, err := allocationSource.List()
	if err != nil {
		r.JSON(500, err)
	} else {
		r.JSON(200, list)
	}
}

func handleGetAllocation(allocationSource allocations.AllocationSource, r render.Render, params martini.Params) {
	allocation, err := allocationSource.Get(params["name"])
	if err != nil {
		r.JSON(500, err)
	} else {
		r.JSON(200, allocation)
	}
}

func handleDeleteAllocation(allocationSource allocations.AllocationSource, r render.Render, params martini.Params) {

	err := allocationSource.Delete(params["name"])
	if err != nil {
		r.JSON(500, err)
	} else {
		r.JSON(200, map[string]bool{"deleted": true})
	}
}

// Check the list of allocations, and create+start any
// containers that are scheduled at the time the method is called
func RunAnyScheduledContainers(client *docker.Client, allocationSource allocations.AllocationSource) {
	allAllocations, err := allocationSource.List()

	if err != nil {
		log.Printf("Couldn't get list of allocations, error was %v", err)
		return
	}

	for _, alloc := range allAllocations {
		if alloc.ShouldRunAt(time.Now()) {
			runAllocation(alloc, client)
		}
	}
}

func runAllocation(alloc *allocations.Allocation, client *docker.Client) {
	log.Printf("Creating container for allocation %v with cron %v", alloc.Name, alloc.Cron)

	// pull image -- might want to this on allocation creation so we can bail
	// if the image doesn't exist, but leaving it here for now
	err := pullImage(alloc, client)
	if err != nil {
		return
	}

	container, err := createContainer(alloc, client)
	if err != nil {
		return
	}

	startContainer(alloc, client, container)
}

func pullImage(alloc *allocations.Allocation, client *docker.Client) error {
	repo, tag := docker.ParseRepositoryTag(alloc.Container.Config.Image)
	opts := docker.PullImageOptions{
		Repository: repo,
		Tag:        tag,
	}

	log.Printf("Pulling %v:%v for %v", repo, tag, alloc.Name)
	err := client.PullImage(opts, docker.AuthConfiguration{})
	if err != nil {
		log.Printf("Failed to pull image for %v, error was %v", alloc.Name, err)
		alloc.Log(err)
		return err
	}
	log.Printf("Pulled %v:%v for allocation %v", repo, tag, alloc.Name)
	alloc.Log("Pulled", repo, tag, alloc.Name)
	return nil
}

func createContainer(alloc *allocations.Allocation, client *docker.Client) (*docker.Container, error) {
	//create container
	container, err := client.CreateContainer(alloc.Container)
	if err != nil {
		log.Printf("Failed to create container for %v, error was %v", alloc.Name, err)
		alloc.Log(err)
		return nil, err
	}

	log.Printf("created: %v %v", container.Name, container.ID)
	alloc.Log("created:", container.Name, container.ID)
	return container, nil
}

func startContainer(alloc *allocations.Allocation, client *docker.Client, container *docker.Container) {
	// start
	err := client.StartContainer(container.ID, alloc.Container.HostConfig)
	if err != nil {
		log.Printf("Failed to start container for %v, error was %v", alloc.Name, err)
		alloc.Log(err)
		client.RemoveContainer(docker.RemoveContainerOptions{ID: container.ID})
		log.Printf("tried to remove container for %v, error was %v", alloc.Name, err)
		alloc.Log("removed container because", err)
		return
	}
	log.Printf("started: %v %v", container.Name, container.ID)
	alloc.Log("started:", container.Name, container.ID)

}
