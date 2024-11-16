package serverscalc

import (
	"log"
	"testing"
)

func TestExtractNumberServer11(t *testing.T) {
	url := "https://api.sms-man.com/control/get-number?token=kdB2QOTDWF6hwgywghVwQGvKNALFoZnU&application_id=1491&country_id=14&hasMultipleSms=false"
	id, number, err := ExtractNumberServer11(url)
	if err != nil {
		t.Error(err)
	}
	log.Println(id, number)
}
