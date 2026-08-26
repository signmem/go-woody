package api

import (
	"fmt"
	"github.com/signmem/go-woody/db"
	"github.com/signmem/go-woody/g"
	"net/http"
	"path"
	"strings"
)

func getSlaveDomain(r *http.Request) (v interface{}, err error) {

	cleanPath := path.Clean(r.URL.Path)
	pathSegments := strings.Split(cleanPath, "/")
	segments := make([]string, 0)

	for _, seg := range pathSegments {
		if seg != "" {
			segments = append(segments, seg)
		}
	}

	if len(segments) == 3 && segments[0] == "api" && segments[1] == "v2" && segments[2] == "slave" {
		domainMeta, err := db.GetDomainMetaData()
		if err != nil {
			return nil, err
		}

		return domainMeta, nil
	}

	msg := fmt.Errorf("getSlaveDomain() Error: params error.")
	g.Logger.Error(msg)

	return nil, msg
}
