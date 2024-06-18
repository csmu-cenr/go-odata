package dataModel

func ExtraFields() []string {
	/*
		ODataId       string `json:"@odata.id,omitempty"`
		ODataEditLink string `json:"@odata.editLink,omitempty"`
		ODataEtag     string `json:"@odata.etag,omitempty"`
		ODataReadLink string `json:"@odata.readLink,omitempty"`
		RowId         int    `json:"ROWID,omitempty"`
		RowModId      int    `json:"ROWMODID,omitempty"`
	*/
	//result := []string{`ODataId`, `ODataEditLink`, `ODataEtag`, `ODataReadLink`, `RowId`, `RowModId`}
	result := []string{`ODataId`, `ODataEditLink`, `RowId`, `RowModId`}
	return result
}
