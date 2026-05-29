package odataClient

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Uffe-Code/go-nullable/nullable"
	"github.com/stretchr/testify/assert"
)

type testModel struct {
	Id          int                       `json:"Id"`
	Number      string                    `json:"Number"`
	Name        string                    `json:"Name"`
	ParentId    nullable.Nullable[int]    `json:"ParentId"`
	Description nullable.Nullable[string] `json:"Description"`
}

type testModelDefinition[T any] struct {
	client ODataClient
}

func (t testModelDefinition[T]) DataSet() ODataDataSet[T, ODataModelDefinition[T]] {
	return NewDataSet[T](t.client, t)
}

func (t testModelDefinition[T]) Name() string {
	return "Person"
}

func (t testModelDefinition[T]) Url() string {
	return "People"
}

func newTestModelDefinition(wrapper Wrapper) ODataModelCollection[testModel] {
	return testModelDefinition[testModel]{client: wrapper.ODataClient()}
}

func TestNewDataSet(t *testing.T) {
	client := New("http://test.api/")
	dataSet := newTestModelDefinition(client).DataSet()
	assert.Equal(t, "http://test.api/People", dataSet.getCollectionUrl())
	assert.Equal(t, "http://test.api/People(5)", dataSet.getSingleUrl("5"))
}

func TestNewDataSet_WithoutSlash(t *testing.T) {
	client := New("http://test.api")
	dataSet := newTestModelDefinition(client).DataSet()
	assert.Equal(t, "http://test.api/People", dataSet.getCollectionUrl())
	assert.Equal(t, "http://test.api/People(5)", dataSet.getSingleUrl("5"))
}

func TestOdataDataSet_Single(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/People(5)" {
			writer.WriteHeader(404)
			return
		}
		writer.WriteHeader(200)
		data, _ := json.Marshal(testModel{
			Id:          5,
			Number:      "002",
			Name:        "Donald Duck",
			ParentId:    nullable.Null[int](),
			Description: nullable.Value("Test description"),
		})
		_, _ = writer.Write(data)
	}))
	defer testServer.Close()

	client := New(testServer.URL)
	def := newTestModelDefinition(client)
	dataSet := def.DataSet()
	model, err := dataSet.Single("5", ODataQueryOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 5, model.Id)
	assert.Equal(t, "002", model.Number)
	assert.Equal(t, "Donald Duck", model.Name)
	assert.False(t, model.ParentId.IsNotZero())
	assert.True(t, model.Description.IsNotEmpty())
	assert.Equal(t, "Test description", model.Description.Data)
}

func TestOdataDataSet_List(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/People" {
			writer.WriteHeader(404)
			return
		}
		writer.WriteHeader(200)
		data, _ := json.Marshal(struct {
			Value []testModel `json:"value"`
		}{
			Value: []testModel{
				{
					Id:          3,
					Number:      "001",
					Name:        "Uffe Code",
					ParentId:    nullable.Null[int](),
					Description: nullable.Value("Test description"),
				},
				{
					Id:          5,
					Number:      "002",
					Name:        "Donald Duck",
					ParentId:    nullable.Null[int](),
					Description: nullable.Value("Test description"),
				},
			},
		})
		_, _ = writer.Write(data)
	}))
	defer testServer.Close()

	client := New(testServer.URL)
	def := newTestModelDefinition(client)
	dataSet := def.DataSet()
	metaChan, modelsChan, errsChan := dataSet.List(ODataQueryOptions{})

	var models []testModel
	done := make(chan bool)
	go func() {
		for m := range modelsChan {
			models = append(models, m)
		}
		done <- true
	}()

	for err := range errsChan {
		assert.NoError(t, err)
	}

	for meta := range metaChan {
		assert.Equal(t, "People", meta.Model)
	}
	<-done

	assert.Equal(t, 2, len(models))
	assert.Equal(t, 3, models[0].Id)
	assert.Equal(t, "001", models[0].Number)
	assert.Equal(t, 5, models[1].Id)
	assert.Equal(t, "002", models[1].Number)
}

func Test_Insert(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/People" {
			writer.WriteHeader(404)
			return
		}
		if request.Method != "POST" {
			writer.WriteHeader(404)
			panic("wrong request method " + request.Method)
		}
		writer.WriteHeader(200)
		buf := &bytes.Buffer{}
		_, err := buf.ReadFrom(request.Body)
		if err != nil {
			panic(err.Error())
		}
		_, _ = writer.Write(buf.Bytes())
	}))
	defer testServer.Close()

	client := New(testServer.URL)
	def := newTestModelDefinition(client)
	dataSet := def.DataSet()
	model := testModel{
		Id:          0,
		Number:      "1234",
		Name:        "FooBar",
		ParentId:    nullable.Null[int](),
		Description: nullable.Null[string](),
	}

	res, err := dataSet.Insert(model, []string{})

	assert.NoError(t, err)
	assert.Equal(t, 0, res.Id)
	assert.False(t, res.ParentId.IsNotEmpty())
	assert.Equal(t, "FooBar", res.Name)
}

func Test_Update(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/People(5)" {
			writer.WriteHeader(404)
			return
		}
		if request.Method != "PATCH" {
			writer.WriteHeader(404)
			panic("wrong request method " + request.Method)
		}
		writer.WriteHeader(200)

		// Read the body to see what was sent
		buf := &bytes.Buffer{}
		_, _ = buf.ReadFrom(request.Body)
		var sentData map[string]interface{}
		_ = json.Unmarshal(buf.Bytes(), &sentData)

		// Return a response that matches the expected model in the test
		// The test expects Id: 5, Number: "1234", Name: "FooBar"
		data, _ := json.Marshal(testModel{
			Id:          5,
			Number:      "1234",
			Name:        "FooBar",
			ParentId:    nullable.Null[int](),
			Description: nullable.Null[string](),
		})
		_, _ = writer.Write(data)
	}))
	defer testServer.Close()

	client := New(testServer.URL)
	def := newTestModelDefinition(client)
	dataSet := def.DataSet()
	model := testModel{
		Id:          5,
		Number:      "1234",
		Name:        "FooBar",
		ParentId:    nullable.Null[int](),
		Description: nullable.Null[string](),
	}

	res, err := dataSet.Update("5", model, url.Values{"select": []string{"Id,Number,Name,ParentId,Description"}})

	assert.NoError(t, err)
	assert.Equal(t, 5, res.Id)
	assert.Equal(t, "1234", res.Number)
	assert.False(t, res.ParentId.IsNotEmpty())
	assert.Equal(t, "FooBar", res.Name)
}
