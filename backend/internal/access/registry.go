package access

import "sort"

type Resource struct {
	Slug           ResourceSlug
	Name           string
	Group          string
	Description    string
	AllowedActions map[ActionCode]struct{}
}

type Registry struct {
	resources map[ResourceSlug]Resource
}

func NewRegistry(resources ...Resource) *Registry {
	r := &Registry{
		resources: make(map[ResourceSlug]Resource),
	}

	for _, res := range resources {
		if _, exists := r.resources[res.Slug]; exists {
			panic("duplicate resource: " + string(res.Slug))
		}

		r.register(res)
	}

	return r
}

func (r *Registry) register(res Resource) {
	r.resources[res.Slug] = res
}

func (r *Registry) IsValid(p Permission) bool {
	res, ok := r.resources[p.Resource]
	if !ok {
		return false
	}

	if _, ok := res.AllowedActions[All]; ok {
		return true
	}

	_, ok = res.AllowedActions[p.Action]
	return ok
}

func (r *Registry) List() []Resource {
	out := make([]Resource, 0, len(r.resources))
	for _, res := range r.resources {
		out = append(out, res)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Group < out[j].Group
	})

	return out
}

func actions(list ...ActionCode) map[ActionCode]struct{} {
	m := make(map[ActionCode]struct{}, len(list))
	for _, a := range list {
		m[a] = struct{}{}
	}
	return m
}
