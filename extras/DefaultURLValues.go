package dataModel

import (
	"fmt"
	"net/url"
)

func DefaultURLValues(rowModId int) url.Values {

	values := url.Values{}
	if rowModId > 0 {
		//
		filter := fmt.Sprintf(`%s eq %d`, ROWMODID, rowModId)
		values.Set(FILTER, filter)
	}
	values.Set(QUOTED, FALSE)
	values.Set(QUOTE, ID)
	values.Set(ODATA_EDIT_LINK, TRUE)
	values.Set(DEQUOTE, ROWID)
	values.Add(DEQUOTE, ROWMODID)

	return values

}
