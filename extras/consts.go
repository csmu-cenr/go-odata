package dataModel

// OData constants used in FileMaker Pro
// Used here to make it easy to swap the dollar signs in or out.
// When they are no longer required by FileMaker Pro.
const (
	COUNT                            = `$count`
	CROSSJOIN                        = `$crossjoin`
	DEQUOTE                          = `dequote`
	EXPAND                           = `$expand`
	FALSE                            = `false`
	FILTER                           = `$filter`
	FORMAT                           = `$format`
	ID                               = `id`
	METADATA                         = `$metadata`
	NAIVE_TIMESTAMP_YYYY_MM_DD_HH_MM = `2006-01-02 15:04`
	NEXT_LINK                        = `$nextLink`
	ODATA_EDIT_LINK                  = `$odataeditlink`       // not required or supported by FileMaker. Added to facilitate extraction of @odata.editLink
	ODATA_ID                         = `$odataid`             // not required or supported by FileMaker. Added to facilitate extraction of @odata.id
	ODATA_NAVIGATION_LINK            = `$odatanavigationlink` // not supported yet. Will facilitate extraction of @odata.navigationLink
	ODATA_READ_LINK                  = `$odatareadlink`       // not supported yet. Will facilitate extraction of @odata.readLink
	ORDERBY                          = `$orderby`
	QUOTE                            = `quote`
	QUOTED                           = `quoted`
	REF                              = `$ref`
	ROWID                            = `ROWID`
	ROWMODID                         = `ROWMODID`
	SELECT                           = `$select`
	SKIP                             = `$skip`
	TOP                              = `$top`
	TRUE                             = `true`
	UUID_EQ_S                        = `uuid eq '%s'`
	VALUE                            = `$value`
)
