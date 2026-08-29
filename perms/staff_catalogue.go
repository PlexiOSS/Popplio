package perms

const (
	StaffAdministrator Perm = "administrator"

	StaffViewPanel     Perm = "view_panel"
	StaffViewAuditLogs Perm = "view_audit_logs"
	StaffViewSensitive Perm = "view_sensitive_data"
	StaffUseStagingKey Perm = "use_staging_keys"

	StaffReviewEntities      Perm = "review_entities"
	StaffCertifyEntities     Perm = "certify_entities"
	StaffTransferBots        Perm = "transfer_bots"
	StaffForceRemoveEntities Perm = "force_remove_entities"

	StaffManagePremium   Perm = "manage_premium"
	StaffFeatureEntities Perm = "feature_entities"
	StaffManageVotes     Perm = "manage_votes"
	StaffBanVoters       Perm = "ban_voters"
	StaffBanUsers        Perm = "ban_users"

	StaffViewApps    Perm = "view_apps"
	StaffManageApps  Perm = "manage_apps"
	StaffBanAppUsers Perm = "ban_app_users"

	StaffViewStaff            Perm = "view_staff"
	StaffManageStaffMembers   Perm = "manage_staff_members"
	StaffManageStaffRoles     Perm = "manage_staff_roles"
	StaffManageDisciplinaries Perm = "manage_disciplinaries"
	StaffViewOnboarding       Perm = "view_onboarding"

	StaffViewShop           Perm = "view_shop"
	StaffManageShop         Perm = "manage_shop"
	StaffManageBotWhitelist Perm = "manage_bot_whitelist"

	StaffManagePartners  Perm = "manage_partners"
	StaffManageBlog      Perm = "manage_blog"
	StaffManageChangelog Perm = "manage_changelog"

	StaffReviewReports Perm = "review_reports"

	StaffViewTickets   Perm = "view_tickets"
	StaffManageTickets Perm = "manage_tickets"

	StaffModerateGuild Perm = "moderate_guild"
	StaffWarnUsers     Perm = "warn_users"
	StaffPurgeMessages Perm = "purge_messages"
	StaffLockChannels  Perm = "lock_channels"
	StaffViewModCases  Perm = "view_mod_cases"

	StaffManageBadges    Perm = "manage_badges"
	StaffAssignBadges    Perm = "assign_badges"
	StaffManageTemplates Perm = "manage_templates"

	StaffViewCDN   Perm = "view_cdn"
	StaffManageCDN Perm = "manage_cdn"

	StaffMarkerDeveloper      Perm = "marker_developer"
	StaffMarkerLeadDeveloper  Perm = "marker_lead_developer"
	StaffMarkerHumanResources Perm = "marker_human_resources"
	StaffMarkerReviewer       Perm = "marker_reviewer"
	StaffMarkerServiceAccount Perm = "marker_service_account"
	StaffMarkerDisciplinary   Perm = "marker_disciplinary"
)

var Staff = NewCatalogue("staff", StaffAdministrator, []Definition{
	{
		ID:          StaffAdministrator,
		Name:        "Administrator",
		Description: "Full control over everything. This implies every other staff permission and should be held by as few people as possible.",
		Category:    "Administration",
		Dangerous:   true,
		Legacy:      []string{"global.*", "arcadia.*"},
	},
	{
		ID:          StaffViewPanel,
		Name:        "View Panel",
		Description: "Access the staff panel and see the data on it.",
		Category:    "Administration",
		Legacy:      []string{"global.view"},
	},
	{
		ID:          StaffViewAuditLogs,
		Name:        "View Audit Logs",
		Description: "See the log of every staff action taken through the panel and the staff bot.",
		Category:    "Administration",
		Legacy:      []string{"rpc_logs.view", "rpc_logs.*"},
	},
	{
		ID:          StaffViewSensitive,
		Name:        "View Sensitive Data",
		Description: "See data hidden from other staff, such as private contact details on an entity.",
		Category:    "Administration",
		Dangerous:   true,
		Legacy:      []string{"global.view_sensitive"},
	},
	{
		ID:          StaffUseStagingKey,
		Name:        "Use Staging Keys",
		Description: "Perform actions that use test payment keys on staging and development instances.",
		Category:    "Administration",
		Legacy:      []string{"popplio_staging.sensitive", "popplio_staging.*"},
	},

	{
		ID:          StaffReviewEntities,
		Name:        "Review Entities",
		Description: "Claim, unclaim, approve, deny and unverify bots and servers in the review queue.",
		Category:    "Entity Reviews",
		Legacy:      []string{"rpc.Claim", "rpc.Unclaim", "rpc.Approve", "rpc.Deny", "rpc.Unverify"},
	},
	{
		ID:          StaffCertifyEntities,
		Name:        "Certify Entities",
		Description: "Grant and remove certification on a bot or server.",
		Category:    "Entity Reviews",
		Legacy:      []string{"rpc.CertifyAdd", "rpc.CertifyRemove"},
	},
	{
		ID:          StaffTransferBots,
		Name:        "Transfer Bots",
		Description: "Move a bot to a different owner or team.",
		Category:    "Content Management",
		Legacy:      []string{"rpc.BotTransferOwnershipUser", "rpc.BotTransferOwnershipTeam"},
	},
	{
		ID:          StaffForceRemoveEntities,
		Name:        "Force Remove Entities",
		Description: "Delete a bot, server, or pack from the list outright. This cannot be undone.",
		Category:    "Content Management",
		Dangerous:   true,
		Legacy:      []string{"rpc.ForceRemove"},
	},

	{
		ID:          StaffManagePremium,
		Name:        "Manage Premium",
		Description: "Give and take premium status on an entity.",
		Category:    "Content Management",
		Dangerous:   true,
		Legacy:      []string{"rpc.PremiumAdd", "rpc.PremiumRemove"},
	},
	{
		ID:          StaffFeatureEntities,
		Name:        "Feature Entities",
		Description: "Feature or spotlight a bot or server on the home page for a given time period, or remove it early.",
		Category:    "Content Management",
	},
	{
		ID:          StaffManageVotes,
		Name:        "Manage Votes",
		Description: "Reset the votes of an entity, or of every entity at once.",
		Category:    "Content Management",
		Dangerous:   true,
		Legacy:      []string{"rpc.VoteReset", "rpc.VoteResetAll"},
	},
	{
		ID:          StaffBanVoters,
		Name:        "Ban Voters",
		Description: "Vote ban and unban a user.",
		Category:    "Content Management",
		Legacy:      []string{"rpc.VoteBanAdd", "rpc.VoteBanRemove"},
	},
	{
		ID:          StaffBanUsers,
		Name:        "Ban Users",
		Description: "Ban and unban a user's account from the platform entirely — blocks every authenticated action except submitting a ban appeal.",
		Category:    "Users & Votes",
		Dangerous:   true,
	},

	{
		ID:          StaffViewApps,
		Name:        "View Applications",
		Description: "Read staff and partner applications.",
		Category:    "Applications",
		Legacy:      []string{"apps.view"},
	},
	{
		ID:          StaffManageApps,
		Name:        "Manage Applications",
		Description: "Approve, deny and otherwise act on applications.",
		Category:    "Applications",
		Legacy:      []string{"apps.manage"},
	},
	{
		ID:          StaffBanAppUsers,
		Name:        "Ban Applicants",
		Description: "Bar a user from submitting further applications, and lift that ban.",
		Category:    "Applications",
		Legacy:      []string{"rpc.AppBanUser", "rpc.AppUnbanUser"},
	},

	{
		ID:          StaffViewStaff,
		Name:        "View Staff",
		Description: "See the staff list, the roles that exist and who holds them.",
		Category:    "Staff Management",
		Legacy:      []string{"staff_members.view", "staff_positions.view"},
	},
	{
		ID:          StaffManageStaffMembers,
		Name:        "Manage Staff Members",
		Description: "Edit a staff member's extra permissions and their sync settings.",
		Category:    "Staff Management",
		Dangerous:   true,
		Legacy:      []string{"staff_members.edit"},
	},
	{
		ID:          StaffManageStaffRoles,
		Name:        "Manage Staff Roles",
		Description: "Create, edit, reorder and delete staff roles and the permissions attached to them.",
		Category:    "Staff Management",
		Dangerous:   true,
		Legacy: []string{
			"staff_positions.create", "staff_positions.edit", "staff_positions.delete",
			"staff_positions.set_index", "staff_positions.swap_index",
		},
	},
	{
		ID:          StaffManageDisciplinaries,
		Name:        "Manage Disciplinaries",
		Description: "Create, edit and delete the disciplinary types that limit a staff member's permissions.",
		Category:    "Staff Management",
		Dangerous:   true,
		Legacy: []string{
			"staff_disciplinary_types.create", "staff_disciplinary_types.update",
			"staff_disciplinary_types.delete", "staff_disciplinary_types.*",
		},
	},
	{
		ID:          StaffViewOnboarding,
		Name:        "View Onboarding",
		Description: "Read the onboarding responses submitted by new staff.",
		Category:    "Staff Management",
		Legacy:      []string{"persepolis.view_onboarding_responses", "persepolis.*"},
	},

	{
		ID:          StaffViewShop,
		Name:        "View Shop",
		Description: "See shop items, benefits, coupons and vote credit tiers.",
		Category:    "Shop",
		Legacy:      []string{"shop.view", "shop_items.view", "shop_item_benefits.view", "shop_coupons.list"},
	},
	{
		ID:          StaffManageShop,
		Name:        "Manage Shop",
		Description: "Create, edit and delete shop items, benefits, coupons and vote credit tiers.",
		Category:    "Shop",
		Legacy: []string{
			"shop_items.create", "shop_items.update", "shop_items.delete",
			"shop_item_benefits.create", "shop_item_benefits.update", "shop_item_benefits.delete",
			"shop_coupons.create", "shop_coupons.update", "shop_coupons.delete",
			"vote_credit_tiers.create", "vote_credit_tiers.update", "vote_credit_tiers.delete",
		},
	},
	{
		ID:          StaffManageBotWhitelist,
		Name:        "Manage Bot Whitelist",
		Description: "Control which bots are whitelisted for shop purchases.",
		Category:    "Shop",
		Legacy: []string{
			"bot_whitelist.create", "bot_whitelist.update", "bot_whitelist.delete", "bot_whitelist.*",
		},
	},

	{
		ID:          StaffManagePartners,
		Name:        "Manage Partners",
		Description: "Create, edit and delete partners.",
		Category:    "Content Management",
		Legacy:      []string{"partners.create", "partners.update", "partners.delete", "partners.*"},
	},
	{
		ID:          StaffManageBlog,
		Name:        "Manage Blog",
		Description: "Create, edit and delete blog entries.",
		Category:    "Content Management",
		Legacy:      []string{"blog.create_entry", "blog.update_entry", "blog.delete_entry", "blog.*"},
	},
	{
		ID:          StaffManageChangelog,
		Name:        "Manage Changelog",
		Description: "Create, edit and delete changelog entries for Popplio, Omniplex, and Keel.",
		Category:    "Content Management",
	},

	{
		ID:          StaffReviewReports,
		Name:        "Review Reports",
		Description: "List, resolve and dismiss user-filed content reports (license violations, ToS violations, spam, etc) against bots, servers and packs.",
		Category:    "Content Management",
	},

	{
		ID:          StaffViewTickets,
		Name:        "View Tickets",
		Description: "Read support tickets opened by users.",
		Category:    "Support",
		Legacy:      []string{"popplio.tickets", "popplio.*"},
	},
	{
		ID:          StaffManageTickets,
		Name:        "Manage Tickets",
		Description: "Reply to and close/reopen any user's support ticket.",
		Category:    "Support",
	},

	{
		ID:          StaffModerateGuild,
		Name:        "Moderate Guild",
		Description: "Kick, ban, and timeout members in the community Discord servers.",
		Category:    "Guild Moderation",
		Dangerous:   true,
	},
	{
		ID:          StaffWarnUsers,
		Name:        "Warn Users",
		Description: "Send a formal warning to a member and log it, without kicking, banning, or timing them out.",
		Category:    "Guild Moderation",
	},
	{
		ID:          StaffPurgeMessages,
		Name:        "Purge Messages",
		Description: "Bulk-delete recent messages in a channel, optionally filtered to one user.",
		Category:    "Guild Moderation",
		Dangerous:   true,
	},
	{
		ID:          StaffLockChannels,
		Name:        "Lock/Unlock Channels",
		Description: "Deny or restore @everyone's Send Messages permission in a channel during an incident.",
		Category:    "Guild Moderation",
		Dangerous:   true,
	},
	{
		ID:          StaffViewModCases,
		Name:        "View Mod Case History",
		Description: "Look up a member's past kicks, bans, timeouts, and warns.",
		Category:    "Guild Moderation",
	},

	{
		ID:          StaffManageBadges,
		Name:        "Manage Badges",
		Description: "Create, edit and delete entries in the badge catalog.",
		Category:    "Badges",
	},
	{
		ID:          StaffAssignBadges,
		Name:        "Assign Badges",
		Description: "Assign and unassign catalog badges to and from an entity.",
		Category:    "Badges",
	},

	{
		ID:          StaffManageTemplates,
		Name:        "Manage Templates",
		Description: "Create, edit and delete the pre-built answers staff pick from when approving, denying, or otherwise reviewing a bot or server.",
		Category:    "Entity Reviews",
	},

	{
		ID:          StaffViewCDN,
		Name:        "View CDN",
		Description: "List CDN scopes and read the files in them.",
		Category:    "External Services",
		Legacy:      []string{"cdn.list_scopes", "cdn.list", "cdn#ibl@main.list", "cdn#ibl@main.read_file"},
	},
	{
		ID:          StaffManageCDN,
		Name:        "Manage CDN",
		Description: "Upload, replace and delete files on the CDN.",
		Category:    "External Services",
		Dangerous:   true,
		Legacy:      []string{"cdn.add_file", "cdn.upload_chunk"},
	},
	{
		ID:          StaffMarkerDeveloper,
		Name:        "Developer",
		Description: "Marks the holder as a developer. Carries no power on its own.",
		Category:    "Markers",
		Legacy:      []string{"developer.marker"},
	},
	{
		ID:          StaffMarkerLeadDeveloper,
		Name:        "Lead Developer",
		Description: "Marks the holder as a lead developer. Carries no power on its own.",
		Category:    "Markers",
		Legacy:      []string{"lead_developer.marker"},
	},
	{
		ID:          StaffMarkerHumanResources,
		Name:        "Human Resources",
		Description: "Marks the holder as human resources. Carries no power on its own.",
		Category:    "Markers",
		Legacy:      []string{"human_resources.marker"},
	},
	{
		ID:          StaffMarkerReviewer,
		Name:        "Reviewer",
		Description: "Marks the holder as a bot/server reviewer. Carries no power on its own.",
		Category:    "Markers",
		Legacy:      []string{"bot_reviewer.marker"},
	},
	{
		ID:          StaffMarkerServiceAccount,
		Name:        "Service Account",
		Description: "Marks the holder as a service account rather than a person. Carries no power on its own.",
		Category:    "Markers",
		Legacy:      []string{"service_account.marker"},
	},
	{
		ID:          StaffMarkerDisciplinary,
		Name:        "Under Disciplinary",
		Description: "Marks the holder as being under a disciplinary, such as a hiatus. Carries no power on its own, and is what a disciplinary that strips everything else leaves behind.",
		Category:    "Markers",
		Legacy:      []string{"dt.marker"},
	},
})
