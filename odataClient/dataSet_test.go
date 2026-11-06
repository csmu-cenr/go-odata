package odataClient

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Uffe-Code/go-nullable/nullable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testModel struct {
	Id          int
	Number      string
	Name        string
	ParentId    nullable.Nullable[int]
	Description nullable.Nullable[string]
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
		assert.Equal(t, http.MethodGet, request.Method)
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
	require.NoError(t, err)
	assert.Equal(t, 5, model.Id)
	assert.Equal(t, "002", model.Number)
	assert.Equal(t, "Donald Duck", model.Name)
	assert.False(t, model.ParentId.IsValid)
	assert.True(t, model.Description.IsValid)
	assert.Equal(t, "Test description", model.Description.Data)
}

func TestOdataDataSet_List(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Equal(t, "true", request.URL.Query().Get("$count"))
		if request.URL.Path != "/People" {
			writer.WriteHeader(404)
			return
		}
		writer.WriteHeader(200)
		data, _ := json.Marshal(struct {
			Count int         `json:"@odata.count"`
			Value []testModel `json:"value"`
		}{
			Count: 2,
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
	meta, models, errs := dataSet.List(ODataQueryOptions{Count: "true"})
	var got []testModel
	var metadata []Result
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	// List sends on unbuffered channels; consume all three until they close.
	for meta != nil || models != nil || errs != nil {
		select {
		case result, ok := <-meta:
			if !ok {
				meta = nil
				continue
			}
			metadata = append(metadata, result)
		case model, ok := <-models:
			if !ok {
				models = nil
				continue
			}
			got = append(got, model)
		case err, ok := <-errs:
			if !ok {
				errs = nil
				continue
			}
			assert.NoError(t, err)
		case <-timer.C:
			t.Fatal("timed out waiting for List channels to close")
		}
	}
	require.Len(t, metadata, 1)
	assert.Equal(t, "People", metadata[0].Model)
	require.NotNil(t, metadata[0].Count)
	assert.Equal(t, 2, *metadata[0].Count)
	assert.Equal(t, []testModel{
		{Id: 3, Number: "001", Name: "Uffe Code", ParentId: nullable.Null[int](), Description: nullable.Value("Test description")},
		{Id: 5, Number: "002", Name: "Donald Duck", ParentId: nullable.Null[int](), Description: nullable.Value("Test description")},
	}, got)
}

func Test_Insert(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/People" {
			writer.WriteHeader(404)
			return
		}
		if !assert.Equal(t, http.MethodPost, request.Method) {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		assert.Equal(t, "application/json;odata.metadata=minimal", request.Header.Get("Content-Type"))
		assert.Equal(t, "return=representation", request.Header.Get("Prefer"))
		writer.WriteHeader(200)
		buf := &bytes.Buffer{}
		_, err := buf.ReadFrom(request.Body)
		if err != nil {
			t.Error(err)
			return
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
	require.NoError(t, err)
	assert.Equal(t, 0, res.Id)
	assert.False(t, res.ParentId.IsValid)
	assert.Equal(t, "FooBar", res.Name)
	assert.Equal(t, model, res)
}

func Test_Update(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/People(5)" {
			writer.WriteHeader(404)
			return
		}
		if !assert.Equal(t, http.MethodPatch, request.Method) {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		assert.Equal(t, "application/json;odata.metadata=minimal", request.Header.Get("Content-Type"))
		assert.Equal(t, "return=representation", request.Header.Get("Prefer"))
		writer.WriteHeader(200)
		buf := &bytes.Buffer{}
		_, err := buf.ReadFrom(request.Body)
		if err != nil {
			t.Error(err)
			return
		}
		_, _ = writer.Write(buf.Bytes())
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
	res, err := dataSet.Update("5", model, []string{})
	require.NoError(t, err)
	assert.Equal(t, 5, res.Id)
	assert.False(t, res.ParentId.IsValid)
	assert.Equal(t, "FooBar", res.Name)
	assert.Equal(t, model, res)
}
