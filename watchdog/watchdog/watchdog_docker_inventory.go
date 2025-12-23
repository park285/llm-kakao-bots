package watchdog

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/moby/moby/client"
)

// DockerContainerStatus is a snapshot of Docker container list (including managed status).
type DockerContainerStatus struct {
	Name    string   `json:"name"`
	Names   []string `json:"names,omitempty"`
	ID      string   `json:"id"`
	Image   string   `json:"image"`
	State   string   `json:"state"`
	Status  string   `json:"status"`
	Managed bool     `json:"managed"`
}

// ListDockerContainers 는 동작을 수행한다.
func (w *Watchdog) ListDockerContainers(ctx context.Context) ([]DockerContainerStatus, error) {
	listCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	result, err := w.cli.ContainerList(listCtx, client.ContainerListOptions{
		All: true,
	})
	if err != nil {
		return nil, fmt.Errorf("docker container list failed: %w", err)
	}

	cfg := w.GetConfig()
	managedSet := make(map[string]struct{}, len(cfg.Containers))
	for _, name := range cfg.Containers {
		if name == "" {
			continue
		}
		managedSet[name] = struct{}{}
	}

	out := make([]DockerContainerStatus, 0, len(result.Items))
	for i := range result.Items {
		item := &result.Items[i]

		names := make([]string, 0, len(item.Names))
		primaryName := ""
		for _, raw := range item.Names {
			n := CanonicalContainerName(raw)
			if n == "" {
				continue
			}
			names = append(names, n)
			if primaryName == "" {
				primaryName = n
			}
		}
		if primaryName == "" {
			primaryName = item.ID
		}

		managed := false
		for _, n := range names {
			if _, ok := managedSet[n]; ok {
				primaryName = n
				managed = true
				break
			}
		}

		out = append(out, DockerContainerStatus{
			Name:    primaryName,
			Names:   names,
			ID:      item.ID,
			Image:   item.Image,
			State:   string(item.State),
			Status:  trimStatusValue(item.Status),
			Managed: managed,
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})

	return out, nil
}