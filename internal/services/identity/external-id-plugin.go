/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package identity

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"golang.org/x/crypto/ssh"
)

// This plugin is a toy that illustrates how an identity plugin can maintain it's
// own user database but can allow users to log in.
//
// To make this example work, we need to create a workload for this plugin. That
// plugin should have an attribute `am://role/login-plugin`. That attribute
// should allow View and VouchFor permissions for the `am://user/yoyodyne`
// directory. Those permissions will be inherited by all of the users under that
// directory. These permissions are set up in the `new_sample` bootstrap image.
//
// To log in as one of the `yoyodyne` users run this program with a username and
// password on the command line. To get a list of legal users and their
// passwords, run this program with `--list`.
//
// If you give a valid user and password, this program will output an Access
// Manager credential for that user. Save that output to a file and set
// `AM_USER` to the name of that filename with an asterisk in front. Once you do
// that, you will be able to use the `am` program as that user.

func getUserCredential(userName string, ourCredential []byte) (string, error) {
	u, err := url.ParseRequestURI("http://localhost:8080")

	if err != nil {
		return "", fmt.Errorf("can't happen: %w", err)
	}
	u.Path = "/credential"
	params := url.Values{}
	params.Set("path", userName)
	params.Set("caller_id", string(ourCredential))
	u.RawQuery = params.Encode()
	fmt.Printf("URL: %s\n", u.String())
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return "", fmt.Errorf(`cannot create request for "%s": %w`, u.String(), err)
	}
	fmt.Printf("request: %v\n", request)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", fmt.Errorf(`cannot access "%s": %w`, u.String(), err)
	}
	fmt.Printf("response %v\n", response)
	//goland:noinspection GoUnhandledErrorResult
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf(`unexpected status code "%d"`, response.StatusCode)
	}
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf(`error reading response from %s: %w`, u.String(), err)
	}
	return string(responseBody), nil
}

var (
	privateKey = `
-----BEGIN OPENSSH PRIVATE KEY-----
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
-----END OPENSSH PRIVATE KEY-----
`
	publicKey = `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDRi48hMKqdGhOXSjGAi/x8WwEXnJExiH` +
		`XoEMcRfP9XMzo9SpxFl+X4atFtg1Y8A/YFv0UdmZ7iM6+PvS/mKQ81agbCHw+Zmbn9U+btaZVWw4vc` +
		`d3aWT5W71hhAxpwUmKPpEFIIAzjsfCUVoUAbva5ZySoDBlxEGT7fX4J6brJdqKkxwIIgo4fyxuBApa` +
		`dkZ0h4k26a06IJKfUENTkIWTRCJLHgQCLULXoYF+6LOreu6JeQnHNliNqzQpU4il533pdF8tbYA2F7` +
		`NZHdMe16vekwP3Zcl4Dax8GfkAZ95BfXjUpgPqw7hkmYJx7yYt7c6Qp+OxMC1ukfR7f/nZvPtLhJcO` +
		`ncDcFxgnUy3QHo7A7ef3OmYPEb3Umxztbjqm5SmCl9zxQNIrK+SX2751LMvs1iI6z1oDU0mrciujj6` +
		`7d+9STGAW9HuLCgbtMomMez65fhb3FgsT8Pvz/Ago8tdxdwfSBn91xYMqsI9K/gItRxYinbWDsjpMs` +
		`RYsm9L1L+C4TE= foo.local`
	hostKey = `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC4rmtkcfVvjWHqdgEvT9zZJ8v0csU6Bhzj` +
		`0W+95tG01IOCeIyarkDSIqDVy6aXEoBNU7PHpZa2Ek2WKOHVZedAHlJwAd0fSD4qocRxrFJ552lAEW` +
		`lfGOIlyRnhzukL0xG51xqT5LY7E5YO52PF5kPZhl1/ofCHspOqiIlYpOlTwrFbklaIgwT1VwABR7fj` +
		`jD4zfOVsvF2LQOY+u/eKDqcJuwYj7ELnpmePYAnRKooW67KKteyUE3GulMqKgijVlbuG0y8DLyInGg` +
		`l6yRTYHna3ut3uthvkwhy7u04YHsnLF/DEZHeayK8fIb7oftJ7bXqNR11zzxKIeYm+Tnm43D5DWI21` +
		`XdnF5HPU5l2VHwxFgYpJ4RIyvgb6N9ErQioeJhkwfrPWkY94rodMLkAqQjIT7NsI0/5awcjYXY+9+i` +
		`rYf03GNK6VIxAW1x4S1WJzIqMo8tnFZFZRhZ8DM85/M43BRjrNdBelbve2xt0p2DQqITP23gR0RsTH` +
		`6c9jHL0m0uM= hostkey.local`
)

func getSshCredential(caller string) ([]byte, error) {
	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		log.Fatalf("unable to parse private key: %v", err)
	}

	key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(hostKey))
	if err != nil {
		return nil, err
	}
	config := &ssh.ClientConfig{
		User: caller,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.FixedHostKey(key),
	}

	conn, err := ssh.Dial("tcp", "localhost:2222", config)
	if err != nil {
		log.Fatal("unable to connect: ", err)
		return nil, err
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		log.Fatal("unable to create session: ", err)
		return nil, err
	}
	defer session.Close()
	credential, err := session.CombinedOutput("")
	if err != nil {
		log.Fatal("combined failed: ", err)
		return nil, err
	}
	log.Printf("output: %s", credential)
	return credential, nil
}
