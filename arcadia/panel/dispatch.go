package panel

import (
	"context"
	"net/http"

	"popplio/arcadia/types"
)

/*
dispatch is the main entrypoint for all panel queries. It is responsible for routing the request to the appropriate handler based on the fields present in the request.
Each case in the switch statement corresponds to a specific type of request, and the appropriate handler function is called with the context and request data.
If no recognized operation is specified in the request, an error response is returned indicating a bad request.
*/
func (s *Server) dispatch(ctx context.Context, req *types.PanelQuery) (response, error) {
	switch {
	case req.Authorize != nil:
		return s.authorize(ctx, req.Authorize)
	case req.Hello != nil:
		return s.hello(ctx, req.Hello)
	case req.BaseAnalytics != nil:
		return s.baseAnalytics(ctx, req.BaseAnalytics)
	case req.GetUser != nil:
		return s.getUser(ctx, req.GetUser)
	case req.BotQueue != nil:
		return s.botQueue(ctx, req.BotQueue)
	case req.ServerQueue != nil:
		return s.serverQueue(ctx, req.ServerQueue)
	case req.ExecuteRpc != nil:
		return s.executeRpc(ctx, req.ExecuteRpc)
	case req.GetRpcMethods != nil:
		return s.getRpcMethods(ctx, req.GetRpcMethods)
	case req.GetRpcLogEntries != nil:
		return s.getRpcLogEntries(ctx, req.GetRpcLogEntries)
	case req.SearchEntitys != nil:
		return s.searchEntitys(ctx, req.SearchEntitys)
	case req.UpdatePartners != nil:
		return s.updatePartners(ctx, req.UpdatePartners)
	case req.UpdateReports != nil:
		return s.updateReports(ctx, req.UpdateReports)
	case req.UpdateChangelog != nil:
		return s.updateChangelog(ctx, req.UpdateChangelog)
	case req.UpdateBlog != nil:
		return s.updateBlog(ctx, req.UpdateBlog)
	case req.UpdateStaffPositions != nil:
		return s.updateStaffPositions(ctx, req.UpdateStaffPositions)
	case req.UpdateStaffMembers != nil:
		return s.updateStaffMembers(ctx, req.UpdateStaffMembers)
	case req.UpdateStaffDisciplinaryType != nil:
		return s.updateStaffDisciplinaryType(ctx, req.UpdateStaffDisciplinaryType)
	case req.UpdateVoteCreditTiers != nil:
		return s.updateVoteCreditTiers(ctx, req.UpdateVoteCreditTiers)
	case req.UpdateShopItems != nil:
		return s.updateShopItems(ctx, req.UpdateShopItems)
	case req.UpdateShopItemBenefits != nil:
		return s.updateShopItemBenefits(ctx, req.UpdateShopItemBenefits)
	case req.UpdateShopCoupons != nil:
		return s.updateShopCoupons(ctx, req.UpdateShopCoupons)
	case req.UpdateBotWhitelist != nil:
		return s.updateBotWhitelist(ctx, req.UpdateBotWhitelist)
	case req.UpdateBadges != nil:
		return s.updateBadges(ctx, req.UpdateBadges)
	case req.PopplioStaff != nil:
		return s.popplioStaff(ctx, req.PopplioStaff)
	default:
		return response{}, errStatus(http.StatusBadRequest, "No operation was specified")
	}
}
