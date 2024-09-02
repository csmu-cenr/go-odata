package odataClient

// OData constants used in FileMaker Pro
// Used here to make it easy to swap the dollar signs in or out.
// When they are no longer required by FileMaker Pro.
const (
	COMMA               = `,`
	COUNT               = `$count`
	CROSSJOIN           = `$crossjoin`
	DOUBLE_SINGLE_QUOTE = `''`
	EXPAND              = `$expand`
	FILTER              = `$filter`
	FORMAT              = `$format`
	LEFT_BRACKET        = `(`
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
	RIGHT_BRACKET       = `)`
	SELECT              = `$select`
	SKIP                = `$skip`
	TOP                 = `$top`
	TRUE                = `true`
	UNEXPECTED_ERROR    = `unexpected error`
	VALUE               = `$value`
)
