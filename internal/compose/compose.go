// Package compose deterministically maps a docker-compose file into an
// engine.Spec so the web wizard's "Import a docker-compose file" option can
// pre-fill the same review form the prompt and manual options land in.
//
// The compose file is authoritative for STRUCTURE (which services exist, which
// are built from source vs. pulled as infrastructure images, their ports and
// replica counts) — so this mapping is a pure function with no LLM: what the
// user already wrote is honored exactly, offline and for free. Compose is
// silent on semantic ROLE, so the pattern (which drives every trait default) is
// only a light heuristic that the user corrects in the form. Anything we cannot
// map cleanly is surfaced as a human-readable note rather than silently dropped
// or invented — see ToSpec.
package compose

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/types"
	"gopkg.in/yaml.v2"

	"github.com/kriipke/chartpress/internal/engine"
)

// imageBaseToKey maps the final path component of a container image reference to
// a chartpress infrastructure-dependency key. A service pulling one of these
// images (and not built from source) becomes a dependency, never a subchart.
// Every value MUST be a key in engine's dependency registry — compose_test.go's
// TestImageMapTargetsAreKnownDependencies enforces that single source of truth.
var imageBaseToKey = map[string]string{
	"postgres":      "postgresql",
	"postgresql":    "postgresql",
	"postgis":       "postgresql",
	"mysql":         "mysql",
	"mariadb":       "mysql",
	"redis":         "redis",
	"valkey":        "valkey",
	"kafka":         "kafka",
	"cp-kafka":      "kafka",
	"rabbitmq":      "rabbitmq",
	"mongo":         "mongodb",
	"mongodb":       "mongodb",
	"elasticsearch": "elasticsearch",
}

var nonKebab = regexp.MustCompile(`[^a-z0-9]+`)

// ToSpec maps a docker-compose file to an engine.Spec plus notes describing
// anything that couldn't be mapped cleanly (renamed services, unrecognized
// images kept as subcharts, dropped extra ports, an all-infrastructure file).
// It returns an error only when the file can't be parsed or has no services;
// the drafted spec is intentionally NOT hard-validated (the user edits it in the
// form before /generate validates).
func ToSpec(data []byte) (engine.Spec, []string, error) {
	if strings.TrimSpace(string(data)) == "" {
		return engine.Spec{}, nil, fmt.Errorf("compose file is required")
	}
	proj, err := parse(data)
	if err != nil {
		return engine.Spec{}, nil, fmt.Errorf("invalid docker-compose file: %w", err)
	}
	if len(proj.Services) == 0 {
		return engine.Spec{}, nil, fmt.Errorf("no services found in the compose file")
	}

	notes := []string{}
	spec := engine.Spec{Rules: engine.DefaultRules()}

	// Umbrella name: the compose top-level `name:` if present. Read directly —
	// the loader's project name is overridden to a placeholder below so it never
	// depends on a working directory. The web form's App-name field overrides
	// this client-side (mirrors the prompt path), so an empty value is fine.
	var top struct {
		Name string `yaml:"name"`
	}
	_ = yaml.Unmarshal(data, &top)
	if name, changed := kebab(top.Name); name != "" {
		spec.UmbrellaChartName = name
		if changed {
			notes = append(notes, fmt.Sprintf("Umbrella name normalized to %q (chart names must be kebab-case).", name))
		}
	}

	// Deterministic order: compose services are a map.
	names := make([]string, 0, len(proj.Services))
	for k := range proj.Services {
		names = append(names, k)
	}
	sort.Strings(names)

	depSet := map[string]bool{}
	seen := map[string]bool{}
	sawEnv := false

	for _, key := range names {
		svc := proj.Services[key]
		if len(svc.Environment) > 0 {
			sawEnv = true
		}

		// Classification (Q4): built-from-source ⇒ subchart; a recognized infra
		// image ⇒ dependency; an unrecognized image ⇒ subchart, but flagged.
		if svc.Build == nil {
			if depKey, ok := imageToDependency(svc.Image); ok {
				if !depSet[depKey] {
					depSet[depKey] = true
					spec.Dependencies = append(spec.Dependencies, depKey)
				}
				continue
			}
			if svc.Image != "" {
				notes = append(notes, fmt.Sprintf(
					"Service %q uses image %q, which isn't a recognized dependency — imported as a subchart. Delete it if it's dev-only tooling.",
					key, svc.Image))
			}
		}

		name, changed := kebab(key)
		if name == "" {
			notes = append(notes, fmt.Sprintf("Service %q has no valid chart name after normalization — skipped.", key))
			continue
		}
		if changed {
			notes = append(notes, fmt.Sprintf("Renamed service %q → %q (chart names must be kebab-case).", key, name))
		}
		if seen[name] {
			deduped := name
			for i := 2; seen[deduped]; i++ {
				deduped = fmt.Sprintf("%s-%d", name, i)
			}
			notes = append(notes, fmt.Sprintf("Two services map to the chart name %q; kept the later one as %q.", name, deduped))
			name = deduped
		}
		seen[name] = true

		sc := engine.Subchart{Name: name}

		port, extra := firstPort(svc)
		if port > 0 {
			sc.Port = port
			// Pattern heuristic (Q5): serves a port ⇒ api-microservice.
			sc.Pattern = "api-microservice"
		} else {
			// Nothing to serve ⇒ worker (its default exposure is none).
			sc.Pattern = "worker"
		}
		if extra {
			notes = append(notes, fmt.Sprintf("Service %q publishes multiple ports; kept %d — set the rest in the form if needed.", key, port))
		}

		if svc.Deploy != nil {
			if strings.EqualFold(svc.Deploy.Mode, "global") {
				sc.Workload = "daemonset"
			}
			if svc.Deploy.Replicas != nil && *svc.Deploy.Replicas > 1 {
				sc.Scaling = "fixed"
			}
		}

		spec.Subcharts = append(spec.Subcharts, sc)
	}

	sort.Strings(spec.Dependencies)

	if len(spec.Subcharts) == 0 {
		notes = append(notes, "No application services detected (every service mapped to infrastructure). Add at least one subchart before generating.")
	}
	if sawEnv {
		notes = append(notes, "Environment variables (and any volumes, networks, or secrets) aren't imported — resolve them in the generated HANDOFF.md and each subchart's values.")
	}

	return engine.Normalize(spec), notes, nil
}

// parse loads the compose file purely for its structure: interpolation,
// validation, normalization, consistency checks and path resolution are all off
// so we never touch disk (no working dir, no .env), unresolved ${VARS} don't
// error, and arbitrary dev compose files load leniently. Services are still
// decoded (including the ports short syntax), which is all we read.
func parse(data []byte) (*types.Project, error) {
	return loader.LoadWithContext(context.Background(), types.ConfigDetails{
		WorkingDir:  "/",
		ConfigFiles: []types.ConfigFile{{Filename: "docker-compose.yaml", Content: data}},
		Environment: types.Mapping{},
	}, func(o *loader.Options) {
		o.SkipInterpolation = true
		o.SkipValidation = true
		o.SkipNormalization = true
		o.SkipConsistencyCheck = true
		o.ResolvePaths = false
		o.SetProjectName("compose", true)
	})
}

// firstPort returns the first container (target) port a service exposes, and
// whether it exposes more than one. Host-published ports are irrelevant to a
// chart — only the container port matters.
func firstPort(svc types.ServiceConfig) (port int, multiple bool) {
	var ports []int
	for _, p := range svc.Ports {
		if p.Target > 0 {
			ports = append(ports, int(p.Target))
		}
	}
	for _, e := range svc.Expose {
		if n := portNumber(e); n > 0 {
			ports = append(ports, n)
		}
	}
	if len(ports) == 0 {
		return 0, false
	}
	return ports[0], len(ports) > 1
}

// portNumber parses a bare exposed port like "3000" or "3000/tcp".
func portNumber(s string) int {
	if i := strings.IndexByte(s, '/'); i >= 0 {
		s = s[:i]
	}
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}

// imageToDependency reports whether an image reference is recognized infra and,
// if so, its chartpress dependency key. Registry host, tag and digest are
// stripped; the final path component is matched (so `postgres`, `redis:7`,
// `bitnami/postgresql:16`, and `docker.io/library/mongo` all resolve).
func imageToDependency(image string) (string, bool) {
	base := imageBase(image)
	if base == "" {
		return "", false
	}
	key, ok := imageBaseToKey[base]
	return key, ok
}

func imageBase(image string) string {
	s := strings.TrimSpace(image)
	if s == "" {
		return ""
	}
	if i := strings.IndexByte(s, '@'); i >= 0 { // strip digest
		s = s[:i]
	}
	// Strip the tag, but only within the last path segment — a registry host
	// may carry a ":port" that is not a tag (e.g. myreg:5000/team/api).
	slash := strings.LastIndexByte(s, '/')
	last := s[slash+1:]
	if c := strings.IndexByte(last, ':'); c >= 0 {
		last = last[:c]
	}
	return strings.ToLower(last)
}

// kebab normalizes a name to chartpress's kebab-case (lowercase, digits and
// hyphens), reporting whether it had to change anything. An input that reduces
// to empty returns "".
func kebab(s string) (string, bool) {
	orig := strings.TrimSpace(s)
	if orig == "" {
		return "", false
	}
	out := nonKebab.ReplaceAllString(strings.ToLower(orig), "-")
	out = strings.Trim(out, "-")
	return out, out != orig
}
