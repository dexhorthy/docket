package allocations

import (
	"github.com/fsouza/go-dockerclient"
	"testing"
)

func TestInMemory(t *testing.T) {
	allocations := InMemory()

	allocations.CreateOrUpdate(&AllocationSpecification{
		Name: "foo",
		Cron: "* * * * * *",
		Container: docker.CreateContainerOptions{
			Config: &docker.Config{
				Image: "busybox:latest",
			},
		},
	})

	a, _ := allocations.Get("foo")
	if a.Cron != "* * * * * *" {
		t.Errorf("expected cron to be \"* * * * * * \" but was %v", a.Cron)
	}

	allocations.CreateOrUpdate(&AllocationSpecification{
		Name: "foo",
		Cron: "1 * * * * *",
		Container: docker.CreateContainerOptions{
			Config: &docker.Config{
				Image: "busybox:latest",
			},
		},
	})
	if a.Cron != "1 * * * * *" {
		t.Errorf("expected cron to be \"1 * * * * * \" but was %v", a.Cron)
	}

	list, _ := allocations.List()

	if len(list) != 1 {
		t.Errorf("expected list to return exactly 1 item but returned %v", len(list))
	}

	if list[0].Name != "foo" {
		t.Errorf("expected list contain 1 item with anme \"foo\" but name was %v", list[0].Name)
	}

	err := allocations.Delete("bar")

	if err == nil {
		t.Error("Expected err on deleting non existent allocation bar")
	}

	allocations.Delete("foo")

	list, _ = allocations.List()

	if len(list) != 0 {
		t.Errorf("expected list to return 0 items but returned %v", len(list))
	}

}
