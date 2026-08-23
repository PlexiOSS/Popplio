package config

import (
	"github.com/disgoorg/snowflake/v2"
)

type Config struct {
	DiscordAuth   DiscordAuth   `yaml:"discord_auth" validate:"required"`
	Sites         Sites         `yaml:"sites" validate:"required"`
	Channels      Channels      `yaml:"channels" validate:"required"`
	Roles         Roles         `yaml:"roles" validate:"required"`
	JAPI          JAPI          `yaml:"japi" validate:"required"`
	Notifications Notifications `yaml:"notifications" validate:"required"`
	Servers       Servers       `yaml:"servers" validate:"required"`
	Meta          Meta          `yaml:"meta" validate:"required"`
	Arcadia       Arcadia       `yaml:"arcadia" validate:"required"`
	Infernoplex   Infernoplex   `yaml:"infernoplex" validate:"required"`
	Captcha       Captcha       `yaml:"captcha" validate:"required"`
}

type DiscordAuth struct {
	Token            string   `yaml:"token" comment:"Discord bot token" validate:"required"`
	ClientID         string   `yaml:"client_id" default:"815553000470478850" comment:"Discord Client ID" validate:"required"`
	ClientSecret     string   `yaml:"client_secret" comment:"Discord Client Secret" validate:"required"`
	AllowedRedirects []string `yaml:"allowed_redirects" default:"http://localhost:3000/auth/sauron,http://localhost:8000/auth/sauron,https://reedwhisker.infinitybots.gg/auth/sauron,https://infinitybots.gg/auth/sauron,https://botlist.site/auth/sauron,https://infinitybots.xyz/auth/sauron" validate:"required"`
}

type Sites struct {
	Frontend    string `yaml:"frontend" default:"https://reedwhisker.infinitybots.gg" comment:"Frontend URL" validate:"required"`
	API         string `yaml:"api" default:"https://spider.infinitybots.gg" comment:"API URL" validate:"required"`
	Panel       string `yaml:"panel" default:"https://panel.infinitybots.gg" comment:"Panel URL" validate:"required"`
	Infernoplex string `yaml:"infernoplex" default:"https://infernoplex.infinitybots.gg" comment:"Infernoplex URL" validate:"required"`
	Instatus    string `yaml:"instatus" default:"https://infinity-bots.instatus.com" comment:"Instatus Status Page URL" validate:"required"`
}

type Roles struct {
	AwaitingStaff      snowflake.ID   `yaml:"awaiting_staff" default:"1029058929361174678" comment:"Awaiting Staff Role" validate:"required"`
	Apps               snowflake.ID   `yaml:"apps" default:"907729844605968454" comment:"Apps Role" validate:"required"`
	CertBot            snowflake.ID   `yaml:"cert_bot" default:"759468236999491594" comment:"Certified Bot Role" validate:"required"`
	PremiumRoles       []snowflake.ID `yaml:"premium_roles" default:"759468236999491594" comment:"Premium Roles" validate:"required"`
	BotDeveloper       snowflake.ID   `yaml:"bot_developer" default:"758756147313246209" comment:"Bot Developer Role" validate:"required"`
	CertifiedDeveloper snowflake.ID   `yaml:"certified_developer" default:"759468303344992266" comment:"Certified Developer Role" validate:"required"`
	BotRole            snowflake.ID   `yaml:"bot_role" default:"758652296459976715" comment:"Role given to bots joining the main server" validate:"required"`
	BugHunters         snowflake.ID   `yaml:"bug_hunters" default:"1042546603795427398" comment:"Bug Hunters Role" validate:"required"`
	TopReviewers       snowflake.ID   `yaml:"top_reviewers" default:"1239696066350420038" comment:"Top Reviewers Role" validate:"required"`
}

type Channels struct {
	BotLogs       snowflake.ID `yaml:"bot_logs" default:"762077915499593738" comment:"Bot Logs Channel" validate:"required"`
	ModLogs       snowflake.ID `yaml:"mod_logs" default:"911907978926493716" comment:"Mod Logs Channel" validate:"required"`
	Apps          snowflake.ID `yaml:"apps" default:"1034075132030894100" comment:"Apps Channel, should be a staff only channel" validate:"required"`
	VoteLogs      snowflake.ID `yaml:"vote_logs" default:"762077981811146752" comment:"Vote Logs Channel" validate:"required"`
	BanAppeals    snowflake.ID `yaml:"ban_appeals" default:"870950610692878337" comment:"Ban Appeals Channel" validate:"required"`
	AuthLogs      snowflake.ID `yaml:"auth_logs" default:"1075091440117498007" comment:"Auth Logs Channel" validate:"required"`
	TestingLounge snowflake.ID `yaml:"testing_lounge" default:"891611731699335209" comment:"Testing Lounge Channel, auto-unclaims are announced here" validate:"required"`
	System        snowflake.ID `yaml:"system" default:"762958420277067786" comment:"System Channel" validate:"required"`
	Uptime        snowflake.ID `yaml:"uptime" default:"1083108330442076292" comment:"Uptime Channel" validate:"required"`
	StaffLogs     snowflake.ID `yaml:"staff_logs" default:"1186195848497999912" comment:"Staff Logs Channel" validate:"required"`
}

type JAPI struct {
	Key string `yaml:"key" comment:"JAPI Key. Get it from https://japi.rest" validate:"required"`
}

type Notifications struct {
	VapidPublicKey  string `yaml:"vapid_public_key" default:"BIdUNSqYzqVjbdJhn8WK6SDYDVj85mKtctrEgj14KkjxIMerxQ9wywvvxECkuP8rL3s8zDgZSE9HSqW1wmhVPM8" comment:"Vapid Public Key (https://www.stephane-quantin.com/en/tools/generators/vapid-keys)" validate:"required"`
	VapidPrivateKey string `yaml:"vapid_private_key" comment:"Vapid Private Key (https://www.stephane-quantin.com/en/tools/generators/vapid-keys)" validate:"required"`
}

type Servers struct {
	Main    snowflake.ID `yaml:"main" default:"758641373074423808" comment:"Main Server ID" validate:"required"`
	Staff   snowflake.ID `yaml:"staff" default:"870950609291972618" comment:"Staff Server ID" validate:"required"`
	Testing snowflake.ID `yaml:"testing" default:"870952645811134475" comment:"Testing Server ID" validate:"required"`
}

type Meta struct {
	PostgresURL         string   `yaml:"postgres_url" default:"postgresql:///infinity" comment:"Postgres URL" validate:"required"`
	RedisURL            string   `yaml:"redis_url" default:"redis://localhost:6379" comment:"Redis URL" validate:"required"`
	Port                string   `yaml:"port" default:":8081" comment:"Port to run the server on" validate:"required"`
	VulgarList          []string `yaml:"vulgar_list" default:"fuck,suck,shit,kill" validate:"required"`
	UrgentMentions      string   `yaml:"urgent_mentions" default:"<@&1061643797315993701>" comment:"Urgent mentions" validate:"required"`
	PaypalClientID      string   `yaml:"paypal_client_id" default:"" comment:"Paypal Client ID" validate:"required"`
	PaypalSecret        string   `yaml:"paypal_secret" default:"" comment:"Paypal Secret" validate:"required"`
	StripePublicKey     string   `yaml:"stripe_public_key" default:"" comment:"Stripe Public Key" validate:"required"`
	StripeSecretKey     string   `yaml:"stripe_secret_key" default:"" comment:"Stripe Public Key" validate:"required"`
	UptimeRobotROAPIKey string   `yaml:"uptime_robot_ro_api_key" default:"" comment:"Uptime Robot Read-Only API Key" validate:"required"`
	PopplioProxy        string   `yaml:"popplio_proxy" default:"https://gateway.nodebyte.host/proxy/discord" comment:"Popplio Proxy URL" validate:"required"`
	OpenAIAPIKey        string   `yaml:"openai_api_key" default:"sk-proj-your-key-here" comment:"OpenAI API key, used to run submitted bot/server descriptions through the (free) moderation endpoint at submission time. Moderation is skipped entirely when this is unset" required:"false"`
}

type Arcadia struct {
	Token          string         `yaml:"token" comment:"Staff bot Discord token. This is a SEPARATE Discord application from Popplio's" validate:"required"`
	ServerPort     int            `yaml:"server_port" default:"3010" comment:"Port the staff panel API listens on" validate:"required"`
	PrefixCommands bool           `yaml:"prefix_commands" default:"false" comment:"Enable legacy prefix commands. Requires the privileged Message Content intent to be granted"`
	Prefix         string         `yaml:"prefix" default:"ibs!" comment:"Staff bot prefix, only used when prefix_commands is on" validate:"required"`
	Owners         []snowflake.ID `yaml:"owners" default:"510065483693817867" comment:"Bot owners, these users always hold the 'owner' staff position" validate:"required"`
	ProtectedBots  []snowflake.ID `yaml:"protected_bots" default:"1019662370278228028" comment:"Bots that cannot be force-removed with kick enabled" validate:"required"`
	Panel          Panel          `yaml:"panel" validate:"required"`
}

type Panel struct {
	ClientID           string   `yaml:"client_id" comment:"Discord client ID of the panel login app" validate:"required"`
	ClientSecret       string   `yaml:"client_secret" comment:"Discord client secret of the panel login app" validate:"required"`
	RedirectURL        []string `yaml:"redirect_url" comment:"Allow-list of panel login redirect URLs" validate:"required"`
	PanelScope         string   `yaml:"panel_scope" comment:"Static handshake value the frontend sends" validate:"required"`
	PanelResponseScope string   `yaml:"panel_response_scope" comment:"Static handshake value the frontend expects back" validate:"required"`
}

type Infernoplex struct {
	ClientID     string `yaml:"client_id" comment:"Infernoplex bot Discord client ID" validate:"required"`
	ClientSecret string `yaml:"client_secret" comment:"Infernoplex bot Discord client secret" validate:"required"`
	Prefix       string `yaml:"prefix" default:"inf!" comment:"Infernoplex bot prefix" validate:"required"`
	ServerPort   int    `yaml:"server_port" default:"3012" comment:"Port the Infernoplex bot API listens on" validate:"required"`
	Token        string `yaml:"token" comment:"Infernoplex bot Discord token" validate:"required"`
}

type Captcha struct {
	HMACSecret string `yaml:"hmac_secret" comment:"Secret used to sign and verify proof-of-work vote captcha challenges. Generate with e.g. openssl rand -hex 32 — rotating it invalidates all outstanding challenges" validate:"required"`
}

type Naevis struct {
	ClientID string `yaml:"client_id" comment:"Naevis bot Discord client ID" validate:"required"`
	Token    string `yaml:"token" comment:"Naevis bot Discord token" validate:"required"`
	Prefix   string `yaml:"prefix" default:"nae!" comment:"Naevis bot prefix" validate:"required"`
}
