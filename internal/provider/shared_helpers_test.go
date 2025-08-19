// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"
	"time"
)

func TestWithTimeout_NoDeadlineWhenZero(t *testing.T) {
	ctx := context.Background()
	ctx2, cancel := withTimeout(ctx, 0)
	defer cancel()
	if dl, ok := ctx2.Deadline(); ok {
		t.Fatalf("expected no deadline, got %v", dl)
	}
}

func TestWithTimeout_HasDeadlineWhenPositive(t *testing.T) {
	d := 100 * time.Millisecond
	before := time.Now()
	ctx2, cancel := withTimeout(context.Background(), d)
	defer cancel()
	dl, ok := ctx2.Deadline()
	if !ok {
		t.Fatalf("expected a deadline")
	}
	// Allow small scheduling slack (<= +50ms)
	if dl.Before(before.Add(d-10*time.Millisecond)) || dl.After(before.Add(d+50*time.Millisecond)) {
		t.Fatalf("deadline out of expected range; before=%v d=%v got=%v", before, d, dl)
	}
}
