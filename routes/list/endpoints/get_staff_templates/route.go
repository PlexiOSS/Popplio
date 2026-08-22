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
	"strings"

	"github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

var (
	templateTypesColsArr = db.GetCols(types.StaffTemplateType{})
	templateTypesCols    = strings.Join(templateTypesColsArr, ",")

	templateColsArr = db.GetCols(types.StaffTemplate{})
	templateCols    = strings.Join(templateColsArr, ",")
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

	query := "SELECT " + templateCols + " FROM staff_templates"
	args := []any{}

	if entityType != "" {
		query += " WHERE entity_type = $1"
		args = append(args, entityType)
	}

	query += " ORDER BY created_at DESC"

	rows, err := state.Pool.Query(d.Context, query, args...)

	if err != nil {
		return resp.Err("Failed to fetch staff templates list [db fetch]", err)
	}

	defer rows.Close()

	templates, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.StaffTemplate])

	if err != nil {
		return resp.Err("Failed to fetch staff templates list [db fetch]", err)
	}

	typeRows, err := state.Pool.Query(d.Context, "SELECT "+templateTypesCols+" FROM staff_template_types ORDER BY created_at DESC")

	if err != nil {
		return resp.Err("Failed to fetch staff templates types list [db fetch]", err)
	}

	defer rows.Close()

	templatesTypes, err := pgx.CollectRows(typeRows, pgx.RowToStructByName[types.StaffTemplateType])

	if err != nil {
		return resp.Err("Failed to fetch staff templates type list [db fetch]", err)
	}

	return uapi.HttpResponse{
		Json: types.StaffTemplateList{
			Templates:     templates,
			TemplateTypes: templatesTypes,
		},
	}
}
