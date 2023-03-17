package podutil

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/sirupsen/logrus"
)

type HTTPDialContext func(ctx context.Context, network, addr string) (net.Conn, error)

func UserSocketPath() string {
	sock_dir := os.Getenv("XDG_RUNTIME_DIR")
	socket := sock_dir + "/podman/podman.sock"
	return socket
}

type Connection struct {
	client *http.Client
	base   string
}

func New(socketPath string) (*Connection, error) {
	transport := cleanhttp.DefaultTransport()
	transport.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
		dialer := net.Dialer{}
		return dialer.DialContext(ctx, "unix", socketPath)
	}

	conn := &Connection{
		base: "http://d/v4.0.0/libpod/",
		client: &http.Client{
			Transport: transport,
		},
	}

	err := conn.Ping()
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (c *Connection) request(opts ...RequestOption) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, c.base, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	for _, o := range opts {
		err := o(req)
		if err != nil {
			return nil, fmt.Errorf("apply option: %w", err)
		}
	}

	return c.client.Do(req)
}

func (c *Connection) Ping() error {
	resp, err := c.request(RequestPath("_ping"))
	if err != nil {
		return fmt.Errorf("_ping: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("_ping: %q", resp.Status)
	}

	return nil
}

func (c *Connection) CreateContainer(in CreateContainerInput, opts ...CreateContainerOption) (*CreateContainerResult, error) {
	for _, o := range opts {
		o(&in)
	}

	resp, err := c.request(
		RequestPath("containers/create"),
		RequestMethod(http.MethodPost),
		RequestJSONBody(in),
	)
	if err != nil {
		return nil, fmt.Errorf("create container: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("create container: %w", decodeError(resp.Body))
	}

	var result CreateContainerResult
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("decode JSON: %w", err)
	}

	return &result, nil
}

func (c *Connection) RemoveContainer(name string) error {
	resp, err := c.request(
		RequestPath("containers/%s", name),
		RequestMethod(http.MethodDelete),
	)
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("check status code: %d: %w", resp.StatusCode, decodeError(resp.Body))
	}

	return nil
}

func (c *Connection) StartContainer(name string) error {
	resp, err := c.request(
		RequestPath("containers/%s/start", name),
		RequestMethod(http.MethodPost),
	)
	if err != nil {
		return fmt.Errorf("start container: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusNotModified &&
		resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("check status code: %d: %w", resp.StatusCode, decodeError(resp.Body))
	}

	return nil
}

func (c *Connection) StopContainer(name string) error {
	resp, err := c.request(
		RequestPath("containers/%s/stop", name),
		RequestMethod(http.MethodPost),
	)
	if err != nil {
		return fmt.Errorf("stop container: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("check status code: %d: %w", resp.StatusCode, decodeError(resp.Body))
	}

	return nil
}

func (c *Connection) PullImage(in PullImageInput) error {
	resp, err := c.request(
		RequestPath("images/pull"),
		RequestMethod(http.MethodPost),
		RequestQueryStruct(in),
	)
	if err != nil {
		return fmt.Errorf("pull image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pull image: %w", decodeError(resp.Body))
	}

	// Podman is returning a json stream with the pull progress, but it takes
	// some extra steps to read it, while we do not have a use for this. But if
	// we do not consume the body, the request will terminated early and Podman
	// stops downloading the images. Therefore we just log it.
	w := logrus.WithField("image", in.Reference).WriterLevel(logrus.InfoLevel)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		return fmt.Errorf("reading body: %w", err)
	}

	return nil
}

func (c *Connection) InspectContainer(name string) (*InspectContainerResult, error) {
	resp, err := c.request(
		RequestPath("containers/%s/json", name),
	)
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("check status code: %w", decodeError(resp.Body))
	}

	var result InspectContainerResult
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("decode JSON: %w", err)
	}

	return &result, nil
}
