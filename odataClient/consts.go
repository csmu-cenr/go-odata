package odataClient

// OData constants used in FileMaker Pro
// Used here to make it easy to swap the dollar signs in or out.
// When they are no longer required by FileMaker Pro.
const (
	COUNT               = `$count`
	CROSSJOIN           = `$crossjoin`
	EXPAND              = `$expand`
	FILTER              = `$filter`
	FORMAT              = `$format`
	NEXTLINK            = `$nextLink`
	METADATA            = `$metadata`
	ODATAETAG           = `$odataetag`           // not supported yet. Will facilitate extraction of @odata.etag
	ODATAEDITLINK       = `$odataeditlink`       // not required or supported by FileMaker. Added to facilitate extraction of @odata.editLink
	ODATAID             = `$odataid`             // not required or supported by FileMaker. Added to facilitate extraction of @odata.id
	ODATANAVIGATIONLINK = `$odatanavigationlink` // not supported yet. Will facilitate extraction of @odata.navigationLink
	ODATAREADLINK       = `$odatareadlink`       // not supported yet. Will facilitate extraction of @odata.readLink
	ORDERBY             = `$orderby`
	NOTHING             = ``
	REF                 = `$ref`
	SELECT              = `$select`
	SKIP                = `$skip`
	TOP                 = `$top`
	TRUE                = `true`
	VALUE               = `$value`
)
