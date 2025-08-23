// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

package provider

// validationErr captures a configuration validation error and optional attribute path.
type validationErr struct {
	attr    string // empty for general error
	summary string
	detail  string
}

// resolvedConfig contains normalized provider configuration used to initialize clients.
type resolvedConfig struct {
	endpoint              string
	authMethod            string
	email                 string
	apiToken              string
	username              string
	password              string
	httpTimeoutSeconds    int
	retryOn4295xx         bool
	retryMaxAttempts      int
	retryInitialBackoffMs int
	retryMaxBackoffMs     int
	emailRedactionMode    string
}
