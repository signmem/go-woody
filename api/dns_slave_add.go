package api

import (
	"encoding/json"
	"fmt"
	"github.com/signmem/go-woody/db"
	"github.com/signmem/go-woody/g"
	"io"
	"mime"
	"net/http"
)

func syncSlaveDomain(r *http.Request) (htmlMsg ReturnMsg, err error) {

	headerContentType := r.Header.Get("Content-Type")

	mediaType, _, err := mime.ParseMediaType(headerContentType)
	if err != nil || mediaType != "application/json" {
		msg := fmt.Errorf("syncSlaveDomain() Error: body not json format")
		g.Logger.Error(msg)
		htmlMsg.Msg = "syncSlaveDomain() Post data not valid, body not json format!"
		return htmlMsg, msg
	}

	defer func() {
		_ = r.Body.Close()
	}()

	body, err := io.ReadAll(r.Body)

	if err != nil {
		msg := fmt.Errorf("syncSlaveDomain() Error: body read error")
		g.Logger.Error(msg)
		htmlMsg.Msg = "syncSlaveDomain() Post data not valid, body read error!"
		return htmlMsg, msg
	}

	var syncStatus SyncSlaveDomain
	err = json.Unmarshal(body, &syncStatus)

	if err != nil {
		msg := fmt.Errorf("syncSlaveDomain() Error: body json unmarshal error")
		g.Logger.Error(msg)
		htmlMsg.Msg = "syncSlaveDomain() Post data not valid, body json unmarshal format error!"
		return htmlMsg, msg
	}

	TrimAllStrings(&syncStatus)

	if syncStatus.Sync == false{
		msg := fmt.Errorf("syncSlaveDomain() Error: sync status is false")
		g.Logger.Error(msg)
		htmlMsg.Msg = "syncSlaveDomain() Error: sync status is false!"
		return htmlMsg, msg
	}

	domainType := "MASTER"
	masterDomain, err := db.GetAllPDNSDomain(domainType)

	if err != nil {
		msg := fmt.Errorf("syncSlaveDomain() get domain info err: %s", err)
		g.Logger.Error(msg)
		htmlMsg.Msg = fmt.Sprintf("syncSlaveDomain() get domain info err:%s", err)
		return htmlMsg, msg
	}

	var success, failed int

	for _, domain := range masterDomain {

		domainID := domain.ID
		var s, f int

		err := func() error {
			tx, err := db.DB.Begin()
			if err != nil {
				return err
			}

			commitOk := false
			defer func() {
				if !commitOk {
					_ = tx.Rollback()
				}
			}()


			sLoc, fLoc, err := DomainMetaDataAdd(tx, domainID)

			if err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return err
			}

			commitOk = true

			s = sLoc
			f = fLoc
			return nil
		}()

		if err != nil {
			g.Logger.Errorf("sync domain %d failed: %v", domain.ID, err)
			failed++
			continue
		}

		success += s
		failed += f

	}

	msg := fmt.Sprintf("domain metadata insert success %d, skip %d", success, failed)
	htmlMsg.Msg = msg
	return htmlMsg, nil
}
