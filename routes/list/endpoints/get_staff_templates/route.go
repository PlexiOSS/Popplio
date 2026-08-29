// Package get_staff_templates implements GET /list/staff-templates — "Get
// Staff Templates".
//
// Returns the staff templates used for reviewing bots and servers,
// optionally filtered to one entity type via ?entity_type=.
package get_staff_templates

import (
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *doclib.Doc {
	return &doclib.Doc{
		Summary:     "Get Staff Templates",
		Description: "Returns all of the staff templates used for reviewing bots and servers. Filter to one entity type with ?entity_type=bot or ?entity_type=server; omit it to get both.",
		Resp:        types.StaffTemplateList{},
		Params: []doclib.Parameter{
			{
				Name:        "entity_type",
				Description: "Filter templates to just \"bot\" or \"server\". Omit for both.",
				Required:    false,
				In:          "query",
				Schema:      doclib.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	entityType := r.URL.Query().Get("entity_type")

	if entityType != "" && entityType != "bot" && entityType != "server" {
		return resp.BadRequest("entity_type must be \"bot\" or \"server\"")
	}

	q := db.New(state.Pool)

	entityTypeFilter := pgtype.Text{}
	if entityType != "" {
		entityTypeFilter = pgtype.Text{String: entityType, Valid: true}
	}

	rows, err := q.GetStaffTemplates(d.Context, entityTypeFilter)

	if err != nil {
		return resp.Err("Failed to fetch staff templates list [db fetch]", err)
	}

	templates := make([]types.StaffTemplate, len(rows))
	for i, row := range rows {
		templates[i] = types.StaffTemplate{
			ID:          row.ID,
			Name:        row.Name,
			Emoji:       row.Emoji,
			Tags:        row.Tags,
			Description: row.Description,
			Type:        row.Type,
			EntityType:  row.EntityType,
			CreatedAt:   row.CreatedAt.Time,
		}
	}

	typeRows, err := q.GetStaffTemplateTypes(d.Context)

	if err != nil {
		return resp.Err("Failed to fetch staff templates types list [db fetch]", err)
	}

	templatesTypes := make([]types.StaffTemplateType, len(typeRows))
	for i, row := range typeRows {
		templatesTypes[i] = types.StaffTemplateType{
			ID:    row.ID,
			Name:  row.Name,
			Icon:  row.Icon,
			Short: row.Short,
		}
	}

	return uapi.HttpResponse{
		Json: types.StaffTemplateList{
			Templates:     templates,
			TemplateTypes: templatesTypes,
		},
	}
}
