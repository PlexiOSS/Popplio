// Copyright (C) 2026 NodeByte LTD

package webhooks

import (
	"popplio/webhooks/core/drivers"
	"popplio/webhooks/core/events"
	_ "popplio/webhooks/events"
	_ "popplio/webhooks/hooks"

	docs "github.com/PlexiOSS/Keel/doclib"
)

func Setup() {
	docs.AddTag(
		"Webhooks",
		"Webhooks are a way to receive events from Omniplex in real time. You can use webhooks to receive events such as new votes, new reviews, and more.",
	)

	events.RegisterAddedEvents()
	go drivers.PullPendingForAll()
}
