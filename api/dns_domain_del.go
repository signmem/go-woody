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

	rollbackNeeded := true
	defer func() {
		if rollbackNeeded {
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

	rollbackNeeded = false
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


/*
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
*/

func RemoveZoneFromFile(filePath, domainName string) error {
	// 1. open file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file %s failed: %w", filePath, err)
	}
	defer file.Close()

	// 2. get permission
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("get file info failed: %w", err)
	}

	// 3. create tempfile
	dir := "/tmp/"
	tmpFile, err := os.CreateTemp(dir, "zone.conf.*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file failed: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // remove tempfile

	// 4. scan file
	target := fmt.Sprintf(`zone "%s"`, domainName)
	scanner := bufio.NewScanner(file)
	writer := bufio.NewWriter(tmpFile)

	removed := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, target) {
			removed = true
			continue // skip file
		}
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("write to temp file failed: %w", err)
		}
	}

	// 5. check file
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan file failed: %w", err)
	}

	// 6. flush file
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush writer failed: %w", err)
	}

	// 7. close file handler
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file failed: %w", err)
	}

	// 8. check file
	if !removed {
		return fmt.Errorf("domain %s not found in zone file", domainName)
	}

	// 9. keep permission
	if err := os.Chmod(tmpPath, fileInfo.Mode()); err != nil {
		return fmt.Errorf("set temp file permission failed: %w", err)
	}

	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("replace original file failed: %w", err)
	}

	return nil
}
