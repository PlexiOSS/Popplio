// Copyright (C) 2026 NodeByte LTD

package types

import (
	"time"

	"popplio/validators/timex"

	"github.com/disgoorg/snowflake/v2"

	"github.com/PlexiOSS/Keel/dovewing/dovetypes"
	"github.com/PlexiOSS/Keel/uapi"
)

type QuestionCondition struct {
	QuestionID string   `json:"question_id" description:"An earlier question whose answer decides whether this one shows"`
	Values     []string `json:"values" description:"This question shows when that answer is (or, for multiple choice, includes) one of these"`
}

type Question struct {
	ID          string             `json:"id" validate:"required"`
	Question    string             `json:"question" validate:"required"`
	Paragraph   string             `json:"paragraph" description:"Help text shown under the question"`
	Placeholder string             `json:"placeholder"`
	Short       bool               `json:"short" description:"Used when type is empty: a one-line answer instead of a paragraph"`
	Type        string             `json:"type,omitempty" description:"short, long, choice, multi, yesno, number, url or rating. Empty means short or long, from the short flag"`
	Options     []string           `json:"options,omitempty" description:"The choices for choice and multi questions"`
	Optional    bool               `json:"optional,omitempty" description:"Whether the question can be left blank"`
	MinLength   int                `json:"min_length,omitempty" description:"Fewest characters for short and long answers. 0 uses the default"`
	MaxLength   int                `json:"max_length,omitempty" description:"Most characters for short and long answers. 0 uses the default"`
	ShowIf      *QuestionCondition `json:"show_if,omitempty" description:"Only show this question for certain answers to an earlier one"`
	Flag        []string           `json:"flag,omitempty" description:"Answers staff should see flagged when reviewing, such as No to having read the rules"`
}

type Position struct {
	ID                  string                                                                                 `json:"id" validate:"required"`
	Tags                []string                                                                               `json:"tags" validate:"required"`
	Info                string                                                                                 `json:"info" validate:"required"`
	Name                string                                                                                 `json:"name" validate:"required"`
	Questions           []Question                                                                             `json:"questions" validate:"gt=0,required"`
	Hidden              bool                                                                                   `json:"hidden"`
	Closed              bool                                                                                   `json:"closed"`
	RequiresMainServer  bool                                                                                   `json:"requires_main_server" description:"Whether applicants must be in the Omniplex Discord server"`
	Cooldown            timex.Duration                                                                         `json:"cooldown"`
	Channel             func() snowflake.ID                                                                    `json:"-"`
	ExtraLogic          func(d uapi.RouteData, p Position, answers map[string]string) error                    `json:"-"`
	PositionDescription func(d uapi.RouteData, p Position) string                                              `json:"-"`
	AllowedForBanned    bool                                                                                   `json:"-"`
	BannedOnly          bool                                                                                   `json:"-"`
	ReviewLogic         func(d uapi.RouteData, resp AppResponse, reason string, approved bool) (string, error) `json:"-"`
	ApprovalNote        string                                                                                 `json:"-"`
	HelpURL             string                                                                                 `json:"help_url,omitempty" description:"A page explaining the requirements for this position"`
	Summary             string                                                                                 `json:"summary,omitempty" description:"One line describing the position, for lists of positions"`
	HelpLabel           string                                                                                 `json:"help_label,omitempty" description:"The link text for help_url"`
	Behaviour           string                                                                                 `json:"-"`
}

type ManageAppResponse struct {
	StaffNote string `json:"staff_note,omitempty" description:"Something staff need to follow up on by hand, such as a Discord role that couldn't be given"`
}

type AppMeta struct {
	Positions []Position `json:"positions"`
	Stable    bool       `json:"stable"`
}

type AppResponse struct {
	AppID          string                  `db:"app_id" json:"app_id"`
	User           *dovetypes.PlatformUser `db:"-" json:"user,omitempty"`
	UserID         string                  `db:"user_id" json:"user_id"`
	Questions      []Question              `db:"questions" json:"questions"`
	Answers        map[string]string       `db:"answers" json:"answers"`
	State          string                  `db:"state" json:"state"`
	CreatedAt      time.Time               `db:"created_at" json:"created_at"`
	Position       string                  `db:"position" json:"position"`
	ReviewFeedback *string                 `db:"review_feedback" json:"review_feedback"`
	InMainServer   *bool                   `db:"-" json:"in_main_server,omitempty" description:"Whether the applicant is in the Omniplex Discord server. Only set on pending applications in the staff list, and left out when Discord couldn't be checked"`
}

type AppListResponse struct {
	Apps []AppResponse `json:"apps"`
}
