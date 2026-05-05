package access

type ResourcePerms struct {
	r    *Registry
	slug ResourceSlug
}

func (r *Registry) R(slug ResourceSlug) ResourcePerms {
	return ResourcePerms{
		r:    r,
		slug: slug,
	}
}

func (rp ResourcePerms) Do(act ActionCode) Permission {
	p := Permission{
		Resource: rp.slug,
		Action:   act,
	}

	if !rp.r.IsValid(p) {
		panic("invalid permission: " + p.Key())
	}

	return p
}

func (rp ResourcePerms) Read() Permission {
	return rp.Do(Read)
}

func (rp ResourcePerms) Write() Permission {
	return rp.Do(Write)
}

func (rp ResourcePerms) Delete() Permission {
	return rp.Do(Delete)
}

func (rp ResourcePerms) All() Permission {
	return rp.Do(All)
}
