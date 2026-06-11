package models

type Access struct {
	Role   string   `json:"role"`
	Domain string   `json:"domain"`
	Perms  []string `json:"permissions"`
}

type GetPoliciesDTO struct {
	RealmId string
}
