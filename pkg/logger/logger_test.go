/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package logger

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestInitLogger(t *testing.T) {
	var out bytes.Buffer
	InitLogger("info", "test", &out)
	GetLogger().Info().Msg("foo!")
	var m map[string]any
	err := json.Unmarshal(out.Bytes(), &m)
	if err != nil {
		t.Errorf("failed to get map: %s", err.Error())
	}
	// the actual commit is only available when building with modules
	// we check for the default message instead
	if m["git.commit"] != "no commit available" {
		t.Fatalf("expected default commit info, got %s", m["git.commit"])
	}
}
