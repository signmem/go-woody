package api

import (
	"bufio"
	"database/sql"
	"fmt"
	"github.com/signmem/go-woody/db"
	"github.com/signmem/go-woody/g"
	"github.com/signmem/go-woody/tools"
	"net/http"
	"os"
	"strings"
)

func domainDelete(r *http.Request)  (domaininfo DomainInfo, err error)  {


	defer func() {
		_ = r.Body.Close()
	}()

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/domains/")
	pathParts := strings.Split(path, "/")

	if len(pathParts) != 1 || pathParts[0] == "" {
		msg := fmt.Errorf("Error: path error")
		g.Logger.Error(msg)
		return domaininfo, msg
	}

	domain_name := pathParts[0]

	tx, err := db.DB.Begin()
	if err != nil {
		msg := fmt.Errorf("Error: failed to begin transaction: %w", err)
		g.Logger.Error(msg)
		return domaininfo, msg
	}

	defer func() {
		if err != nil {
			if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
				msg := fmt.Sprintf("domainDelete() Error: transaction rollback error %v", err)
				g.Logger.Errorf(msg)
			}
		}
	}()


	domainDetail, err := db.GetDomainsByName(domain_name)

	if err != nil || domainDetail.ID == 0  {

		msg := fmt.Errorf("domainDelete Error: domain %s not found in DB.", domain_name)
		g.Logger.Error(msg)
		return domaininfo, msg

	}

	_, err = db.DeleteRecordByDomainID(tx, domainDetail.ID )

	if err != nil {

		msg := fmt.Errorf("domainDelete Error: domain id %d delete from DB " +
			"err: %w", domainDetail.ID, err)
		g.Logger.Error(msg)
		return domaininfo, msg

	}

	err = db.DeleteDomainMetaData(tx, domainDetail.ID )

	if err != nil {
		msg := fmt.Errorf("domainDelete Error: domain metadata id %d delete from DB.", pathParts[0])
		g.Logger.Error(msg)
		return domaininfo, msg
	}

	if err = tx.Commit(); err != nil {

		msg := fmt.Errorf("Error: DB commit.", pathParts[0])
		g.Logger.Error(msg)
		return domaininfo, msg

	}

	zoneFile := g.Config().ZoneFile

	if g.Config().Named == true {

		err = RemoveZoneFromFile(zoneFile, domaininfo.DomainName)
		if err != nil {
			msg := fmt.Errorf("Error: domainDelete() delete domain  %s file " +
				"write error: %s", domaininfo.DomainName, err)
			g.Logger.Error(msg)
			// return domaininfo, msg
		}

		err = tools.RestartNamed()
		if err != nil {
			msg := fmt.Sprintf("domainDelete() Error: restart named %s", err)
			g.Logger.Error(msg)
			return domaininfo, err
		}

	}

	domaininfo.DomainID = domainDetail.ID
	domaininfo.DomainName  = domainDetail.Name

	msg := fmt.Sprintf("domainDelete() delete domain  %s Success", domaininfo.DomainName)

	if g.Config().Debug == true {
		g.Logger.Debug(msg)
	}

	return domaininfo, nil
}



func RemoveZoneFromFile(filePath, domainName string) error {

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file %s failed: %w", filePath, err)
	}
	defer file.Close()

	target := fmt.Sprintf(`zone "%s"`, domainName)
	var keptLines []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, target) {
			continue
		}
		keptLines = append(keptLines, line)
	}

	if err = scanner.Err(); err != nil {
		return fmt.Errorf("scan file failed: %w", err)
	}

	outputContent := strings.Join(keptLines, "\n")
	err = os.WriteFile(filePath, []byte(outputContent), 0644)
	if err != nil {
		return fmt.Errorf("write file failed: %w", err)
	}
	return nil

}
