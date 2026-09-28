package api

import (
	"fmt"
	"github.com/signmem/go-woody/db"
	"net/http"
	"path"
	"strconv"
	"strings"
)

func domainSOAGet(r *http.Request) (v interface{}, err error) {

	cleanPath := path.Clean(r.URL.Path)
	pathSegments := strings.Split(cleanPath, "/")
	segments := make([]string, 0)

	for _, seg := range pathSegments {
		if seg != "" {
			segments = append(segments, seg)
		}
	}

	if len(segments) == 5 && segments[0] == "api" && segments[1] == "v2" &&
		segments[2] == "soa" && segments[3] == "domain_id" {
		id := segments[4]
		idInt, err := strconv.Atoi(id)

		if err != nil {
			msg := fmt.Errorf("domain_id must be digital")
			return nil, msg
		}

		if idInt <= 0 {
			return nil, fmt.Errorf("domain_id must greater than zero")
		}

		return db.GetDomainSOAByID(int64(idInt))
	}

	if len(segments) == 5 && segments[0] == "api" && segments[1] == "v2" &&
		segments[2] == "soa" && segments[3] == "domain_name" {

		domain_name := strings.TrimSpace(segments[4])
		if len(domain_name) == 0 {
			msg := fmt.Errorf("domain_name can not be none")
			return nil, msg
		}

		return db.GetDomainSOAByName(domain_name)

	}

	if len(segments) == 3 && segments[0] == "api" && segments[1] == "v2" && segments[2] == "soa" {

		return db.GetDomainSOA()
	}

	err = fmt.Errorf("Error: params error.")
	return nil, err
}