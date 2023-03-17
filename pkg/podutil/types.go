package podutil

import "fmt"

type CreateContainerInput struct {
	Name              string            `json:"name,omitempty"`
	Image             string            `json:"image,omitempty"`
	PublishImagePorts bool              `json:"publish_image_ports,omitempty"`
	Remove            bool              `json:"remove,omitempty"`
	TimeoutSeconds    uint              `json:"timeout,omitempty"`
	Env               map[string]string `json:"env,omitempty"`
}

type CreateContainerOption func(*CreateContainerInput)

func WithEnv(name, value string) CreateContainerOption {
	return func(in *CreateContainerInput) {
		if in.Env == nil {
			in.Env = map[string]string{}
		}

		in.Env[name] = value
	}
}

type CreateContainerResult struct {
	ID       string   `json:"Id"`
	Warnings []string `json:"Warnings"`
}

const (
	ImagePullPolicyAlways  = `always`
	ImagePullPolicyMissing = `missing`
	ImagePullPolicyNewer   = `newer`
	ImagePullPolicyNever   = `never`
)

type PullImageInput struct {
	Reference string `json:"-" url:"reference"`
	Policy    string `json:"-" url:"policy"`
}

type InspectContainerResult struct {
	ID string `json:"Id"`

	NetworkSettings struct {
		Ports map[string][]struct {
			HostIP   string `json:"hostIp"`
			HostPort string `json:"hostPort"`
		} `json:"Ports"`
	} `json:"NetworkSettings"`
}

func (r InspectContainerResult) TCPHostPort(port int) string {
	key := fmt.Sprintf("%d/tcp", port)
	slice := r.NetworkSettings.Ports[key]
	if len(slice) == 0 {
		return ""
	}

	var (
		ip          = slice[0].HostIP
		exposedPort = slice[0].HostPort
	)

	if ip == "" {
		ip = "localhost"
	}

	return fmt.Sprintf("%s:%s", ip, exposedPort)
}
