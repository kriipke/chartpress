// internal/engine/resolve.go
//
// Trait resolution: pattern defaults + explicit overrides → the concrete
// traits the engine generates from. The spec stores intent (pattern plus the
// keys the user wrote); resolution is recomputed on every generate, by every
// surface (server decode, CLI, operator reconcile) through this one function.
//
// Defaults apply DEPENDENTLY — they are functions of the already-resolved
// keys, not a flat merge. The governing invariant: a default can never cause
// a validation error; only keys the user actually wrote can fail. So
// `pattern: api-microservice` with `exposure: tcp` silently resolves the
// inherited ingress default to false, while an explicit `ingress: true` next
// to `exposure: tcp` errors.
package engine

import "fmt"

// Default ports per exposure. http/tcp share the conventional scaffold port;
// grpc uses its own convention.
const (
	defaultPortHTTP = 8080
	defaultPortGRPC = 50051
)

// ResolvedTraits are the concrete per-subchart generation inputs after
// pattern expansion and dependent defaulting. Port is 0 when exposure is
// "none"; Scaling is "fixed" for daemonsets (which never autoscale).
type ResolvedTraits struct {
	Pattern   string
	Workload  string
	Exposure  string
	Port      int
	Ingress   bool
	Scaling   string
	SharedEnv bool
}

// ResolveTraits resolves one subchart's traits against its pattern and the
// umbrella rules. Errors are returned only for keys explicitly present on the
// subchart (invalid enums, out-of-range port, contradictions).
func ResolveTraits(sc Subchart, rules Rules) (ResolvedTraits, error) {
	id := sc.Pattern
	if id == "" {
		id = DefaultPattern
	}
	p, ok := PatternByID(id)
	if !ok {
		return ResolvedTraits{}, fmt.Errorf("unknown pattern %q (see docs/subchart-patterns.md for the list)", sc.Pattern)
	}
	rt := ResolvedTraits{Pattern: p.ID}

	// workload: explicit key wins, else the pattern's default.
	rt.Workload = sc.Workload
	if rt.Workload == "" {
		rt.Workload = p.Defaults.Workload
	} else if !contains(AllowedWorkloads, rt.Workload) {
		return ResolvedTraits{}, fmt.Errorf("invalid workload %q (allowed: %v)", sc.Workload, AllowedWorkloads)
	}

	// exposure: explicit key wins, else the pattern's default.
	rt.Exposure = sc.Exposure
	if rt.Exposure == "" {
		rt.Exposure = p.Defaults.Exposure
	} else if !contains(AllowedExposures, rt.Exposure) {
		return ResolvedTraits{}, fmt.Errorf("invalid exposure %q (allowed: %v)", sc.Exposure, AllowedExposures)
	}

	// port: depends on the resolved exposure. An explicit port alongside
	// exposure "none" is ignored (the values it would feed don't exist),
	// matching "defaults never fail / explicit keys are validated".
	switch {
	case rt.Exposure == "none":
		rt.Port = 0
	case sc.Port != 0:
		if sc.Port < 1 || sc.Port > 65535 {
			return ResolvedTraits{}, fmt.Errorf("port %d out of range 1-65535", sc.Port)
		}
		rt.Port = sc.Port
	case rt.Exposure == "grpc":
		rt.Port = defaultPortGRPC
	default:
		rt.Port = defaultPortHTTP
	}

	// ingress: an explicit true must be routable; the pattern's default is
	// clamped to false whenever it wouldn't be.
	ingressable := (rt.Exposure == "http" || rt.Exposure == "grpc") && rules.Ingress != "none"
	if sc.Ingress != nil {
		if *sc.Ingress && !ingressable {
			if rules.Ingress == "none" {
				return ResolvedTraits{}, fmt.Errorf("ingress: true but rules.ingress is \"none\"")
			}
			return ResolvedTraits{}, fmt.Errorf("ingress: true requires exposure http or grpc (got %q)", rt.Exposure)
		}
		rt.Ingress = *sc.Ingress
	} else {
		rt.Ingress = p.Defaults.Ingress && ingressable
	}

	// scaling: daemonsets run one pod per node — they never autoscale and
	// "singleton" contradicts them. The pattern default is clamped to fixed;
	// explicit auto/singleton on a daemonset is a user-written contradiction.
	rt.Scaling = sc.Scaling
	if rt.Scaling == "" {
		rt.Scaling = p.Defaults.Scaling
		if rt.Workload == "daemonset" {
			rt.Scaling = "fixed"
		}
	} else {
		if !contains(AllowedScalings, rt.Scaling) {
			return ResolvedTraits{}, fmt.Errorf("invalid scaling %q (allowed: %v)", sc.Scaling, AllowedScalings)
		}
		if rt.Workload == "daemonset" && rt.Scaling != "fixed" {
			return ResolvedTraits{}, fmt.Errorf("scaling %q is invalid for a daemonset (one pod per node; it cannot autoscale or be a singleton)", rt.Scaling)
		}
	}

	// shared_env: plain choice; "on with the shared rules off" stays a silent
	// no-op by design, so no dependency here.
	if sc.SharedEnv != nil {
		rt.SharedEnv = *sc.SharedEnv
	} else {
		rt.SharedEnv = p.Defaults.SharedEnv
	}

	return rt, nil
}

// resolveAll resolves every subchart; callers run Validate first, so errors
// here are defensive.
func resolveAll(s Spec) ([]ResolvedTraits, error) {
	out := make([]ResolvedTraits, len(s.Subcharts))
	for i, sc := range s.Subcharts {
		rt, err := ResolveTraits(sc, s.Rules)
		if err != nil {
			return nil, fmt.Errorf("subchart %q: %w", sc.Name, err)
		}
		out[i] = rt
	}
	return out, nil
}

// Warnings returns the non-blocking spec-level lint findings. Warnings never
// mutate resolution — resolution stays a pure function of one subchart entry;
// these surface in the CLI output, the CR status, and the generated handoff.
func Warnings(s Spec) []string {
	resolved, err := resolveAll(s)
	if err != nil {
		return nil // invalid specs get errors, not warnings
	}
	var warns []string
	var gateways, public []string
	for i, sc := range s.Subcharts {
		if resolved[i].Pattern == "edge-gateway" {
			gateways = append(gateways, sc.Name)
		}
		if resolved[i].Ingress {
			public = append(public, sc.Name)
		}
		if resolved[i].Pattern == "admin-dashboard" && resolved[i].Ingress {
			warns = append(warns, fmt.Sprintf(
				"subchart %q (admin-dashboard) is reachable via ingress — protect this route (auth annotations / private ingress class) before going live", sc.Name))
		}
	}
	if len(gateways) > 0 && len(public) > 1 {
		warns = append(warns, fmt.Sprintf(
			"edge gateway %v is meant to be the platform's only public entry, but %d subcharts resolve ingress: true (%v) — intended?",
			gateways, len(public), public))
	}
	return warns
}
