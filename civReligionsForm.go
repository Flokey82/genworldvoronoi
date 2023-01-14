package genworldvoronoi

const (
	// Expansion modes.
	ReligionExpGlobal  = "global"
	ReligionExpState   = "state"
	ReligionExpCulture = "culture"

	// Religion groups.
	ReligionGroupFolk      = "Folk"
	ReligionGroupOrganized = "Organized"
	ReligionGroupCult      = "Cult"
	ReligionGroupHeresy    = "Heresy"

	// Religion forms.
	ReligionFormShamanism       = "Shamanism"
	ReligionFormAnimism         = "Animism"
	ReligionFormAncestorWorship = "Ancestor worship"
	ReligionFormPolytheism      = "Polytheism"
	ReligionFormDualism         = "Dualism"
	ReligionFormMonotheism      = "Monotheism"
	ReligionFormNontheism       = "Non-theism"
	ReligionFormCult            = "Cult"
	ReligionFormDarkCult        = "Dark Cult"
	ReligionFormHeresy          = "Heresy"
	// ReligionFormNature = "Nature"
)

var forms = map[string]map[string]int{
	ReligionGroupFolk: {
		ReligionFormShamanism:       2,
		ReligionFormAnimism:         2,
		ReligionFormAncestorWorship: 1,
		ReligionFormPolytheism:      2,
	},
	ReligionGroupOrganized: {
		ReligionFormPolytheism: 5,
		ReligionFormDualism:    1,
		ReligionFormMonotheism: 4,
		ReligionFormNontheism:  1,
	},
	ReligionFormCult: {
		ReligionFormCult:     1,
		ReligionFormDarkCult: 1,
	},
	ReligionFormHeresy: {
		ReligionFormHeresy: 1,
	},
}

var types = map[string]map[string]int{
	ReligionFormShamanism: {
		"Beliefs":   3,
		"Shamanism": 2,
		"Spirits":   1,
	},
	ReligionFormAnimism: {
		"Spirits": 1,
		"Beliefs": 1,
	},
	ReligionFormAncestorWorship: {
		"Beliefs":     1,
		"Forefathers": 2,
		"Ancestors":   2,
	},
	ReligionFormPolytheism: {
		"Deities":  3,
		"Faith":    1,
		"Gods":     1,
		"Pantheon": 1,
	},
	ReligionFormDualism: {
		"Religion": 3,
		"Faith":    1,
		"Cult":     1,
	},
	ReligionFormMonotheism: {
		"Religion": 1,
		"Church":   1,
	},
	ReligionFormNontheism: {
		"Beliefs": 3,
		"Spirits": 1,
	},
	ReligionFormCult: {
		"Cult":    4,
		"Sect":    4,
		"Arcanum": 1,
		"Coterie": 1,
		"Order":   1,
		"Worship": 1,
	},
	ReligionFormDarkCult: {
		"Cult":      2,
		"Sect":      2,
		"Blasphemy": 1,
		"Circle":    1,
		"Coven":     1,
		"Idols":     1,
		"Occultism": 1,
	},
	ReligionFormHeresy: {
		"Heresy":      3,
		"Sect":        2,
		"Apostates":   1,
		"Brotherhood": 1,
		"Circle":      1,
		"Dissent":     1,
		"Dissenters":  1,
		"Iconoclasm":  1,
		"Schism":      1,
		"Society":     1,
	},
}
