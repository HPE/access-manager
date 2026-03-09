/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hpe/access-manager/internal/services/metadata"
	"github.com/spf13/cobra"
	"net/http"
	"regexp"
	"strings"
)

var annotateCmd = &cobra.Command{
	Use:   "annotate path key=value",
	Short: "Adds a user annotation to the access manager",
	Long: `
    The "annotate" command adds a user annotation to the metadata for a 
    specified path in the access manager. The annotation is given in the form
    of a key-value pair, where the key is the kind of annotation and the
    value is the value of the annotation. The key can be any string composed of
    alphanumeric characters, underscores, periods, and dashes. The value has 
    the same format.

    An example of such an annotation would be an ssh public key for a user.
    Such an annotation would be given in the form of "ssh_pubkey=ssh-rsa ...".
`,
	Args: cobra.ExactArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		return annotate(args[0], args[1])
	},
}

func annotate(path, annotation string) error {
	parts := strings.SplitN(annotation, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("annotation must be in the form key=value, got %s", annotation)
	}
	validPattern := regexp.MustCompile(`^[^\n\r]+$`)
	tag := strings.TrimSpace(parts[0])
	v := strings.TrimSpace(parts[1])
	if !validPattern.MatchString(tag) {
		return fmt.Errorf("invalid annotation key: %s", tag)
	}
	if !validPattern.MatchString(v) {
		return fmt.Errorf("invalid annotation value: %s", v)
	}
	az, err := json.Marshal(&metadata.UserAnnotation{
		Tag:  tag,
		Data: v,
	})
	if err != nil {
		return fmt.Errorf("error marshalling annotation: %w", err)
	}
	if Verbose {
		fmt.Printf("Adding annotation to %s\n", path)
	}
	body, err := request(context.Background(), "/annotate", http.MethodPost, "path", path, "annotation", string(az))
	details := struct {
		Error Status `json:"error"`
	}{}
	err = json.Unmarshal(body, &details)
	if err != nil {
		return fmt.Errorf(`error parsing response: %w`, err)
	}
	if details.Error.Error != 0 {
		return fmt.Errorf(`error annotating "%s": %s`, path, details.Error.Message)
	}

	return nil
}
