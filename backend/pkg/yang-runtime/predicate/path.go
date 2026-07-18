package predicate

import (
	"strings"
)

// Prefix returns a predicate that matches events with a path starting with the given prefix
func Prefix(prefix string) Predicate {
	return Funcs{
		CreateFunc: func(evt Event) bool {
			return strings.HasPrefix(evt.Path, prefix)
		},
		UpdateFunc: func(evt Event) bool {
			return strings.HasPrefix(evt.Path, prefix)
		},
		DeleteFunc: func(evt Event) bool {
			return strings.HasPrefix(evt.Path, prefix)
		},
		GenericFunc: func(evt Event) bool {
			return strings.HasPrefix(evt.Path, prefix)
		},
	}
}

// Exact returns a predicate that matches events with an exact path match
func Exact(path string) Predicate {
	return Funcs{
		CreateFunc: func(evt Event) bool {
			return evt.Path == path
		},
		UpdateFunc: func(evt Event) bool {
			return evt.Path == path
		},
		DeleteFunc: func(evt Event) bool {
			return evt.Path == path
		},
		GenericFunc: func(evt Event) bool {
			return evt.Path == path
		},
	}
}

// Contains returns a predicate that matches events with path contains the given substring
func Contains(substr string) Predicate {
	return Funcs{
		CreateFunc: func(evt Event) bool {
			return strings.Contains(evt.Path, substr)
		},
		UpdateFunc: func(evt Event) bool {
			return strings.Contains(evt.Path, substr)
		},
		DeleteFunc: func(evt Event) bool {
			return strings.Contains(evt.Path, substr)
		},
		GenericFunc: func(evt Event) bool {
			return strings.Contains(evt.Path, substr)
		},
	}
}

// ModulePrefix returns a predicate that matches events where the path starts with the module prefix
// For example, "huawei-ifm:" will match any path that starts with "/huawei-ifm:
func ModulePrefix(module string) Predicate {
	prefix := "/" + module + ":"
	return Prefix(prefix)
}

// DeviceIDEqual returns a predicate that matches events for the specific device ID
func DeviceIDEqual(deviceID string) Predicate {
	return Funcs{
		CreateFunc: func(evt Event) bool {
			return evt.DeviceID == deviceID
		},
		UpdateFunc: func(evt Event) bool {
			return evt.DeviceID == deviceID
		},
		DeleteFunc: func(evt Event) bool {
			return evt.DeviceID == deviceID
		},
		GenericFunc: func(evt Event) bool {
			return evt.DeviceID == deviceID
		},
	}
}

// ByType returns a predicate that matches specific event types
func ByType(types ...EventType) Predicate {
	typeSet := make(map[EventType]bool)
	for _, t := range types {
		typeSet[t] = true
	}
	return Funcs{
		CreateFunc: func(evt Event) bool {
			_, ok := typeSet[evt.Type]
			return ok
		},
		UpdateFunc: func(evt Event) bool {
			_, ok := typeSet[evt.Type]
			return ok
		},
		DeleteFunc: func(evt Event) bool {
			_, ok := typeSet[evt.Type]
			return ok
		},
		GenericFunc: func(evt Event) bool {
			_, ok := typeSet[evt.Type]
			return ok
		},
	}
}
