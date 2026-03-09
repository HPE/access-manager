/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package common

import (
	"encoding/json"
	"math/rand/v2"
	"regexp"
	"strings"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
)

/*
Join Cleanly concatenates two strings separated by a '/', but avoiding doubled slashes
*/
func Join(a, b string) string {
	return Cleanup(a + "/" + b)
}

/*
Cleanup eliminates double / (except after colon) and trims trailing /
*/
func Cleanup(a string) string {
	r := strings.Builder{}
	last := a[0]
	for i := 0; i < len(a); i++ {
		if last == ':' {
			r.WriteByte('/')
		}

		if a[i] != '/' {
			if last == '/' {
				r.WriteByte('/')
			}
			r.WriteByte(a[i])
		}

		last = a[i]
	}

	return r.String()
}

/*
PathComponents returns an array of indexes for all proper component prefixes of a URI omitting the
standard lead-in.

For a URI of "am://abc/def/g", this will give back [6, 9, 13, 15].
```
"am://abc/def/g"[0:6] = "am://"
"am://abc/def/g"[0:9] = "am://abc"
"am://abc/def/g"[0:13] = "am://abc/def"
"am://abc/def/g"[0:15] = "am://abc/def/g"
```
*/
func PathComponents(path string) []int {
	if !strings.HasPrefix(path, StandardPrefix) {
		return []int{}
	}
	base := len(StandardPrefix)
	n := len(path)
	if path[n-1] == '/' {
		n -= 1
	}
	r := []int{base}
	for i := base; i < n; {
		delta := strings.Index(path[i:], "/")
		if delta < 0 {
			r = append(r, n)
			return r
		}
		i += delta
		r = append(r, i)
		i++
	}
	return r
}

func CleanJson(actual any) string {
	b1, err := json.MarshalIndent(actual, "", "   ")
	if err != nil {
		return ""
	}
	return FixJson(b1)
}

func EqualSets[T comparable](actual, expected []T) (bool, mapset.Set[T]) {
	difference := mapset.NewSet(actual...).SymmetricDifference(mapset.NewSet(expected...))
	return difference.Cardinality() == 0, difference
}

func IsExpired(endTimeInMilli int64) bool {
	//nolint:gosec
	if endTimeInMilli != 0 && endTimeInMilli <= time.Now().UnixMilli() {
		return true
	}

	return false
}

func FixJson(b []byte) string {
	empty := []*regexp.Regexp{
		regexp.MustCompile(`\n\s*\n`),
		regexp.MustCompile(`\s*{},?\s*\n`),
		regexp.MustCompile(`\s*"Children":\s*\[\s*],?\s*\n`),
		regexp.MustCompile(`\s*"Meta":\s*\[\s*],?\s*\n`),
		regexp.MustCompile(`\s*"Role":\s*"",?\s*\n`),
		regexp.MustCompile(`\s*"SpiffeId":\s*"",?\s*\n`),
		regexp.MustCompile(`\s*"Value": {\s*"Op": "",\s*"Local": false,\s*"Permissions": \[],\s*}\n`),
		regexp.MustCompile(`\s*"Annotation": {\s*}\n`),
		regexp.MustCompile(`\s*"Op": "",\s*"Local": false,\s*"Permissions": \[],\n`),
		regexp.MustCompile(`\s*"roles":\s*\[\s*],?\s*\n`),
		regexp.MustCompile(`\s*"inheritedRoles":\s*\[\s*],?\s*\n`),
		regexp.MustCompile(`\s*"aces":\s*\[\s*],?\s*\n`),
		regexp.MustCompile(`\s*"inheritedAces":\s*\[\s*],?\s*\n`),
	}
	permfix := regexp.MustCompile(`"permissions": \[\s*{\s*"roles": \[\n\s*("[^"]+")\s*]\s*}\s*]`)

	s0 := strings.ReplaceAll(string(b), ": null", ": []")
	s1 := permfix.ReplaceAllString(s0, `"permissions": [{"roles": [$1]}]`)
	s := ""
	for i := 0; i < 10 && s != s1; i++ {
		s = s1
		for _, re := range empty {
			s1 = re.ReplaceAllString(s1, "\n")

		}
	}

	if s == "null" {
		return "[]"
	} else {
		return s
	}
}

func Parent(path string) string {
	if path == StandardPrefix {
		return StandardPrefix
	}
	pieces := PathComponents(path)
	return path[:pieces[len(pieces)-2]]
}

// SafeUnique generates a random integer that javascript
// won't choke on
func SafeUnique() int64 {
	return rand.Int64() & ((1 << 53) - 1) //nolint:gosec
}

func ValidPrincipal(id string) bool {
	return strings.HasPrefix(id, UserPrefix) || strings.HasPrefix(id, WorkloadPrefix)
}
