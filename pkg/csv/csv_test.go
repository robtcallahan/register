package csv

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	cfg "register/pkg/config"
	"register/pkg/models"
)

const (
	FinanceDir = "/Users/rob/Dropbox/Finances/"
	JSONDIR    = "/Users/rob/ws/go/src/register/api/services/sheets_service/json/"
)

type FakeReadFiler struct {
	Str string
}

// ReadFile here's a fake ReadFile method that matches the signature of ioutil.ReadFile
func (f FakeReadFiler) ReadFile(filename string) ([]byte, error) {
	buf := bytes.NewBufferString(f.Str)
	return ioutil.ReadAll(buf)
}

func TestNew(t *testing.T) {
	type args struct {
		o ConfigOptions
	}
	tests := []struct {
		name string
		args args
		want *Client
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.args.o); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_ReadCSVFiles(t *testing.T) {
	type fields struct {
		FinanceDir string
		Banks      map[string]cfg.Bank
	}
	tests := []struct {
		name    string
		fields  fields
		want    []*models.Transaction
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				FinanceDir: tt.fields.FinanceDir,
				Banks:      tt.fields.Banks,
			}
			got, err := c.ReadCSVFiles()
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadCSVFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadCSVFiles() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_readWellsFargoCSV(t *testing.T) {
	config, err := cfg.ReadConfig("../../config/config.json")
	if err != nil {
		t.Fatalf(err.Error())
	}

	j, err := os.ReadFile(JSONDIR + "wells_fargo_transactions.json")
	checkTestingError(t, err)
	var trans []*models.Transaction
	err = json.Unmarshal(j, &trans)
	checkTestingError(t, err)

	csvFile := `"date","amount","dummy1","dummy2","name"
"07/17/2023","-14.01","*","","AMERICAN STRATEG 8662748765 1A55EB86FCFE ROBERT CALLAHAN"
"07/17/2023","-25.00","*","110","CHECK # 110"
"07/13/2023","5086.51","*","","MSPBNA ACH TRNSFR 230712 84178376906 45469742"
"07/07/2023","-2.55","*","","BILL PAY MasterCard ON-LINE xxxxxxxxxxxx5189 ON 07-07"
"07/05/2023","-105.00","*","","SHENANDOAH VALLE UTILITY 230704 3957509 ROBERT T *CALLAHAN"
"07/03/2023","-75.53","*","","GLO FIBER BILLPAY 230702 GLO FIBER ROBERT CALLAHAN"
"07/03/2023","-1350.00","*","","Olive Tree Prope WEB PMTS 070323 YP2MT7 Robert Callahan"
`
	fake := FakeReadFiler{Str: csvFile}
	myReadFile = fake.ReadFile

	type fields struct {
		FinanceDir string
		Banks      map[string]cfg.Bank
	}
	tests := []struct {
		name    string
		fields  fields
		want    []*models.Transaction
		wantErr bool
	}{
		{
			name:    "Test read Wells Fargo CSV file",
			fields:  fields{FinanceDir: FinanceDir, Banks: config.Banks},
			want:    trans,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				FinanceDir: tt.fields.FinanceDir,
				Banks:      tt.fields.Banks,
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					if got, _ := c.readWellsFargoCSV(); !reflect.DeepEqual(got, tt.want) {
						t.Errorf("readWellsFargoCSV() = %v, want %v", got, tt.want)
					}
				})
			}
		})
	}
}

func TestClient_readFidelityCSV(t *testing.T) {
	type fields struct {
		FinanceDir string
		Banks      map[string]cfg.Bank
	}
	tests := []struct {
		name    string
		fields  fields
		want    []*models.Transaction
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				FinanceDir: tt.fields.FinanceDir,
				Banks:      tt.fields.Banks,
			}
			got, err := c.readFidelityCSV()
			if (err != nil) != tt.wantErr {
				t.Errorf("readFidelityCSV() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readFidelityCSV() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_readChaseCSV(t *testing.T) {
	type fields struct {
		FinanceDir string
		Banks      map[string]cfg.Bank
	}
	tests := []struct {
		name    string
		fields  fields
		want    []*models.Transaction
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				FinanceDir: tt.fields.FinanceDir,
				Banks:      tt.fields.Banks,
			}
			got, err := c.readChaseCSV()
			if (err != nil) != tt.wantErr {
				t.Errorf("readChaseCSV() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readChaseCSV() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_readBankOfAmericaCSV(t *testing.T) {
	type fields struct {
		FinanceDir string
		Banks      map[string]cfg.Bank
	}
	tests := []struct {
		name    string
		fields  fields
		want    []*models.Transaction
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				FinanceDir: tt.fields.FinanceDir,
				Banks:      tt.fields.Banks,
			}
			got, err := c.readBankOfAmericaCSV()
			if (err != nil) != tt.wantErr {
				t.Errorf("readBankOfAmericaCSV() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readBankOfAmericaCSV() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_readWellsFargoCSVRows(t *testing.T) {
	type args struct {
		csvFile string
		bankId  string
	}
	tests := []struct {
		name    string
		args    args
		want    []*models.Transaction
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readWellsFargoCSVRows(tt.args.csvFile, tt.args.bankId)
			if (err != nil) != tt.wantErr {
				t.Errorf("readWellsFargoCSVRows() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readWellsFargoCSVRows() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_processWellsFargoData(t *testing.T) {
	type args struct {
		wellsFargo []*WellsFargo
		bankId     string
	}
	tests := []struct {
		name string
		args args
		want []*models.Transaction
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := processWellsFargoData(tt.args.wellsFargo, tt.args.bankId); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("processWellsFargoData() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_processCheck(t *testing.T) {
	type args struct {
		name string
		t    *models.Transaction
	}
	tests := []struct {
		name string
		args args
		want *models.Transaction
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := processCheck(tt.args.name, tt.args.t); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("processCheck() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_doWellsFargoHeadsExist(t *testing.T) {
	type args struct {
		fileBytes []byte
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := doWellsFargoHeadsExist(tt.args.fileBytes); got != tt.want {
				t.Errorf("doWellsFargoHeadsExist() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_addWellsFargoHeads(t *testing.T) {
	type args struct {
		fileName  string
		fileBytes []byte
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := addWellsFargoHeads(tt.args.fileName, tt.args.fileBytes); (err != nil) != tt.wantErr {
				t.Errorf("addWellsFargoHeads() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_readFidelityCSVRows(t *testing.T) {
	type args struct {
		csvFile string
		bankId  string
	}
	tests := []struct {
		name    string
		args    args
		want    []*models.Transaction
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readFidelityCSVRows(tt.args.csvFile, tt.args.bankId)
			if (err != nil) != tt.wantErr {
				t.Errorf("readFidelityCSVRows() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readFidelityCSVRows() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_processFidelityData(t *testing.T) {
	type args struct {
		fidelity []*FidelityVisa
		bankId   string
	}
	tests := []struct {
		name string
		args args
		want []*models.Transaction
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := processFidelityData(tt.args.fidelity, tt.args.bankId); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("processFidelityData() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_readChaseCSVRows(t *testing.T) {
	type args struct {
		csvFile string
		bankId  string
	}
	tests := []struct {
		name    string
		args    args
		want    []*models.Transaction
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readChaseCSVRows(tt.args.csvFile, tt.args.bankId)
			if (err != nil) != tt.wantErr {
				t.Errorf("readChaseCSVRows() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readChaseCSVRows() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_processChaseData(t *testing.T) {
	type args struct {
		chase  []*ChaseVisa
		bankId string
	}
	tests := []struct {
		name string
		args args
		want []*models.Transaction
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := processChaseData(tt.args.chase, tt.args.bankId); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("processChaseData() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_readBankOfAmericaCSVRows(t *testing.T) {
	type args struct {
		csvFile string
		bankId  string
	}
	tests := []struct {
		name    string
		args    args
		want    []*models.Transaction
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readBankOfAmericaCSVRows(tt.args.csvFile, tt.args.bankId)
			if (err != nil) != tt.wantErr {
				t.Errorf("readBankOfAmericaCSVRows() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readBankOfAmericaCSVRows() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_processBankOfAmericaData(t *testing.T) {
	type args struct {
		boa    []*BankOfAmerica
		bankId string
	}
	tests := []struct {
		name string
		args args
		want []*models.Transaction
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := processBankOfAmericaData(tt.args.boa, tt.args.bankId); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("processBankOfAmericaData() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_readDateValue(t *testing.T) {
	type args struct {
		date string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test '01/02/2023' date values",
			args: args{date: "01/02/2023"},
			want: "01/02/23",
		},
		{
			name: "Test '2023-01-02' date values",
			args: args{date: "2023-01-02"},
			want: "01/02/23",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := readDateValue(tt.args.date); got != tt.want {
				t.Errorf("readDateValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func checkTestingError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("error: %s\n", err.Error())
	}
}
