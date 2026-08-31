package add_server_template

import (
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/routes/servertemplates/assets"
	"popplio/state"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-playground/validator/v10"
)

var compiledMessages = uapi.CompileValidationErrors(CreateServerTemplate{})

type CreateServerTemplate struct {
	Code  string   `json:"code" validate:"required,min=2,max=32,notblank" msg:"Template code is required"`
	Short string   `json:"short" validate:"required,min=10,max=150,noxss" msg:"Description must be between 10 and 150 characters"`
	Tags  []string `json:"tags" validate:"required,unique,min=1,max=5,dive,min=3,max=30,notblank,nonvulgar" msg:"There must be between 1 and 5 tags without duplicates" amsg:"Each tag must be between 3 and 30 characters and alphabetic"`
	NSFW  bool     `json:"nsfw"`
}

type CreateServerTemplateResponse struct {
	ID string `json:"id" description:"The created template's internal ID"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create Server Template",
		Description: "Submits a Discord server template listing. The template's name is pulled from Discord's own public template metadata at submission time -- Discord validates the code, not Popplio.",
		Req:         CreateServerTemplate{},
		Resp:        CreateServerTemplateResponse{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The user's ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var payload CreateServerTemplate

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	if err := state.Validator.Struct(payload); err != nil {
		errs := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errs)
	}

	q := db.New(state.Pool)

	exists, err := q.CountServerTemplateByCode(d.Context, payload.Code)

	if err != nil {
		return resp.Err("Error checking for an existing template with this code", err)
	}

	if exists {
		return resp.BadRequest("A template with that code has already been submitted")
	}

	meta, err := assets.FetchDiscordTemplate(d.Context, payload.Code)

	if err != nil {
		return resp.BadRequest(err.Error())
	}

	id, err := q.InsertServerTemplate(d.Context, db.InsertServerTemplateParams{
		Code:       payload.Code,
		Name:       meta.Name,
		Short:      payload.Short,
		Tags:       payload.Tags,
		Nsfw:       payload.NSFW,
		Owner:      d.Auth.ID,
		UsageCount: meta.UsageCount,
	})

	if err != nil {
		return resp.Err("Error inserting server template", err)
	}

	return uapi.HttpResponse{
		Status: http.StatusCreated,
		Json: CreateServerTemplateResponse{
			ID: id,
		},
	}
}
