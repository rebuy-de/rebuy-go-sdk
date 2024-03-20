package podutil

import (
	"context"
	"fmt"

	"github.com/rebuy-de/rebuy-go-sdk/v8/pkg/cmdutil"
	"github.com/rebuy-de/rebuy-go-sdk/v8/pkg/logutil"
)

func StartDevcontainer(ctx context.Context, conn *Connection, name string, image string, opts ...CreateContainerOption) (*InspectContainerResult, error) {
	go func() {
		<-ctx.Done()

		logutil.Get(ctx).Infof("stopping %q container", name)
		err := conn.StopContainer(name)
		if err != nil {
			logutil.Get(ctx).Errorf("failed stop container %q: %s", name, err.Error())
		}
	}()

	// API is low level, so we must take care about image pulling ourselves.
	err := conn.PullImage(PullImageInput{
		Reference: image,
		Policy:    ImagePullPolicyMissing,
	})
	if err != nil {
		return nil, fmt.Errorf("pulling image %q: %w", image, err)
	}

	container, err := conn.InspectContainer(name)
	if err != nil {
		return nil, fmt.Errorf("inspecting existing container: %w", err)
	}

	if container != nil {
		logutil.Get(ctx).Warning("container already exists, hence reusing it")
	} else {
		_, err := conn.CreateContainer(CreateContainerInput{
			Name:              name,
			Image:             image,
			PublishImagePorts: true,
			Remove:            true, // always remove after stop

			// Make sure we do not have orphan pods when the go app gets killed
			// hard. Ideally we would make the timeout shorter and reset it
			// once in a while, but this does not seem possible with the podman
			// API.
			TimeoutSeconds: 3600 * 8,
		}, opts...)
		cmdutil.Must(err)

		container, err = conn.InspectContainer(name)
		if err != nil {
			return nil, fmt.Errorf("inspecting new container: %w", err)
		}
	}

	err = conn.StartContainer(name)
	if err != nil {
		return nil, fmt.Errorf("starting container: %w", err)
	}

	return container, nil
}
