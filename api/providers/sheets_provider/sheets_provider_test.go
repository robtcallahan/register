package sheets_provider

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"register/api/clients/sheetsclient"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/sheets/v4"
)

var getRequestFunc func(spreadsheetId string, range_ string) (*sheets.ValueRange, error)

type getClientMock struct{}

//We are mocking the client method "Get"
func (c *getClientMock) Get(ssService *sheets.Service, spreadsheetId string, range_ string) (*sheets.ValueRange, error){
	return getRequestFunc(spreadsheetId, range_)
}

//When the everything is good
func TestGeValuesNoError(t *testing.T) {
	// The error we will get is from the "response" so we make the second parameter of the function is nil
	getRequestFunc = func(spreadsheetId string, range_ string) (*sheets.ValueRange, error) {
		j, err := ioutil.ReadFile("/Users/rob/ws/go/src/register/json/sheets_valuerange.json")
		assert.Nil(t, err)
		vr := sheets.ValueRange{}
		err = json.Unmarshal(j, &vr)
		assert.Nil(t, err)
		return &vr, nil
	}
	sheetsclient.ClientStruct = &getClientMock{} //without this line, the real api is fired

	sheetsProvider := New(&sheets.Service{})
	r, err := sheetsProvider.GetValues("spreadsheetID", "readRange")

	//r, err := sheetService.Service.GetValues("sheet-id", "range")
	assert.NotNil(t, r)
	assert.Nil(t, err)/**/
	assert.Len(t, r.Values, 10, "expected length of 10")

	for i, exp := range []string{"4", "WellsFargo", "11/02/20", "AA Meeting (Venmo)", " $ 5.00 ", "", "", " $ (5.00)", " $ 12,272.57 ", " $ -   "} {
		assert.EqualValues(t, exp, r.Values[0][i])
	}
}
