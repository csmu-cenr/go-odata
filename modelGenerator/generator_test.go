package modelGenerator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Generate_struct(t *testing.T) {
	edmx, _ := getParsedEdmx()
	peopleSet := edmx.EntitySets["People"]
	g := Generator{}

	assert.Equal(t, `type Person struct {
	AddressInfo []Location
	Age nullable.Nullable[int64]
	Emails []string
	FavoriteFeature Feature
	Features []Feature
	FirstName string
	Gender PersonGender
	HomeAddress nullable.Nullable[Location]
	LastName nullable.Nullable[string]
	MiddleName nullable.Nullable[string]
	UserName string
}`, g.generateModelStruct(peopleSet.getEntityType(), map[string]string{}))
}

func Test_Generate_enum(t *testing.T) {
	edmx, _ := getParsedEdmx()
	genderEnum := edmx.EnumTypes["PersonGender"]
	g := Generator{}
	assert.Equal(t, `type PersonGender int64

const (
	Male PersonGender = 0
	Female PersonGender = 1
	Unknown PersonGender = 2
)`, g.generateEnumStruct(genderEnum))
}
