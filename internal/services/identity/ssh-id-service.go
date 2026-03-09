/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package identity

import (
	"fmt"
	"github.com/hpe/access-manager/internal/services/metadata"
	"log"
	"time"

	"github.com/gliderlabs/ssh"
)

// NewIntegratedIdentityService starts a service on the designated port that will
// return a user credential when a user connects on the designated port. The
// client should connect with the ssh user id set to the desired access manager
// identity. To be allowed to connect, the user's claimed identity must have an
// `ssh-pubkey` annotation with a public key in the form used in the
// `authorized_keys` used by ssh. The hostkey is the name of a file that contains
// an ssh private key. This is used by ssh as the host key for the ssh connection
// so that clients can populate their `known_hosts` file.
func NewIntegratedIdentityService(hostkey string, port int, meta metadata.MetaStore) {
	// callback to validate the user key
	publicKeyOption := ssh.PublicKeyAuth(func(c ssh.Context, key ssh.PublicKey) bool {
		user := c.Value(ssh.ContextKeyUser).(string)
		annotations, err := meta.Get(c, user, metadata.WithType("ssh-pubkey"))
		if err != nil {
			return false
		}
		if len(annotations) == 0 {
			noIdentityPlugins := true
			err := meta.ScanPath(c, user, true, func(_ string, a *metadata.ACE, done error) error {
				if err != nil {
					return err
				}
				if a != nil && a.Op == metadata.VouchFor {
					noIdentityPlugins = false
					return done
				}
				return nil
			})
			return err == nil && noIdentityPlugins
		}
		for _, annotation := range annotations {
			ua, err := annotation.AsUserAnnotation()
			if err != nil {
				return false
			}
			annotatedKey, _, _, rest, err := ssh.ParseAuthorizedKey([]byte(ua.Data))
			if len(rest) > 0 {
				continue
			}
			if err != nil {
				log.Printf("error decoding key for %s: %s", user, err.Error())
			}
			if ssh.KeysEqual(annotatedKey, key) {
				return true
			}
		}
		return false
	})
	// get the JWT for the user
	ssh.Handle(func(s ssh.Session) {
		cred, err := meta.GetSignedJWTWithClaims(1*time.Hour, metadata.DefaultClaims(s.User(), nil))
		if err != nil {
			// couldn't create the JWT ... nothing worth returning
			log.Printf("error getting token: %s", err.Error())
			return
		}
		fmt.Printf("user: %s\ncred: %s\n", s.User(), cred)
		_, _ = fmt.Fprintf(s, "%s\n", cred)
	})

	hostKeyOption := ssh.HostKeyFile(hostkey)

	log.Fatal(ssh.ListenAndServe(fmt.Sprintf(":%d", port), nil, publicKeyOption, hostKeyOption))
}
