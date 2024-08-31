package dataModel

// OData constants used in FileMaker Pro
// Used here to make it easy to swap the dollar signs in or out.
// When they are no longer required by FileMaker Pro.
const (
	COUNT                            = `$count`
	CROSSJOIN                        = `$crossjoin`
	EXPAND                           = `$expand`
	FILTER                           = `$filter`
	FORMAT                           = `$format`
	METADATA                         = `$metadata`
	NAIVE_TIMESTAMP_YYYY_MM_DD_HH_MM = `2006-01-02 15:04`
	NEXTLINK                         = `$nextLink`
	ODATAEDITLINK                    = `$odataeditlink`       // not required or supported by FileMaker. Added to facilitate extraction of @odata.editLink
	ODATAID                          = `$odataid`             // not required or supported by FileMaker. Added to facilitate extraction of @odata.id
	ODATANAVIGATIONLINK              = `$odatanavigationlink` // not supported yet. Will facilitate extraction of @odata.navigationLink
	ODATAREADLINK                    = `$odatareadlink`       // not supported yet. Will facilitate extraction of @odata.readLink
	ORDERBY                          = `$orderby`
	REF                              = `$ref`
	SELECT                           = `$select`
	SKIP                             = `$skip`
	TOP                              = `$top`
	TRUE                             = `true`
	VALUE                            = `$value`
)
