package downloader

import (
	"database/sql/driver"
	"strings"
	"time"
)

type ISODate struct {
	time.Time
}

func (sd *ISODate) UnmarshalJSON(input []byte) error {
	strInput := string(input)
	strInput = strings.Trim(strInput, `"`)
	newTime, err := time.Parse("2006-01-02", strInput)
	if err != nil {
		return err
	}
	sd.Time = newTime
	return nil
}

func (sd ISODate) Value() (driver.Value, error) {
	return sd.Format("2006-01-02"), nil
}
