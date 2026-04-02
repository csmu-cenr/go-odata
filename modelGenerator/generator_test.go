package modelGenerator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Generate_struct(t *testing.T) {
	edmx, _ := getParsedEdmx()
	peopleSet := edmx.EntitySets["People"]
	// 	g := Generator{}

	g := Generator{}
	actual := g.generateModelStruct(peopleSet.getEntityType())

	assert.Contains(t, actual, `type Person struct {`)
	assert.Contains(t, actual, `AddressInfo []Location`)
	assert.Contains(t, actual, `Age nullable.Nullable[int64]`)
	assert.Contains(t, actual, `Emails []string`)
	assert.Contains(t, actual, `FavoriteFeature Feature`)
	assert.Contains(t, actual, `Features []Feature`)
	assert.Contains(t, actual, `FirstName string`)
	assert.Contains(t, actual, `Gender PersonGender`)
	assert.Contains(t, actual, `HomeAddress nullable.Nullable[Location]`)
	assert.Contains(t, actual, `LastName nullable.Nullable[string]`)
	assert.Contains(t, actual, `MiddleName nullable.Nullable[string]`)
	assert.Contains(t, actual, `UserName string`)
	assert.Contains(t, actual, `type	PersonAlias	[]Person`)
	assert.Contains(t, actual, `type MetaPerson struct {`)

}

func Test_Generate_definition(t *testing.T) {
	edmx, _ := getParsedEdmx()
	peopleSet := edmx.EntitySets["People"]

	assert.Equal(t, `//goland:noinspection GoUnusedExportedFunction
func NewPersonCollection(wrapper odataClient.Wrapper) odataClient.ODataModelCollection[Person] {
	return modelDefinition[Person]{client: wrapper.ODataClient(), name: "Person", url: "People"}
}`, generateModelDefinition(peopleSet))
}

func Test_Generate_enum(t *testing.T) {
	edmx, _ := getParsedEdmx()
	genderEnum := edmx.EnumTypes["PersonGender"]
	assert.Equal(t, `type PersonGender int64

const (
	Male PersonGender = 0
	Female PersonGender = 1
	Unknown PersonGender = 2
)`, generateEnumStruct(genderEnum))
}
