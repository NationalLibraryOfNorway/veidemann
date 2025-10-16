package importutil

import (
	"encoding/json"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config"
)

type SeedDesc struct {
	EntityId          string            `json:"entityId,omitempty" yaml:"entityId,omitempty"`
	EntityName        string            `json:"entityName,omitempty" yaml:"entityName,omitempty"`
	EntityDescription string            `json:"entityDescription,omitempty" yaml:"entityDescription,omitempty"`
	EntityLabel       []*configV1.Label `json:"entityLabel,omitempty" yaml:"entityLabel,omitempty"`
	Uri               string            `json:"uri,omitempty" yaml:"uri,omitempty"`
	SeedDescription   string            `json:"seedDescription,omitempty" yaml:"seedDescription,omitempty"`
	SeedLabel         []*configV1.Label `json:"seedLabel,omitempty" yaml:"seedLabel,omitempty"`

	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	CrawlJobRef []*configV1.ConfigRef
}

func (sd *SeedDesc) String() string {
	b, _ := json.Marshal(sd)
	return string(b)
}

func (sd *SeedDesc) ToEntity() *configV1.ConfigObject {
	return &configV1.ConfigObject{
		ApiVersion: "v1",
		Kind:       configV1.Kind_crawlEntity,
		Id:         sd.EntityId,
		Meta: &configV1.Meta{
			Name:        sd.EntityName,
			Description: sd.EntityDescription,
			Label:       sd.EntityLabel,
		},
	}
}

func (sd *SeedDesc) ToSeed() *configV1.ConfigObject {
	return &configV1.ConfigObject{
		ApiVersion: "v1",
		Kind:       configV1.Kind_seed,
		Meta: &configV1.Meta{
			Name:        sd.Uri,
			Description: sd.SeedDescription,
			Label:       sd.SeedLabel,
		},
		Spec: &configV1.ConfigObject_Seed{
			Seed: &configV1.Seed{
				EntityRef: &configV1.ConfigRef{
					Kind: configV1.Kind_crawlEntity,
					Id:   sd.EntityId,
				},
				JobRef: sd.CrawlJobRef,
			},
		},
	}
}
