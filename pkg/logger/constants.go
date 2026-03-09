/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package logger

const (
	SubscriptionID  string = "subscription_id"
	RequestID       string = "request_id"
	CallerID        string = "caller_id"
	Version         string = "version"
	Global          string = "global"
	Expiry          string = "expiry"
	IncludeChildren string = "include_children"
	Path            string = "path"
	Role            string = "role"
	PathURL         string = "path_url"
	PathWildCardURL string = "path_wild_card_url"
)

// DebugContextKeys defines variables from the context to be included in each log line
var DebugContextKeys = []string{SubscriptionID, RequestID}
